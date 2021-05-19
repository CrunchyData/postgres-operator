/*
 Copyright 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

package postgrescluster

import (
	"context"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/pkg/errors"
	attributes "go.opentelemetry.io/otel/label"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

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
	setNames   sets.String
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
		setNames: make(sets.String),
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

// +kubebuilder:rbac:groups="",resources=pods,verbs=list
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=list

// observeInstances populates cluster.Status.InstanceSets with observations and
// builds an observedInstances by reading from the Kubernetes API.
func (r *Reconciler) observeInstances(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*observedInstances, error) {
	pods := &v1.PodList{}
	runners := &appsv1.StatefulSetList{}

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

	// Fill out status sorted by set name.
	cluster.Status.InstanceSets = cluster.Status.InstanceSets[:0]
	for _, name := range observed.setNames.List() {
		status := v1beta1.PostgresInstanceSetStatus{Name: name}

		for _, instance := range observed.bySet[name] {
			if ready, known := instance.IsReady(); known && ready {
				status.ReadyReplicas++
			}
			if terminating, known := instance.IsTerminating(); known && !terminating {
				status.Replicas++

				if matches, known := instance.PodMatchesPodTemplate(); known && matches {
					status.UpdatedReplicas++
				}
			}
		}

		cluster.Status.InstanceSets = append(cluster.Status.InstanceSets, status)
	}

	return observed, err
}

// +kubebuilder:rbac:groups="",resources=pods,verbs=list
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=patch

// deleteInstances gracefully stops instances of cluster to avoid failovers and
// unclean shutdowns of PostgreSQL. It returns (nil, nil) when finished.
func (r *Reconciler) deleteInstances(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*reconcile.Result, error) {
	// Find all instance pods to determine which to shutdown and in what order.
	pods := &v1.PodList{}
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
	stop := func(pod *v1.Pod) error {
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
			result.RequeueAfter = 10 * time.Second
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

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=delete;list
// +kubebuilder:rbac:groups="",resources=secrets,verbs=delete;list
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=delete;list
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=delete;list

// deleteInstance will delete all resources related to a single instance
func (r *Reconciler) deleteInstance(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	instanceName string,
) error {
	gvks := []schema.GroupVersionKind{{
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
		Kind:    "ConfigMapList",
	}, {
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
		Kind:    "SecretList",
	}, {
		Group:   appsv1.SchemeGroupVersion.Group,
		Version: appsv1.SchemeGroupVersion.Version,
		Kind:    "StatefulSetList",
	}, {
		Group:   v1.SchemeGroupVersion.Group,
		Version: v1.SchemeGroupVersion.Version,
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
	clusterConfigMap *v1.ConfigMap,
	clusterReplicationSecret *v1.Secret,
	rootCA *pki.RootCertificateAuthority,
	clusterPodService *v1.Service,
	instanceServiceAccount *v1.ServiceAccount,
	instances *observedInstances,
	patroniLeaderService *v1.Service,
	primaryCertificate *v1.SecretProjection,
) error {

	// Range over instance sets to scale up and ensure that each set has
	// at least the number of replicas defined in the spec. The set can
	// have more replicas than defined
	for i := range cluster.Spec.InstanceSets {
		_, err := r.scaleUpInstances(
			ctx, cluster, &cluster.Spec.InstanceSets[i],
			clusterConfigMap, clusterReplicationSecret,
			rootCA, clusterPodService, instanceServiceAccount,
			patroniLeaderService, primaryCertificate)
		if err != nil {
			return err
		}
	}

	// Scaledown is called on the whole cluster in order to consider all
	// instances. This is necessary because we have no way to determine
	// which instance or instance set contains the primary pod.
	err := r.scaleDownInstances(ctx, cluster)
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

// +kubebuilder:rbac:groups="",resources=pods,verbs=delete

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
		return r.PodExec(pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
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
				ExecInDatabasesFromQuery(ctx, `SELECT current_database()`,
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
			r.Recorder.Eventf(cluster, v1.EventTypeWarning, "NoCheckpoint",
				"Unable to checkpoint primary before shutdown: %v", err)
		} else if duration > threshold {
			r.Recorder.Eventf(cluster, v1.EventTypeWarning, "SlowCheckpoint",
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
		attributes.Int("instances", len(instances.forCluster)),
		attributes.Int("specified", numSpecified),
		attributes.Int("available", numAvailable),
		attributes.Int("considering", len(consider)),
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
// - If the instance set that contains the primary instance is removed from
//   the spec
// - If the instance set that contains the primary instance is updated to
//   have 0 replicas
// If either of these conditions are met then the primary instance will be
// marked for deletion and deleted after all other instances
func (r *Reconciler) scaleDownInstances(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
) error {
	pods := &v1.PodList{}
	selector, err := naming.AsSelector(naming.ClusterInstances(cluster.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, pods,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}
	if err != nil {
		return err
	}

	want := map[string]int{}
	for _, set := range cluster.Spec.InstanceSets {
		want[set.Name] = int(*set.Replicas)
	}

	namesToKeep := sets.NewString()
	for _, pod := range podsToKeep(pods.Items, want) {
		namesToKeep.Insert(pod.Labels[naming.LabelInstance])
	}

	for _, pod := range pods.Items {
		if !namesToKeep.Has(pod.Labels[naming.LabelInstance]) {
			err := r.deleteInstance(ctx, cluster, pod.Labels[naming.LabelInstance])
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// podsToKeep takes a list of pods and a map containing
// the number of replicas we want for each instance set
// then returns a list of the pods that we want to keep
func podsToKeep(instances []v1.Pod, want map[string]int) []v1.Pod {

	f := func(instances []v1.Pod, want int) []v1.Pod {
		keep := []v1.Pod{}

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

	keepPodList := []v1.Pod{}
	for name, num := range want {
		list := []v1.Pod{}
		for _, instance := range instances {
			if instance.Labels[naming.LabelInstanceSet] == name {
				list = append(list, instance)
			}
		}
		keepPodList = append(keepPodList, f(list, num)...)
	}

	return keepPodList

}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=list

// scaleUpInstances updates the cluster until the number of instances matches
// the cluster spec
func (r *Reconciler) scaleUpInstances(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	set *v1beta1.PostgresInstanceSetSpec,
	clusterConfigMap *v1.ConfigMap,
	clusterReplicationSecret *v1.Secret,
	rootCA *pki.RootCertificateAuthority,
	clusterPodService *v1.Service,
	instanceServiceAccount *v1.ServiceAccount,
	patroniLeaderService *v1.Service,
	primaryCertificate *v1.SecretProjection,
) (*appsv1.StatefulSetList, error) {
	log := logging.FromContext(ctx)

	instances := &appsv1.StatefulSetList{}
	selector, err := naming.AsSelector(naming.ClusterInstanceSet(cluster.Name, set.Name))
	if err == nil {
		err = errors.WithStack(
			r.Client.List(ctx, instances,
				client.InNamespace(cluster.Namespace),
				client.MatchingLabelsSelector{Selector: selector},
			))
	}

	instanceNames := sets.NewString()
	for i := range instances.Items {
		instanceNames.Insert(instances.Items[i].Name)
	}

	// While there are fewer instances than specified, generate another empty one
	// and append it.
	for err == nil && len(instances.Items) < int(*set.Replicas) {
		var span trace.Span
		ctx, span = r.Tracer.Start(ctx, "generateInstanceName")
		next := naming.GenerateInstance(cluster, set)
		for instanceNames.Has(next.Name) {
			next = naming.GenerateInstance(cluster, set)
		}
		span.End()

		instanceNames.Insert(next.Name)
		instances.Items = append(instances.Items, appsv1.StatefulSet{ObjectMeta: next})
	}
	for i := range instances.Items {
		if err == nil {
			err = r.reconcileInstance(
				ctx, cluster, set, clusterConfigMap, clusterReplicationSecret,
				rootCA, clusterPodService, instanceServiceAccount,
				patroniLeaderService, primaryCertificate, &instances.Items[i],
			)
		}
	}
	if err == nil {
		log.V(1).Info("reconciled instance set", "instance-set", set.Name)
	}

	return instances, err
}

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=create;patch

// reconcileInstance writes instance according to spec of cluster.
// See Reconciler.reconcileInstanceSet.
func (r *Reconciler) reconcileInstance(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec,
	clusterConfigMap *v1.ConfigMap,
	clusterReplicationSecret *v1.Secret,
	rootCA *pki.RootCertificateAuthority,
	clusterPodService *v1.Service,
	instanceServiceAccount *v1.ServiceAccount,
	patroniLeaderService *v1.Service,
	primaryCertificate *v1.SecretProjection,
	instance *appsv1.StatefulSet,
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
			clusterPodService.Name, instanceServiceAccount.Name,
			existing.Spec.Replicas, instance)
	}

	var (
		instanceConfigMap    *v1.ConfigMap
		instanceCertificates *v1.Secret
		postgresDataVolume   *corev1.PersistentVolumeClaim
	)

	if err == nil {
		instanceConfigMap, err = r.reconcileInstanceConfigMap(ctx, cluster, spec, instance)
	}
	if err == nil {
		instanceCertificates, err = r.reconcileInstanceCertificates(
			ctx, cluster, spec, instance, rootCA)
	}
	if err == nil {
		postgresDataVolume, err = r.reconcilePostgresDataVolume(ctx, cluster, spec, instance)
	}
	if err == nil {
		postgres.InstancePod(
			ctx, cluster, postgresDataVolume, spec, &instance.Spec.Template.Spec)

		err = patroni.InstancePod(
			ctx, cluster, clusterConfigMap, clusterPodService, patroniLeaderService,
			instanceCertificates, instanceConfigMap, &instance.Spec.Template)
	}

	// Add pgBackRest containers, volumes, etc. to the instance Pod spec
	if err == nil {
		err = addPGBackRestToInstancePodSpec(cluster, &instance.Spec.Template, instance)
	}

	// add the container for the initial copy of the mounted replication client
	// certificate files to the /tmp directory and set the proper file permissions
	postgres.InitCopyReplicationTLS(cluster, &instance.Spec.Template)

	// add the cluster certificate secret volume to the pod to enable Postgres TLS connections
	if err == nil {
		err = errors.WithStack(postgres.AddCertVolumeToPod(cluster, &instance.Spec.Template,
			naming.ContainerClientCertInit, naming.ContainerDatabase, naming.ContainerClientCertCopy,
			primaryCertificate, replicationCertSecretProjection(clusterReplicationSecret)))
	}
	// add nss_wrapper init container and add nss_wrapper env vars to the database and pgbackrest
	// containers
	if err == nil {
		addNSSWrapper(cluster.Spec.Image, &instance.Spec.Template)
	}
	// add an emptyDir volume to the PodTemplateSpec and an associated '/tmp' volume mount to
	// all containers included within that spec
	if err == nil {
		addTMPEmptyDir(&instance.Spec.Template)
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, instance))
	}
	if err == nil {
		log.V(1).Info("reconciled instance", "instance", instance.Name)
	}

	return err
}

func generateInstanceStatefulSetIntent(ctx context.Context,
	cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec,
	clusterPodServiceName string,
	instanceServiceAccountName string,
	existingReplicas *int32,
	sts *appsv1.StatefulSet,
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

	// Match the existing replica count, if any.
	if existingReplicas != nil {
		sts.Spec.Replicas = initialize.Int32(*existingReplicas)
	} else {
		sts.Spec.Replicas = initialize.Int32(1) // TODO(cbandy): start at zero, maybe
	}

	// Though we use a StatefulSet to keep an instance running, we only ever
	// want one Pod from it.
	if *sts.Spec.Replicas > 1 {
		*sts.Spec.Replicas = 1
	}

	// Restart containers any time they stop, die, are killed, etc.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#restart-policy
	sts.Spec.Template.Spec.RestartPolicy = v1.RestartPolicyAlways

	// ShareProcessNamespace makes Kubernetes' pause process PID 1 and lets
	// containers see each other's processes.
	// - https://docs.k8s.io/tasks/configure-pod-container/share-process-namespace/
	sts.Spec.Template.Spec.ShareProcessNamespace = initialize.Bool(true)

	sts.Spec.Template.Spec.ServiceAccountName = instanceServiceAccountName

	podSecurityContext := &v1.PodSecurityContext{SupplementalGroups: []int64{65534}}
	// set fsGroups if not OpenShift
	if cluster.Spec.OpenShift == nil || !*cluster.Spec.OpenShift {
		podSecurityContext.FSGroup = initialize.Int64(26)
	}
	sts.Spec.Template.Spec.SecurityContext = podSecurityContext
}

// addPGBackRestToInstancePodSpec adds pgBackRest configuration to the PodTemplateSpec.  This
// includes adding an SSH sidecar if a pgBackRest repoHost is enabled per the current
// PostgresCluster spec, mounting pgBackRest repo volumes if a dedicated repository is not
// configured, and then mounting the proper pgBackRest configuration resources (ConfigMaps
// and Secrets)
func addPGBackRestToInstancePodSpec(cluster *v1beta1.PostgresCluster,
	template *v1.PodTemplateSpec, instance *appsv1.StatefulSet) error {

	addSSH := pgbackrest.RepoHostEnabled(cluster)
	dedicatedRepoEnabled := pgbackrest.DedicatedRepoHostEnabled(cluster)
	pgBackRestConfigContainers := []string{naming.ContainerDatabase}
	if addSSH {
		pgBackRestConfigContainers = append(pgBackRestConfigContainers,
			naming.PGBackRestRepoContainerName)
		if err := pgbackrest.AddSSHToPod(cluster, template, naming.ContainerDatabase); err != nil {
			return err
		}
	}
	if !dedicatedRepoEnabled {
		if err := pgbackrest.AddRepoVolumesToPod(cluster, template,
			pgBackRestConfigContainers...); err != nil {
			return err
		}
	}
	if err := pgbackrest.AddConfigsToPod(cluster, template, instance.Name+".conf",
		pgBackRestConfigContainers...); err != nil {
		return err
	}

	return nil
}

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=create;patch

// reconcilePostgresDataVolume writes the PersistentVolumeClaim for instance's
// PostgreSQL data volume.
func (r *Reconciler) reconcilePostgresDataVolume(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
) (*corev1.PersistentVolumeClaim, error) {
	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.InstancePostgresDataVolume(instance)}
	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	err := errors.WithStack(r.setControllerReference(cluster, pvc))

	pvc.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		spec.Metadata.GetAnnotationsOrNil())

	pvc.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:     cluster.GetName(),
			naming.LabelInstanceSet: spec.Name,
			naming.LabelInstance:    instance.GetName(),
			naming.LabelRole:        naming.RolePostgresData,
		})

	pvc.Spec = spec.VolumeClaimSpec

	if err == nil {
		err = r.handlePersistentVolumeClaimError(cluster,
			errors.WithStack(r.apply(ctx, pvc)))
	}

	return pvc, err
}

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;patch

// reconcileInstanceConfigMap writes the ConfigMap that contains generated
// files (etc) that apply to instance of cluster.
func (r *Reconciler) reconcileInstanceConfigMap(
	ctx context.Context, cluster *v1beta1.PostgresCluster, spec *v1beta1.PostgresInstanceSetSpec,
	instance *appsv1.StatefulSet,
) (*v1.ConfigMap, error) {
	instanceConfigMap := &v1.ConfigMap{ObjectMeta: naming.InstanceConfigMap(instance)}
	instanceConfigMap.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))

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
		err = patroni.InstanceConfigMap(ctx, cluster, instance, instanceConfigMap)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, instanceConfigMap))
	}

	return instanceConfigMap, err
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// reconcileInstanceCertificates writes the Secret that contains certificates
// and private keys for instance of cluster.
func (r *Reconciler) reconcileInstanceCertificates(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	spec *v1beta1.PostgresInstanceSetSpec, instance *appsv1.StatefulSet,
	root *pki.RootCertificateAuthority,
) (*v1.Secret, error) {
	existing := &v1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}
	err := errors.WithStack(client.IgnoreNotFound(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing)))

	instanceCerts := &v1.Secret{ObjectMeta: naming.InstanceCertificates(instance)}
	instanceCerts.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

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
	instanceCerts.Type = v1.SecretTypeOpaque
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
		err = errors.WithStack(r.apply(ctx, instanceCerts))
	}

	return instanceCerts, err
}
