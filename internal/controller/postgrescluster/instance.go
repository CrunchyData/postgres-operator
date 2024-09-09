// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/config"
	"github.com/crunchydata/postgres-operator/internal/controller/runtime"
	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/pgbackrest"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// Instance represents a single PostgreSQL instance of a PostgresCluster.
type Instance struct {
	Name   string
	Pods   []*corev1.Pod
	Runner *appsv1.StatefulSet
	Spec   *v1beta1.PostgresInstanceSetSpec
}

// IsAvailable is used to choose which instances to redeploy during rolling
// update. It combines information from metadata and status similar to the
// notion of "available" in corev1.Deployment and "healthy" in appsv1.StatefulSet.
func (i Instance) IsAvailable() (available bool, known bool) {
	// StatefulSet will have its own notion of Available in the future.
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#minimum-ready-seconds

	terminating, knownTerminating := i.IsTerminating()
	ready, knownReady := i.IsReady()

	return ready && !terminating, knownReady && knownTerminating
}

// IsPrimary returns whether or not this instance is the Patroni leader.
func (i Instance) IsPrimary() (primary bool, known bool) {
	if len(i.Pods) != 1 {
		return false, false
	}

	return i.Pods[0].Labels[naming.LabelRole] == naming.RolePatroniLeader, true
}

// IsReady returns whether or not this instance is ready to receive PostgreSQL
// connections.
func (i Instance) IsReady() (ready bool, known bool) {
	if len(i.Pods) == 1 {
		for _, condition := range i.Pods[0].Status.Conditions {
			if condition.Type == corev1.PodReady {
				return condition.Status == corev1.ConditionTrue, true
			}
		}
	}

	return false, false
}

// IsRunning returns whether or not container is running.
func (i Instance) IsRunning(container string) (running bool, known bool) {
	if len(i.Pods) == 1 {
		for _, status := range i.Pods[0].Status.ContainerStatuses {
			if status.Name == container {
				return status.State.Running != nil, true
			}
		}
		for _, status := range i.Pods[0].Status.InitContainerStatuses {
			if status.Name == container {
				return status.State.Running != nil, true
			}
		}
	}

	return false, false
}

// IsTerminating returns whether or not this instance is in the process of not
// running.
func (i Instance) IsTerminating() (terminating bool, known bool) {
	if len(i.Pods) != 1 {
		return false, false
	}

	// k8s.io/kubernetes/pkg/registry/core/pod.Strategy implements
	// k8s.io/apiserver/pkg/registry/rest.RESTGracefulDeleteStrategy so that it
	// can set DeletionTimestamp to corev1.PodSpec.TerminationGracePeriodSeconds
	// in the future.
	// - https://releases.k8s.io/v1.21.0/pkg/registry/core/pod/strategy.go#L135
	// - https://releases.k8s.io/v1.21.0/staging/src/k8s.io/apiserver/pkg/registry/rest/delete.go
	return i.Pods[0].DeletionTimestamp != nil, true
}

// IsWritable returns whether or not a PostgreSQL connection could write to its
// database.
func (i Instance) IsWritable() (writable, known bool) {
	if len(i.Pods) != 1 {
		return false, false
	}

	member := i.Pods[0].Annotations["status"]
	role := strings.Index(member, `"role":`)

	if role < 0 {
		return false, false
	}

	// TODO(cbandy): Update this to consider when Patroni is paused.

	return strings.HasPrefix(member[role:], `"role":"master"`), true
}

// PodMatchesPodTemplate returns whether or not the Pod for this instance
// matches its specified PodTemplate. When it does not match, the Pod needs to
// be redeployed.
func (i Instance) PodMatchesPodTemplate() (matches bool, known bool) {
	if i.Runner == nil || len(i.Pods) != 1 {
		return false, false
	}

	if i.Runner.Status.ObservedGeneration != i.Runner.Generation {
		return false, false
	}

	// When the Status is up-to-date, compare the revision of the Pod to that
	// of the PodTemplate.
	podRevision := i.Pods[0].Labels[appsv1.StatefulSetRevisionLabel]
	return podRevision == i.Runner.Status.UpdateRevision, true
}

// instanceSorter implements sort.Interface for some instance comparison.
type instanceSorter struct {
	instances []*Instance
	less      func(i, j *Instance) bool
}

func (s *instanceSorter) Len() int {
	return len(s.instances)
}
func (s *instanceSorter) Less(i, j int) bool {
	return s.less(s.instances[i], s.instances[j])
}
func (s *instanceSorter) Swap(i, j int) {
	s.instances[i], s.instances[j] = s.instances[j], s.instances[i]
}

// byPriority returns a sort.Interface that sorts instances by how much we want
// each to keep running. The primary instance, when known, is always the highest
// priority. Two instances with otherwise-identical priority are ranked by Name.
func byPriority(instances []*Instance) sort.Interface {
	return &instanceSorter{instances: instances, less: func(a, b *Instance) bool {
		// The primary instance is the highest priority.
		if primary, known := a.IsPrimary(); known && primary {
			return false
		}
		if primary, known := b.IsPrimary(); known && primary {
			return true
		}

		// An available instance is a higher priority than not.
		if available, known := a.IsAvailable(); known && available {
			return false
		}
		if available, known := b.IsAvailable(); known && available {
			return true
		}

		return a.Name < b.Name
	}}
}

// observedInstances represents all the PostgreSQL instances of a single PostgresCluster.
type observedInstances struct {
	byName     map[string]*Instance
	bySet      map[string][]*Instance
	forCluster []*Instance
	setNames   sets.Set[string]
}

// newObservedInstances builds an observedInstances from Kubernetes API objects.
func newObservedInstances(
	cluster *v1beta1.PostgresCluster,
	runners []appsv1.StatefulSet,
	pods []corev1.Pod,
) *observedInstances {
	observed := observedInstances{
		byName:   make(map[string]*Instance),
		bySet:    make(map[string][]*Instance),
		setNames: make(sets.Set[string]),
	}

	sets := make(map[string]*v1beta1.PostgresInstanceSetSpec)
	for i := range cluster.Spec.InstanceSets {
		name := cluster.Spec.InstanceSets[i].Name
		sets[name] = &cluster.Spec.InstanceSets[i]
		observed.setNames.Insert(name)
	}
	for i := range runners {
		ri := runners[i].Name
		rs := runners[i].Labels[naming.LabelInstanceSet]

		instance := &Instance{
			Name:   ri,
			Runner: &runners[i],
			Spec:   sets[rs],
		}

		observed.byName[ri] = instance
		observed.bySet[rs] = append(observed.bySet[rs], instance)
		observed.forCluster = append(observed.forCluster, instance)
		observed.setNames.Insert(rs)
	}
	for i := range pods {
		pi := pods[i].Labels[naming.LabelInstance]
		ps := pods[i].Labels[naming.LabelInstanceSet]

		instance := observed.byName[pi]
		if instance == nil {
			instance = &Instance{
				Name: pi,
				Spec: sets[ps],
			}
			observed.byName[pi] = instance
			observed.bySet[ps] = append(observed.bySet[ps], instance)
			observed.forCluster = append(observed.forCluster, instance)
			observed.setNames.Insert(ps)
		}
		instance.Pods = append(instance.Pods, &pods[i])
	}

	return &observed
}

// writablePod looks at observedInstances and finds an instance that matches
// a few conditions. The instance should be non-terminating, running, and
// writable i.e. the instance with the primary. If such an instance exists, it
// is returned along with the instance pod.
func (observed *observedInstances) writablePod(container string) (*corev1.Pod, *Instance) {
	if observed == nil {
		return nil, nil
	}

	for _, instance := range observed.forCluster {
		if terminating, known := instance.IsTerminating(); terminating || !known {
			continue
		}
		if writable, known := instance.IsWritable(); !writable || !known {
			continue
		}
		running, known := instance.IsRunning(container)
		if running && known && len(instance.Pods) > 0 {
			return instance.Pods[0], instance
		}
	}

	return nil, nil
}

// +kubebuilder:rbac:groups="",resources="pods",verbs={list}
// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={list}

// observeInstances populates cluster.Status.InstanceSets with observations and
// builds an observedInstances by reading from the Kubernetes API.
func (r *Reconciler) observeInstances(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*observedInstances, error) {
	pods := &corev1.PodList{}
	runners := &appsv1.StatefulSetList{}

	autogrow := feature.Enabled(ctx, feature.AutoGrowVolumes)

	selector, err := naming.AsSelector(naming.ClusterInstances(cluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, pods,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, runners,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	observed := newObservedInstances(cluster, runners.Items, pods.Items)

	// Save desired volume size values in case the status is removed.
	// This may happen in cases where the Pod is restarted, the cluster
	// is shutdown, etc. Only save values for instances defined in the spec.
	previousDesiredRequests := make(map[string]string)
	if autogrow {
		for _, statusIS := range cluster.Status.InstanceSets {
			if statusIS.DesiredPGDataVolume != nil {
				for k, v := range statusIS.DesiredPGDataVolume {
					previousDesiredRequests[k] = v
				}
			}
		}
	}

	// Fill out status sorted by set name.
	cluster.Status.InstanceSets = cluster.Status.InstanceSets[:0]
	for _, name := range sets.List(observed.setNames) {
		status := v1beta1.PostgresInstanceSetStatus{Name: name}
		status.DesiredPGDataVolume = make(map[string]string)

		for _, instance := range observed.bySet[name] {
			status.Replicas += int32(len(instance.Pods)) //nolint:gosec

			if ready, known := instance.IsReady(); known && ready {
				status.ReadyReplicas++
			}
			if matches, known := instance.PodMatchesPodTemplate(); known && matches {
				status.UpdatedReplicas++
			}
			if autogrow {
				// Store desired pgData volume size for each instance Pod.
				// The 'suggested-pgdata-pvc-size' annotation value is stored in the PostgresCluster
				// status so that 1) it is available to the function 'reconcilePostgresDataVolume'
				// and 2) so that the value persists after Pod restart and cluster shutdown events.
				for _, pod := range instance.Pods {
					// don't set an empty status
					if pod.Annotations["suggested-pgdata-pvc-size"] != "" {
						status.DesiredPGDataVolume[instance.Name] = pod.Annotations["suggested-pgdata-pvc-size"]
					}
				}
			}
		}

		// If autogrow is enabled, get the desired volume size for each instance.
		if autogrow {
			for _, instance := range observed.bySet[name] {
				status.DesiredPGDataVolume[instance.Name] = r.storeDesiredRequest(ctx, cluster,
					name, status.DesiredPGDataVolume[instance.Name], previousDesiredRequests[instance.Name])
			}
		}

		cluster.Status.InstanceSets = append(cluster.Status.InstanceSets, status)
	}

	return observed, err
}

// storeDesiredRequest saves the appropriate request value to the PostgresCluster
// status. If the value has grown, create an Event.
func (r *Reconciler) storeDesiredRequest(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instanceSetName, desiredRequest, desiredRequestBackup string,
) string {
	var current resource.Quantity
	var previous resource.Quantity
	var err error
	log := logging.FromContext(ctx)

	// Parse the desired request from the cluster's status.
	if desiredRequest != "" {
		current, err = resource.ParseQuantity(desiredRequest)
		if err != nil {
			log.Error(err, "Unable to parse pgData volume request from status ("+
				desiredRequest+") for "+cluster.Name+"/"+instanceSetName)
			// If there was an error parsing the value, treat as unset (equivalent to zero).
			desiredRequest = ""
			current, _ = resource.ParseQuantity("")

		}
	}

	// Parse the desired request from the status backup.
	if desiredRequestBackup != "" {
		previous, err = resource.ParseQuantity(desiredRequestBackup)
		if err != nil {
			log.Error(err, "Unable to parse pgData volume request from status backup ("+
				desiredRequestBackup+") for "+cluster.Name+"/"+instanceSetName)
			// If there was an error parsing the value, treat as unset (equivalent to zero).
			desiredRequestBackup = ""
			previous, _ = resource.ParseQuantity("")

		}
	}

	// Determine if the limit is set for this instance set.
	var limitSet bool
	for _, specInstance := range cluster.Spec.InstanceSets {
		if specInstance.Name == instanceSetName {
			limitSet = !specInstance.DataVolumeClaimSpec.Resources.Limits.Storage().IsZero()
		}
	}

	if limitSet && current.Value() > previous.Value() {
		r.Recorder.Eventf(cluster, corev1.EventTypeNormal, "VolumeAutoGrow",
			"pgData volume expansion to %v requested for %s/%s.",
			current.String(), cluster.Name, instanceSetName)
	}

	// If the desired size was not observed, update with previously stored value.
	// This can happen in scenarios where the annotation on the Pod is missing
	// such as when the cluster is shutdown or a Pod is in the middle of a restart.
	if desiredRequest == "" {
		desiredRequest = desiredRequestBackup
	}

	return desiredRequest
}

// +kubebuilder:rbac:groups="",resources="pods",verbs={list}
// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={patch}

// deleteInstances gracefully stops instances of cluster to avoid failovers and
// unclean shutdowns of PostgreSQL. It returns (nil, nil) when finished.
func (r *Reconciler) deleteInstances(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*reconcile.Result, error) {
	// Find all instance pods to determine which to shutdown and in what order.
	pods := &corev1.PodList{}
	instances, err := naming.AsSelector(naming.ClusterInstances(cluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, pods,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: instances},
			))
	}
	if err != nil {
		return nil, err
	}

	if len(pods.Items) == 0 {
		// There are no instances, so there's nothing to do.
		// The caller can do what they like.
		return nil, nil
	}

	// There are some instances, so the caller should at least wait for further
	// events.
	result := reconcile.Result{}

	// stop schedules pod for deletion by scaling its controller to zero.
	stop := func(pod *corev1.Pod) error {
		instance := &unstructured.Unstructured{}
		instance.SetNamespace(cluster.Namespace)

		switch owner := metav1.GetControllerOfNoCopy(pod); {
		case owner == nil:
			return errors.Errorf("pod %q has no owner", client.ObjectKeyFromObject(pod))

		case owner.Kind == "StatefulSet":
			instance.SetAPIVersion(owner.APIVersion)
			instance.SetKind(owner.Kind)
			instance.SetName(owner.Name)

		default:
			return errors.Errorf("unexpected kind %q", owner.Kind)
		}

		// apps/v1.Deployment, apps/v1.ReplicaSet, and apps/v1.StatefulSet all
		// have a "spec.replicas" field with the same meaning.
		patch := client.RawPatch(client.Merge.Type(), []byte(`{"spec":{"replicas":0}}`))
		err := errors.WithStack(r.patch(ctx, instance, patch))

		// When the pod controller is missing, requeue rather than return an
		// error. The garbage collector will stop the pod, and it is not our
		// mistake that something else is deleting objects. Use RequeueAfter to
		// avoid being rate-limited due to a deluge of delete events.
		if err != nil {
			result = runtime.RequeueWithoutBackoff(10 * time.Second)
		}
		return client.IgnoreNotFound(err)
	}

	if len(pods.Items) == 1 {
		// There's one instance; stop it.
		return &result, stop(&pods.Items[0])
	}

	// There are multiple instances; stop the replicas. When none are found,
	// requeue to try again.

	result.Requeue = true
	for i := range pods.Items {
		role := pods.Items[i].Labels[naming.LabelRole]
		if err == nil && role == naming.RolePatroniReplica {
			err = stop(&pods.Items[i])
			result.Requeue = false
		}

		// An instance without a role label is not participating in the Patroni
		// cluster. It may be unhealthy or has not yet (re-)joined. Go ahead and
		// stop these as well.
		if err == nil && len(role) == 0 {
			err = stop(&pods.Items[i])
			result.Requeue = false
		}
	}

	return &result, err
}

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={delete,list}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={delete,list}
// +kubebuilder:rbac:groups="",resources="persistentvolumeclaims",verbs={delete,list}
// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={delete,list}

// deleteInstance will delete all resources related to a single instance
func (r *Reconciler) deleteInstance(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	instanceName string,
) error {
	gvks := []schema.GroupVersionKind{{
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "ConfigMapList",
	}, {
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "SecretList",
	}, {
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "StatefulSetList",
	}, {
		Group:   corev1.SchemeGroupVersion.Group,
		Version: corev1.SchemeGroupVersion.Version,
		Kind:    "PersistentVolumeClaimList",
	}}

	selector, err := naming.AsSelector(naming.ClusterInstance(cluster.Name, instanceName))
	for _, gvk := range gvks {
		if err == nil {
			uList := &unstructured.UnstructuredList{}
			uList.SetGroupVersionKind(gvk)

			err = errors.WithStack(
				r.Client.List(ctx, uList,
					client.InNamespace(cluster.GetNamespace()),
					client.MatchingLabelsSelector{Selector: selector},
				))

			for i := range uList.Items {
				if err == nil {
					err = errors.WithStack(client.IgnoreNotFound(
						r.deleteControlled(ctx, cluster, &uList.Items[i])))
				}
			}
		}
	}

	return err
}

// reconcileInstanceSets reconciles instance sets in the environment to match
// the current spec. This is done by scaling up or down instances where necessary
func (r *Reconciler) reconcileInstanceSets(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	clusterConfigMap *corev1.ConfigMap,
	clusterReplicationSecret *corev1.Secret,
	rootCA *pki.RootCertificateAuthority,
	clusterPodService *corev1.Service,
	instanceServiceAccount *corev1.ServiceAccount,
	instances *observedInstances,
	patroniLeaderService *corev1.Service,
	primaryCertificate *corev1.SecretProjection,
	clusterVolumes []corev1.PersistentVolumeClaim,
	exporterQueriesConfig, exporterWebConfig *corev1.ConfigMap,
	backupsSpecFound bool,
) error {

	// Go through the observed instances and check if a primary has been determined.
	// If the cluster is being shutdown and this instance is the primary, store
	// the instance name as the startup instance. If the primary can be determined
	// from the instance and the cluster is not being shutdown, clear any stored
	// startup instance values.
	for _, instance := range instances.forCluster {
		if primary, known := instance.IsPrimary(); primary && known {
			if cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown {
				cluster.Status.StartupInstance = instance.Name
				cluster.Status.StartupInstanceSet = instance.Spec.Name
			} else {
				cluster.Status.StartupInstance = ""
				cluster.Status.StartupInstanceSet = ""
			}
		}
	}

	// get the number of instance pods from the observedInstance information
	var numInstancePods int
	for i := range instances.forCluster {
		numInstancePods += len(instances.forCluster[i].Pods)
	}

	// Range over instance sets to scale up and ensure that each set has
	// at least the number of replicas defined in the spec. The set can
	// have more replicas than defined
	for i := range cluster.Spec.InstanceSets {
		set := &cluster.Spec.InstanceSets[i]
		_, err := r.scaleUpInstances(
			ctx, cluster, instances, set,
			clusterConfigMap, clusterReplicationSecret,
			rootCA, clusterPodService, instanceServiceAccount,
			patroniLeaderService, primaryCertificate,
			findAvailableInstanceNames(*set, instances, clusterVolumes),
			numInstancePods, clusterVolumes, exporterQueriesConfig, exporterWebConfig,
			backupsSpecFound,
		)

		if err == nil {
			err = r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, set)
		}
		if err != nil {
			return err
		}
	}

	// Scaledown is called on the whole cluster in order to consider all
	// instances. This is necessary because we have no way to determine
	// which instance or instance set contains the primary pod.
	err := r.scaleDownInstances(ctx, cluster, instances)
	if err != nil {
		return err
	}

	// Cleanup Instance Set resources that are no longer needed
	err = r.cleanupPodDisruptionBudgets(ctx, cluster)
	if err != nil {
		return err
	}

	// Rollout changes to instances by calling rolloutInstance.
	err = r.rolloutInstances(ctx, cluster, instances,
		func(ctx context.Context, instance *Instance) error {
			return r.rolloutInstance(ctx, cluster, instances, instance)
		})

	return err
}

// +kubebuilder:rbac:groups="policy",resources="poddisruptionbudgets",verbs={list}

// cleanupPodDisruptionBudgets removes pdbs that do not have an
// associated Instance Set
func (r *Reconciler) cleanupPodDisruptionBudgets(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
) error {
	selector, err := naming.AsSelector(naming.ClusterInstanceSets(cluster.Name))

	pdbList := &policyv1.PodDisruptionBudgetList{}
	if err == nil {
		err = r.Client.List(ctx, pdbList,
			client.InNamespace(cluster.Namespace), client.MatchingLabelsSelector{
				Selector: selector,
			})
	}

	if err == nil {
		setNames := sets.Set[string]{}
		for _, set := range cluster.Spec.InstanceSets {
			setNames.Insert(set.Name)
		}
		for i := range pdbList.Items {
			pdb := pdbList.Items[i]
			if err == nil && !setNames.Has(pdb.Labels[naming.LabelInstanceSet]) {
				err = client.IgnoreNotFound(r.deleteControlled(ctx, cluster, &pdb))
			}
		}
	}

	return client.IgnoreNotFound(err)
}

// TODO (andrewlecuyer): If relevant instance volume (PVC) information is captured for each
// Instance contained within observedInstances, this function might no longer be necessary.
// Instead, available names could be derived by looking at observed Instances that have data
// volumes, but no associated runner.

// findAvailableInstanceNames finds any instance names that are available for reuse within a
// specific instance set.  Available instance names are determined by finding any instance PVCs
// for the instance set specified that are not currently associated with an instance, and then
// returning the instance names associated with those PVC's.
func findAvailableInstanceNames(set v1beta1.PostgresInstanceSetSpec,
	observedInstances *observedInstances, clusterVolumes []corev1.PersistentVolumeClaim) []string {

	availableInstanceNames := []string{}

	// first identify any PGDATA volumes for the instance set specified
	setVolumes := []corev1.PersistentVolumeClaim{}
	for _, pvc := range clusterVolumes {
		// ignore PGDATA PVCs that are terminating
		if pvc.GetDeletionTimestamp() != nil {
			continue
		}
		pvcSet := pvc.GetLabels()[naming.LabelInstanceSet]
		pvcRole := pvc.GetLabels()[naming.LabelRole]
		if pvcRole == naming.RolePostgresData && pvcSet == set.Name {
			setVolumes = append(setVolumes, pvc)
		}
	}

	// If there is a WAL volume defined for the instance set, then a matching WAL volume
	// must also be found in order for the volumes to be reused.  Therefore, filter out
	// any available PGDATA volumes for the instance set that have no corresponding WAL
	// volumes (which means new PVCs will simply be reconciled instead).
	if set.WALVolumeClaimSpec != nil {
		setVolumesWithWAL := []corev1.PersistentVolumeClaim{}
		for _, setVol := range setVolumes {
			setVolInstance := setVol.GetLabels()[naming.LabelInstance]
			for _, pvc := range clusterVolumes {
				// ignore WAL PVCs that are terminating
				if pvc.GetDeletionTimestamp() != nil {
					continue
				}
				pvcSet := pvc.GetLabels()[naming.LabelInstanceSet]
				pvcInstance := pvc.GetLabels()[naming.LabelInstance]
				pvcRole := pvc.GetLabels()[naming.LabelRole]
				if pvcRole == naming.RolePostgresWAL && pvcSet == set.Name &&
					pvcInstance == setVolInstance {
					setVolumesWithWAL = append(setVolumesWithWAL, pvc)
				}
			}
		}
		setVolumes = setVolumesWithWAL
	}

	// Determine whether or not the PVC is associated with an existing instance within the same
	// instance set.  If not, then the instance name associated with that PVC can be be reused.
	for _, pvc := range setVolumes {
		pvcInstanceName := pvc.GetLabels()[naming.LabelInstance]
		instance := observedInstances.byName[pvcInstanceName]
		if instance == nil || instance.Runner == nil {
			availableInstanceNames = append(availableInstanceNames, pvcInstanceName)
		}
	}

	return availableInstanceNames
}

// +kubebuilder:rbac:groups="",resources="pods",verbs={delete}

// rolloutInstance redeploys the Pod of instance by deleting it. Its StatefulSet
// will recreate it according to its current PodTemplate. When instance is the
// primary of a cluster with failover, it is demoted instead.
func (r *Reconciler) rolloutInstance(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instances *observedInstances, instance *Instance,
) error {
	// The StatefulSet and number of Pods should have already been verified, but
	// check again rather than panic.
	// TODO(cbandy): The check for StatefulSet can go away if we watch Pod deletes.
	if instance.Runner == nil || len(instance.Pods) != 1 {
		return errors.Errorf(
			"unexpected instance state during rollout: %v has %v pods",
			instance.Name, len(instance.Pods))
	}

	pod := instance.Pods[0]
	exec := func(_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
		return r.PodExec(ctx, pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
	}

	primary, known := instance.IsPrimary()
	primary = primary && known

	// When the cluster has more than one instance participating in failover,
	// perform a controlled switchover to one of those instances. Patroni will
	// choose the best candidate and demote the primary. It stops PostgreSQL
	// using what it calls "graceful" mode: it takes an immediate checkpoint in
	// the background then uses "pg_ctl" to perform a "fast" shutdown when the
	// checkpoint completes.
	// - https://github.com/zalando/patroni/blob/v2.0.2/patroni/ha.py#L815
	// - https://www.postgresql.org/docs/current/sql-checkpoint.html
	//
	// NOTE(cbandy): The StatefulSet controlling this Pod reflects this change
	// in its Status and triggers another reconcile.
	if primary && len(instances.forCluster) > 1 {
		var span trace.Span
		ctx, span = r.Tracer.Start(ctx, "patroni-change-primary")
		defer span.End()

		success, err := patroni.Executor(exec).ChangePrimaryAndWait(ctx, pod.Name, "")
		if err = errors.WithStack(err); err == nil && !success {
			err = errors.New("unable to switchover")
		}

		span.RecordError(err)
		return err
	}

	// When the cluster has only one instance for failover, perform a series of
	// immediate checkpoints to increase the likelihood that a "fast" shutdown
	// will complete before the SIGKILL near TerminationGracePeriodSeconds.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#pod-termination
	if primary {
		graceSeconds := int64(corev1.DefaultTerminationGracePeriodSeconds)
		if pod.Spec.TerminationGracePeriodSeconds != nil {
			graceSeconds = *pod.Spec.TerminationGracePeriodSeconds
		}

		checkpoint := func(ctx context.Context) (time.Duration, error) {
			ctx, span := r.Tracer.Start(ctx, "postgresql-checkpoint")
			defer span.End()

			start := time.Now()
			stdout, stderr, err := postgres.Executor(exec).
				ExecInDatabasesFromQuery(ctx, `SELECT pg_catalog.current_database()`,
					`SET statement_timeout = :'timeout'; CHECKPOINT;`,
					map[string]string{
						"timeout":       fmt.Sprintf("%ds", graceSeconds),
						"ON_ERROR_STOP": "on", // Abort when any one statement fails.
						"QUIET":         "on", // Do not print successful statements to stdout.
					})
			err = errors.WithStack(err)
			elapsed := time.Since(start)

			logging.FromContext(ctx).V(1).Info("attempted checkpoint",
				"duration", elapsed, "stdout", stdout, "stderr", stderr)

			span.RecordError(err)
			return elapsed, err
		}

		duration, err := checkpoint(ctx)
		threshold := time.Duration(graceSeconds/2) * time.Second

		// The first checkpoint could be flushing up to "checkpoint_timeout"
		// or "max_wal_size" worth of data. Try once more to get a sense of
		// how long "fast" shutdown might take.
		if err == nil && duration > threshold {
			duration, err = checkpoint(ctx)
		}

		// Communicate the lack or slowness of CHECKPOINT and shutdown anyway.
		if err != nil {
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "NoCheckpoint",
				"Unable to checkpoint primary before shutdown: %v", err)
		} else if duration > threshold {
			r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "SlowCheckpoint",
				"Shutting down primary despite checkpoint taking over %v", duration)
		}
	}

	// Delete the Pod so its controlling StatefulSet will recreate it. Patroni
	// will receive a SIGTERM and use "pg_ctl" to perform a "fast" shutdown of
	// PostgreSQL without taking a checkpoint.
	// - https://github.com/zalando/patroni/blob/v2.0.2/patroni/ha.py#L1465
	//
	// NOTE(cbandy): This could return an apierrors.IsConflict() which should be
	// retried by another reconcile (not ignored).
	return errors.WithStack(
		r.Client.Delete(ctx, pod, client.Preconditions{
			UID:             &pod.UID,
			ResourceVersion: &pod.ResourceVersion,
		}))
}

// rolloutInstances compares instances to cluster and calls redeploy on those
// that need their Pod recreated. It considers the overall availability of
// cluster and minimizes Patroni failovers.
func (r *Reconciler) rolloutInstances(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	instances *observedInstances,
	redeploy func(context.Context, *Instance) error,
) error {
	var err error
	var consider []*Instance
	var numAvailable int
	var numSpecified int

	ctx, span := r.Tracer.Start(ctx, "rollout-instances")
	defer span.End()

	for _, set := range cluster.Spec.InstanceSets {
		numSpecified += int(*set.Replicas)
	}

	for _, instance := range instances.forCluster {
		// Skip instances that have no set in cluster spec. They should not be
		// redeployed and should not count toward availability.
		if instance.Spec == nil {
			continue
		}

		// Skip instances that are or might be terminating. They should not be
		// redeployed right now and cannot count toward availability.
		if terminating, known := instance.IsTerminating(); !known || terminating {
			continue
		}

		if available, known := instance.IsAvailable(); known && available {
			numAvailable++
		}

		if matches, known := instance.PodMatchesPodTemplate(); known && !matches {
			consider = append(consider, instance)
			continue
		}
	}

	const maxUnavailable = 1
	numUnavailable := numSpecified - numAvailable

	// When multiple instances need to redeploy, sort them so the lowest
	// priority instances are first.
	if len(consider) > 1 {
		sort.Sort(byPriority(consider))
	}

	span.SetAttributes(
		attribute.Int("instances", len(instances.forCluster)),
		attribute.Int("specified", numSpecified),
		attribute.Int("available", numAvailable),
		attribute.Int("considering", len(consider)),
	)

	// Redeploy instances up to the allowed maximum while "rolling over" any
	// unavailable instances.
	// - https://issue.k8s.io/67250
	for _, instance := range consider {
		if err == nil {
			if available, known := instance.IsAvailable(); known && !available {
				err = redeploy(ctx, instance)
			} else if numUnavailable < maxUnavailable {
				err = redeploy(ctx, instance)
				numUnavailable++
			}
		}
	}

	span.RecordError(err)
	return err
}

// scaleDownInstances removes extra instances from a cluster until it matches
// the spec. This function can delete the primary instance and force the
// cluster to failover under two conditions:
//   - If the instance set that contains the primary instance is removed from
//     the spec
//   - If the instance set that contains the primary instance is updated to
//     have 0 replicas
//
// If either of these conditions are met then the primary instance will be
// marked for deletion and deleted after all other instances
func (r *Reconciler) scaleDownInstances(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	observedInstances *observedInstances,
) error {

	// want defines the number of replicas we want for each instance set
	want := map[string]int{}
	for _, set := range cluster.Spec.InstanceSets {
		want[set.Name] = int(*set.Replicas)
	}

	// grab all pods for the cluster using the observed instances
	pods := []corev1.Pod{}
	for instanceIndex := range observedInstances.forCluster {
		for podIndex := range observedInstances.forCluster[instanceIndex].Pods {
			pods = append(pods, *observedInstances.forCluster[instanceIndex].Pods[podIndex])
		}
	}

	// namesToKeep defines the names of any instances that should be kept
	namesToKeep := sets.NewString()
	for _, pod := range podsToKeep(pods, want) {
		namesToKeep.Insert(pod.Labels[naming.LabelInstance])
	}

	for _, instance := range observedInstances.forCluster {
		for _, pod := range instance.Pods {
			if !namesToKeep.Has(pod.Labels[naming.LabelInstance]) {
				err := r.deleteInstance(ctx, cluster, pod.Labels[naming.LabelInstance])
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// podsToKeep takes a list of pods and a map containing
// the number of replicas we want for each instance set
// then returns a list of the pods that we want to keep
func podsToKeep(instances []corev1.Pod, want map[string]int) []corev1.Pod {

	f := func(instances []corev1.Pod, want int) []corev1.Pod {
		keep := []corev1.Pod{}

		if want > 0 {
			for _, instance := range instances {
				if instance.Labels[naming.LabelRole] == "master" {
					keep = append(keep, instance)
				}
			}
		}

		for _, instance := range instances {
			if instance.Labels[naming.LabelRole] != "master" && len(keep) < want {
				keep = append(keep, instance)
			}
		}

		return keep
	}

	keepPodList := []corev1.Pod{}
	for name, num := range want {
		list := []corev1.Pod{}
		for _, instance := range instances {
			if instance.Labels[naming.LabelInstanceSet] == name {
				list = append(list, instance)
			}
		}
		keepPodList = append(keepPodList, f(list, num)...)
	}

	return keepPodList

}

// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={list}

// scaleUpInstances updates the cluster until the number of instances matches
// the cluster spec
func (r *Reconciler) scaleUpInstances(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	observed *observedInstances,
	set *v1beta1.PostgresInstanceSetSpec,
	clusterConfigMap *corev1.ConfigMap,
	clusterReplicationSecret *corev1.Secret,
	rootCA *pki.RootCertificateAuthority,
	clusterPodService *corev1.Service,
	instanceServiceAccount *corev1.ServiceAccount,
	patroniLeaderService *corev1.Service,
	primaryCertificate *corev1.SecretProjection,
	availableInstanceNames []string,
	numInstancePods int,
	clusterVolumes []corev1.PersistentVolumeClaim,
	exporterQueriesConfig, exporterWebConfig *corev1.ConfigMap,
	backupsSpecFound bool,
) ([]*appsv1.StatefulSet, error) {
	log := logging.FromContext(ctx)

	instanceNames := sets.NewString()
	instances := []*appsv1.StatefulSet{}
	for i := range observed.bySet[set.Name] {
		oi := observed.bySet[set.Name][i]
		// an instance might not have a runner if it was deleted
		if oi.Runner != nil {
			instanceNames.Insert(oi.Name)
			instances = append(instances, oi.Runner)
		}
	}
	// While there are fewer instances than specified, generate another empty one
	// and append it.
	for len(instances) < int(*set.Replicas) {
		var span trace.Span
		ctx, span = r.Tracer.Start(ctx, "generateInstanceName")
		next := naming.GenerateInstance(cluster, set)
		// if there are any available instance names (as determined by observing any PVCs for the
		// instance set that are not currently associated with an instance, e.g. in the event the
		// instance STS was deleted), then reuse them instead of generating a new name
		if len(availableInstanceNames) > 0 {
			next.Name = availableInstanceNames[0]
			availableInstanceNames = availableInstanceNames[1:]
		} else {
			for instanceNames.Has(next.Name) {
				next = naming.GenerateInstance(cluster, set)
			}
		}
		span.End()

		instanceNames.Insert(next.Name)
		instances = append(instances, &appsv1.StatefulSet{ObjectMeta: next})
	}

	var err error
	for i := range instances {
		err = r.reconcileInstance(
			ctx, cluster, observed.byName[instances[i].Name], set,
			clusterConfigMap, clusterReplicationSecret,
			rootCA, clusterPodService, instanceServiceAccount,
			patroniLeaderService, primaryCertificate, instances[i],
			numInstancePods, clusterVolumes, exporterQueriesConfig, exporterWebConfig,
			backupsSpecFound,
		)
	}
	if err == nil {
		log.V(1).Info("reconciled instance set", "instance-set", set.Name)
	}

	return instances, err
}

// +kubebuilder:rbac:groups="apps",resources="statefulsets",verbs={create,patch}

// reconcileInstance writes instance according to spec of cluster.
// See Reconciler.reconcileInstanceSet.
func (r *Reconciler) reconcileInstance(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	observed *Instance,
	spec *v1beta1.PostgresInstanceSetSpec,
	clusterConfigMap *corev1.ConfigMap,
	clusterReplicationSecret *corev1.Secret,
	rootCA *pki.RootCertificateAuthority,
	clusterPodService *corev1.Service,
	instanceServiceAccount *corev1.ServiceAccount,
	patroniLeaderService *corev1.Service,
	primaryCertificate *corev1.SecretProjection,
	instance *appsv1.StatefulSet,
	numInstancePods int,
	clusterVolumes []corev1.PersistentVolumeClaim,
	exporterQueriesConfig, exporterWebConfig *corev1.ConfigMap,
	backupsSpecFound bool,
) error {
	log := logging.FromContext(ctx).WithValues("instance", instance.Name)
	ctx = logging.NewContext(ctx, log)

	existing := instance.DeepCopy()
	*instance = appsv1.StatefulSet{}
	instance.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("StatefulSet"))
	instance.Namespace, instance.Name = existing.Namespace, existing.Name
	err := errors.WithStack(r.setControllerReference(cluster, instance))
	if err == nil {
		generateInstanceStatefulSetIntent(ctx, cluster, spec,
			clusterPodService.Name, instanceServiceAccount.Name, instance,
			numInstancePods)
	}

	var (
		instanceConfigMap    *corev1.ConfigMap
		instanceCertificates *corev1.Secret
		postgresDataVolume   *corev1.PersistentVolumeClaim
		postgresWALVolume    *corev1.PersistentVolumeClaim
		tablespaceVolumes    []*corev1.PersistentVolumeClaim
	)

	if err == nil {
		instanceConfigMap, err = r.reconcileInstanceConfigMap(ctx, cluster, spec, instance)
	}
	if err == nil {
		instanceCertificates, err = r.reconcileInstanceCertificates(
			ctx, cluster, spec, instance, rootCA)
	}
	if err == nil {
		postgresDataVolume, err = r.reconcilePostgresDataVolume(ctx, cluster, spec, instance, clusterVolumes, nil)
	}
	if err == nil {
		postgresWALVolume, err = r.reconcilePostgresWALVolume(ctx, cluster, spec, instance, observed, clusterVolumes)
	}
	if err == nil {
		tablespaceVolumes, err = r.reconcileTablespaceVolumes(ctx, cluster, spec, instance, clusterVolumes)
	}
	if err == nil {
		postgres.InstancePod(
			ctx, cluster, spec,
			primaryCertificate, replicationCertSecretProjection(clusterReplicationSecret),
			postgresDataVolume, postgresWALVolume, tablespaceVolumes,
			&instance.Spec.Template.Spec)

		if backupsSpecFound {
			addPGBackRestToInstancePodSpec(
				ctx, cluster, instanceCertificates, &instance.Spec.Template.Spec)
		}

		err = patroni.InstancePod(
			ctx, cluster, clusterConfigMap, clusterPodService, patroniLeaderService,
			spec, instanceCertificates, instanceConfigMap, &instance.Spec.Template)
	}

	// Add pgMonitor resources to the instance Pod spec
	if err == nil {
		err = addPGMonitorToInstancePodSpec(ctx, cluster, &instance.Spec.Template, exporterQueriesConfig, exporterWebConfig)
	}

	// add nss_wrapper init container and add nss_wrapper env vars to the database and pgbackrest
	// containers
	if err == nil {
		addNSSWrapper(
			config.PostgresContainerImage(cluster),
			cluster.Spec.ImagePullPolicy,
			&instance.Spec.Template)

	}
	// add an emptyDir volume to the PodTemplateSpec and an associated '/tmp' volume mount to
	// all containers included within that spec
	if err == nil {
		addTMPEmptyDir(&instance.Spec.Template)
	}

	// mount shared memory to the Postgres instance
	if err == nil {
		addDevSHM(&instance.Spec.Template)
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, instance))
	}
	if err == nil {
		log.V(1).Info("reconciled instance", "instance", instance.Name)
	}

	return err
}

func generateInstanceStatefulSetIntent(_ context.Context,
	cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec,
	clusterPodServiceName string,
	instanceServiceAccountName string,
	sts *appsv1.StatefulSet,
	numInstancePods int,
) {
	sts.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		spec.Metadata.GetAnnotationsOrNil())
	sts.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
			naming.LabelInstance:    sts.Name,
			naming.LabelData:        naming.DataPostgres,
		})
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
			naming.LabelInstance:    sts.Name,
		},
	}
	sts.Spec.Template.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		spec.Metadata.GetAnnotationsOrNil(),
	)
	sts.Spec.Template.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
			naming.LabelInstance:    sts.Name,
			naming.LabelData:        naming.DataPostgres,
		})

	// Don't clutter the namespace with extra ControllerRevisions.
	// The "controller-revision-hash" label still exists on the Pod.
	sts.Spec.RevisionHistoryLimit = initialize.Int32(0)

	// Give the Pod a stable DNS record based on its name.
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#stable-network-id
	// - https://docs.k8s.io/concepts/services-networking/dns-pod-service/#pods
	sts.Spec.ServiceName = clusterPodServiceName

	// Disable StatefulSet's "RollingUpdate" strategy. The rolloutInstances
	// method considers Pods across the entire PostgresCluster and deletes
	// them to trigger updates.
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#on-delete
	sts.Spec.UpdateStrategy.Type = appsv1.OnDeleteStatefulSetStrategyType

	// Use scheduling constraints from the cluster spec.
	sts.Spec.Template.Spec.Affinity = spec.Affinity
	sts.Spec.Template.Spec.Tolerations = spec.Tolerations
	sts.Spec.Template.Spec.TopologySpreadConstraints = spec.TopologySpreadConstraints
	if spec.PriorityClassName != nil {
		sts.Spec.Template.Spec.PriorityClassName = *spec.PriorityClassName
	}

	// if default pod scheduling is not explicitly disabled, add the default
	// pod topology spread constraints
	if cluster.Spec.DisableDefaultPodScheduling == nil ||
		(cluster.Spec.DisableDefaultPodScheduling != nil &&
			!*cluster.Spec.DisableDefaultPodScheduling) {
		sts.Spec.Template.Spec.TopologySpreadConstraints = append(
			sts.Spec.Template.Spec.TopologySpreadConstraints,
			defaultTopologySpreadConstraints(
				naming.ClusterDataForPostgresAndPGBackRest(cluster.Name),
			)...)
	}

	// Though we use a StatefulSet to keep an instance running, we only ever
	// want one Pod from it. This means that Replicas should only ever be
	// 1, the default case for a running cluster, or 0, if the existing replicas
	// value is set to 0 due to being 'shutdown'.
	// The logic below is designed to make sure that the primary/leader instance
	// is always the first to startup and the last to shutdown.
	if cluster.Status.StartupInstance == "" {
		// there is no designated startup instance; all instances should run.
		sts.Spec.Replicas = initialize.Int32(1)
	} else if cluster.Status.StartupInstance != sts.Name {
		// there is a startup instance defined, but not this instance; do not run.
		sts.Spec.Replicas = initialize.Int32(0)
	} else if cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown &&
		numInstancePods <= 1 {
		// this is the last instance of the shutdown sequence; do not run.
		sts.Spec.Replicas = initialize.Int32(0)
	} else {
		// this is the designated instance, but
		// - others are still running during shutdown, or
		// - it is time to startup.
		sts.Spec.Replicas = initialize.Int32(1)
	}

	// Restart containers any time they stop, die, are killed, etc.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#restart-policy
	sts.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways

	// ShareProcessNamespace makes Kubernetes' pause process PID 1 and lets
	// containers see each other's processes.
	// - https://docs.k8s.io/tasks/configure-pod-container/share-process-namespace/
	sts.Spec.Template.Spec.ShareProcessNamespace = initialize.Bool(true)

	// Patroni calls the Kubernetes API and pgBackRest may interact with a cloud
	// storage provider. Use the instance ServiceAccount and automatically mount
	// its Kubernetes credentials.
	// - https://cloud.google.com/kubernetes-engine/docs/concepts/workload-identity
	// - https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html
	sts.Spec.Template.Spec.ServiceAccountName = instanceServiceAccountName

	// Disable environment variables for services other than the Kubernetes API.
	// - https://docs.k8s.io/concepts/services-networking/connect-applications-service/#accessing-the-service
	// - https://releases.k8s.io/v1.23.0/pkg/kubelet/kubelet_pods.go#L553-L563
	sts.Spec.Template.Spec.EnableServiceLinks = initialize.Bool(false)

	sts.Spec.Template.Spec.SecurityContext = postgres.PodSecurityContext(cluster)

	// Set the image pull secrets, if any exist.
	// This is set here rather than using the service account due to the lack
	// of propagation to existing pods when the CRD is updated:
	// https://github.com/kubernetes/kubernetes/issues/88456
	sts.Spec.Template.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets
}

// addPGBackRestToInstancePodSpec adds pgBackRest configurations and sidecars
// to the PodSpec.
func addPGBackRestToInstancePodSpec(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	instanceCertificates *corev1.Secret, instancePod *corev1.PodSpec,
) {
	pgbackrest.AddServerToInstancePod(ctx, cluster, instancePod,
		instanceCertificates.Name)

	pgbackrest.AddConfigToInstancePod(cluster, instancePod)
}

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={create,patch}

// reconcileInstanceConfigMap writes the ConfigMap that contains generated
// files (etc) that apply to instance of cluster.
func (r *Reconciler) reconcileInstanceConfigMap(
	ctx context.Context, cluster *v1beta1.PostgresCluster, spec *v1beta1.PostgresInstanceSetSpec,
	instance *appsv1.StatefulSet,
) (*corev1.ConfigMap, error) {
	instanceConfigMap := &corev1.ConfigMap{ObjectMeta: naming.InstanceConfigMap(instance)}
	instanceConfigMap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	// TODO(cbandy): Instance StatefulSet as owner?
	err := errors.WithStack(r.setControllerReference(cluster, instanceConfigMap))

	instanceConfigMap.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		spec.Metadata.GetAnnotationsOrNil())
	instanceConfigMap.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
			naming.LabelInstance:    instance.Name,
		})

	if err == nil {
		err = patroni.InstanceConfigMap(ctx, cluster, spec, instanceConfigMap)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, instanceConfigMap))
	}

	return instanceConfigMap, err
}

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,patch}

// reconcileInstanceCertificates writes the Secret that contains certificates
// and private keys for instance of cluster.
func (r *Reconciler) reconcileInstanceCertificates(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
	root *pki.RootCertificateAuthority,
) (*corev1.Secret, error) {
	existing := &corev1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	instanceCerts := &corev1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}
	instanceCerts.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))

	// TODO(cbandy): Instance StatefulSet as owner?
	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, instanceCerts))
	}

	instanceCerts.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		spec.Metadata.GetAnnotationsOrNil())
	instanceCerts.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
			naming.LabelInstance:    instance.Name,
		})

	// This secret is holding certificates, but the "kubernetes.io/tls" type
	// expects an *unencrypted* private key. We're also adding other values and
	// other formats, so indicate that with the "Opaque" type.
	// - https://docs.k8s.io/concepts/configuration/secret/#secret-types
	instanceCerts.Type = corev1.SecretTypeOpaque
	instanceCerts.Data = make(map[string][]byte)

	var leafCert *pki.LeafCertificate

	if err == nil {
		leafCert, err = r.instanceCertificate(ctx, instance, existing, instanceCerts, root)
	}
	if err == nil {
		err = patroni.InstanceCertificates(ctx,
			root.Certificate, leafCert.Certificate,
			leafCert.PrivateKey, instanceCerts)
	}
	if err == nil {
		err = pgbackrest.InstanceCertificates(ctx, cluster,
			root.Certificate, leafCert.Certificate, leafCert.PrivateKey,
			instanceCerts)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, instanceCerts))
	}

	return instanceCerts, err
}

// +kubebuilder:rbac:groups="policy",resources="poddisruptionbudgets",verbs={create,patch,get,delete}

// reconcileInstanceSetPodDisruptionBudget creates a PDB for an instance set. A
// PDB will be created when the minAvailable is determined to be greater than 0.
// MinAvailable can be defined in the spec or a default value will be set based
// on the number of replicas in the instance set.
func (r *Reconciler) reconcileInstanceSetPodDisruptionBudget(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec,
) error {
	if spec.Replicas == nil {
		// Replicas should always have a value because of defaults in the spec
		return errors.New("Replicas should be defined")
	}
	minAvailable := getMinAvailable(spec.MinAvailable, *spec.Replicas)

	meta := naming.InstanceSet(cluster, spec)
	meta.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
		})
	meta.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil(),
		spec.Metadata.GetAnnotationsOrNil())

	selector := naming.ClusterInstanceSet(cluster.Name, spec.Name)
	pdb, err := r.generatePodDisruptionBudget(cluster, meta, minAvailable, selector)

	// If 'minAvailable' is set to '0', we will not reconcile the PDB. If one
	// already exists, we will remove it.
	var scaled int
	if err == nil {
		scaled, err = intstr.GetScaledValueFromIntOrPercent(minAvailable, int(*spec.Replicas), true)
	}
	if err == nil && scaled <= 0 {
		err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(pdb), pdb))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, pdb))
		}
		return client.IgnoreNotFound(err)
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, pdb))
	}
	return err
}
