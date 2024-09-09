// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgbouncer"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// reconcilePGBouncer writes the objects necessary to run a PgBouncer Pod.
func (r *Reconciler) reconcilePGBouncer(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
	primaryCertificate *corev1.SecretProjection,
	root *pki.RootCertificateAuthority,
) error {
	var (
		configmap *corev1.ConfigMap
		secret    *corev1.Secret
	)

	service, err := r.reconcilePGBouncerService(ctx, cluster)
	if err == nil {
		configmap, err = r.reconcilePGBouncerConfigMap(ctx, cluster)
	}
	if err == nil {
		secret, err = r.reconcilePGBouncerSecret(ctx, cluster, root, service)
	}
	if err == nil {
		err = r.reconcilePGBouncerDeployment(ctx, cluster, primaryCertificate, configmap, secret)
	}
	if err == nil {
		err = r.reconcilePGBouncerPodDisruptionBudget(ctx, cluster)
	}
	if err == nil {
		err = r.reconcilePGBouncerInPostgreSQL(ctx, cluster, instances, secret)
	}
	return err
}

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={get}
// +kubebuilder:rbac:groups="",resources="configmaps",verbs={create,delete,patch}

// reconcilePGBouncerConfigMap writes the ConfigMap for a PgBouncer Pod.
func (r *Reconciler) reconcilePGBouncerConfigMap(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{ObjectMeta: naming.ClusterPGBouncer(cluster)}
	configmap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; delete the ConfigMap if it exists. Check the
		// client cache first using Get.
		key := client.ObjectKeyFromObject(configmap)
		err := errors.WithStack(r.Client.Get(ctx, key, configmap))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, configmap))
		}
		return nil, client.IgnoreNotFound(err)
	}

	err := errors.WithStack(r.setControllerReference(cluster, configmap))

	configmap.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetAnnotationsOrNil())
	configmap.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		})

	if err == nil {
		pgbouncer.ConfigMap(cluster, configmap)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, configmap))
	}

	return configmap, err
}

// +kubebuilder:rbac:groups="",resources="pods",verbs={get,list}

// reconcilePGBouncerInPostgreSQL writes the user and other objects needed by
// PgBouncer inside of PostgreSQL.
func (r *Reconciler) reconcilePGBouncerInPostgreSQL(
	ctx context.Context, cluster *v1beta1.PostgresCluster, instances *observedInstances,
	clusterSecret *corev1.Secret,
) error {
	var pod *corev1.Pod

	// Find the PostgreSQL instance that can execute SQL that writes to every
	// database. When there is none, return early.

	for _, instance := range instances.forCluster {
		writable, known := instance.IsWritable()
		if writable && known && len(instance.Pods) > 0 {
			pod = instance.Pods[0]
			break
		}
	}
	if pod == nil {
		return nil
	}

	// PostgreSQL is available for writes. Prepare to either add or remove
	// PgBouncer objects.

	action := func(ctx context.Context, exec postgres.Executor) error {
		return errors.WithStack(pgbouncer.EnableInPostgreSQL(ctx, exec, clusterSecret))
	}
	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled.
		action = func(ctx context.Context, exec postgres.Executor) error {
			return errors.WithStack(pgbouncer.DisableInPostgreSQL(ctx, exec))
		}
	}

	// First, calculate a hash of the SQL that should be executed in PostgreSQL.

	revision, err := safeHash32(func(hasher io.Writer) error {
		// Discard log messages from the pgbouncer package about executing SQL.
		// Nothing is being "executed" yet.
		return action(logging.NewContext(ctx, logging.Discard()), func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, command ...string,
		) error {
			_, err := io.Copy(hasher, stdin)
			if err == nil {
				_, err = fmt.Fprint(hasher, command)
			}
			return err
		})
	})
	if err != nil {
		return err
	}

	if revision == cluster.Status.Proxy.PGBouncer.PostgreSQLRevision {
		// The necessary SQL has already been applied; there's nothing more to do.

		// TODO(cbandy): Give the user a way to trigger execution regardless.
		// The value of an annotation could influence the hash, for example.
		return nil
	}

	// Apply the necessary SQL and record its hash in cluster.Status. Include
	// the hash in any log messages.

	if err == nil {
		ctx := logging.NewContext(ctx, logging.FromContext(ctx).WithValues("revision", revision))
		err = action(ctx, func(ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
			return r.PodExec(ctx, pod.Namespace, pod.Name, naming.ContainerDatabase, stdin, stdout, stderr, command...)
		})
	}
	if err == nil {
		cluster.Status.Proxy.PGBouncer.PostgreSQLRevision = revision
	}

	return err
}

// +kubebuilder:rbac:groups="",resources="secrets",verbs={get}
// +kubebuilder:rbac:groups="",resources="secrets",verbs={create,delete,patch}

// reconcilePGBouncerSecret writes the Secret for a PgBouncer Pod.
func (r *Reconciler) reconcilePGBouncerSecret(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	root *pki.RootCertificateAuthority, service *corev1.Service,
) (*corev1.Secret, error) {
	existing := &corev1.Secret{ObjectMeta: naming.ClusterPGBouncer(cluster)}
	err := errors.WithStack(
		r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
	if client.IgnoreNotFound(err) != nil {
		return nil, err
	}

	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		// PgBouncer is disabled; delete the Secret if it exists.
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, existing))
		}
		return nil, client.IgnoreNotFound(err)
	}

	err = client.IgnoreNotFound(err)

	intent := &corev1.Secret{ObjectMeta: naming.ClusterPGBouncer(cluster)}
	intent.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
	intent.Type = corev1.SecretTypeOpaque

	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, intent))
	}

	intent.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetAnnotationsOrNil())
	intent.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		})

	if err == nil {
		err = pgbouncer.Secret(ctx, cluster, root, existing, service, intent)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}

	return intent, err
}

// generatePGBouncerService returns a v1.Service that exposes PgBouncer pods.
// The ServiceType comes from the cluster proxy spec.
func (r *Reconciler) generatePGBouncerService(
	cluster *v1beta1.PostgresCluster) (*corev1.Service, bool, error,
) {
	service := &corev1.Service{ObjectMeta: naming.ClusterPGBouncer(cluster)}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		return service, false, nil
	}

	service.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetAnnotationsOrNil())
	service.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetLabelsOrNil())

	if spec := cluster.Spec.Proxy.PGBouncer.Service; spec != nil {
		service.Annotations = naming.Merge(service.Annotations,
			spec.Metadata.GetAnnotationsOrNil())
		service.Labels = naming.Merge(service.Labels,
			spec.Metadata.GetLabelsOrNil())
	}

	// add our labels last so they aren't overwritten
	service.Labels = naming.Merge(service.Labels,
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		})

	// Allocate an IP address and/or node port and let Kubernetes manage the
	// Endpoints by selecting Pods with the PgBouncer role.
	// - https://docs.k8s.io/concepts/services-networking/service/#defining-a-service
	service.Spec.Selector = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RolePGBouncer,
	}

	// The TargetPort must be the name (not the number) of the PgBouncer
	// ContainerPort. This name allows the port number to differ between Pods,
	// which can happen during a rolling update.
	servicePort := corev1.ServicePort{
		Name:       naming.PortPGBouncer,
		Port:       *cluster.Spec.Proxy.PGBouncer.Port,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPGBouncer),
	}

	if spec := cluster.Spec.Proxy.PGBouncer.Service; spec == nil {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	} else {
		service.Spec.Type = corev1.ServiceType(spec.Type)
		if spec.NodePort != nil {
			if service.Spec.Type == corev1.ServiceTypeClusterIP {
				// The NodePort can only be set when the Service type is NodePort or
				// LoadBalancer. However, due to a known issue prior to Kubernetes
				// 1.20, we clear these errors during our apply. To preserve the
				// appropriate behavior, we log an Event and return an error.
				// TODO(tjmoore4): Once Validation Rules are available, this check
				// and event could potentially be removed in favor of that validation
				r.Recorder.Eventf(cluster, corev1.EventTypeWarning, "MisconfiguredClusterIP",
					"NodePort cannot be set with type ClusterIP on Service %q", service.Name)
				return nil, true, fmt.Errorf("NodePort cannot be set with type ClusterIP on Service %q", service.Name)
			}
			servicePort.NodePort = *spec.NodePort
		}
		if spec.ExternalTrafficPolicy != nil {
			service.Spec.ExternalTrafficPolicy = *spec.ExternalTrafficPolicy
		}
		if spec.InternalTrafficPolicy != nil {
			service.Spec.InternalTrafficPolicy = spec.InternalTrafficPolicy
		}
	}
	service.Spec.Ports = []corev1.ServicePort{servicePort}

	err := errors.WithStack(r.setControllerReference(cluster, service))

	return service, true, err
}

// +kubebuilder:rbac:groups="",resources="services",verbs={get}
// +kubebuilder:rbac:groups="",resources="services",verbs={create,delete,patch}

// reconcilePGBouncerService writes the Service that resolves to PgBouncer.
func (r *Reconciler) reconcilePGBouncerService(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.Service, error) {
	service, specified, err := r.generatePGBouncerService(cluster)

	if err == nil && !specified {
		// PgBouncer is disabled; delete the Service if it exists. Check the client
		// cache first using Get.
		key := client.ObjectKeyFromObject(service)
		err := errors.WithStack(r.Client.Get(ctx, key, service))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, service))
		}
		return nil, client.IgnoreNotFound(err)
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, service))
	}
	return service, err
}

// generatePGBouncerDeployment returns an appsv1.Deployment that runs PgBouncer pods.
func (r *Reconciler) generatePGBouncerDeployment(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	primaryCertificate *corev1.SecretProjection,
	configmap *corev1.ConfigMap, secret *corev1.Secret,
) (*appsv1.Deployment, bool, error) {
	deploy := &appsv1.Deployment{ObjectMeta: naming.ClusterPGBouncer(cluster)}
	deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))

	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		return deploy, false, nil
	}

	deploy.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetAnnotationsOrNil())
	deploy.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		})
	deploy.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		},
	}
	deploy.Spec.Template.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetAnnotationsOrNil())
	deploy.Spec.Template.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		})

	// if the shutdown flag is set, set pgBouncer replicas to 0
	if cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown {
		deploy.Spec.Replicas = initialize.Int32(0)
	} else {
		deploy.Spec.Replicas = cluster.Spec.Proxy.PGBouncer.Replicas
	}

	// Don't clutter the namespace with extra ReplicaSets.
	deploy.Spec.RevisionHistoryLimit = initialize.Int32(0)

	// Ensure that the number of Ready pods is never less than the specified
	// Replicas by starting new pods while old pods are still running.
	// - https://docs.k8s.io/concepts/workloads/controllers/deployment/#rolling-update-deployment
	deploy.Spec.Strategy.Type = appsv1.RollingUpdateDeploymentStrategyType
	deploy.Spec.Strategy.RollingUpdate = &appsv1.RollingUpdateDeployment{
		MaxUnavailable: intstr.ValueOrDefault(nil, intstr.FromInt(0)),
	}

	// Use scheduling constraints from the cluster spec.
	deploy.Spec.Template.Spec.Affinity = cluster.Spec.Proxy.PGBouncer.Affinity
	deploy.Spec.Template.Spec.Tolerations = cluster.Spec.Proxy.PGBouncer.Tolerations

	if cluster.Spec.Proxy.PGBouncer.PriorityClassName != nil {
		deploy.Spec.Template.Spec.PriorityClassName = *cluster.Spec.Proxy.PGBouncer.PriorityClassName
	}

	deploy.Spec.Template.Spec.TopologySpreadConstraints =
		cluster.Spec.Proxy.PGBouncer.TopologySpreadConstraints

	// if default pod scheduling is not explicitly disabled, add the default
	// pod topology spread constraints
	if cluster.Spec.DisableDefaultPodScheduling == nil ||
		(cluster.Spec.DisableDefaultPodScheduling != nil &&
			!*cluster.Spec.DisableDefaultPodScheduling) {
		deploy.Spec.Template.Spec.TopologySpreadConstraints = append(
			deploy.Spec.Template.Spec.TopologySpreadConstraints,
			defaultTopologySpreadConstraints(*deploy.Spec.Selector)...)
	}

	// Restart containers any time they stop, die, are killed, etc.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#restart-policy
	deploy.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways

	// ShareProcessNamespace makes Kubernetes' pause process PID 1 and lets
	// containers see each other's processes.
	// - https://docs.k8s.io/tasks/configure-pod-container/share-process-namespace/
	deploy.Spec.Template.Spec.ShareProcessNamespace = initialize.Bool(true)

	// There's no need for individual DNS names of PgBouncer pods.
	deploy.Spec.Template.Spec.Subdomain = ""

	// PgBouncer does not make any Kubernetes API calls. Use the default
	// ServiceAccount and do not mount its credentials.
	deploy.Spec.Template.Spec.AutomountServiceAccountToken = initialize.Bool(false)

	// Do not add environment variables describing services in this namespace.
	deploy.Spec.Template.Spec.EnableServiceLinks = initialize.Bool(false)

	deploy.Spec.Template.Spec.SecurityContext = initialize.PodSecurityContext()

	// set the image pull secrets, if any exist
	deploy.Spec.Template.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets

	err := errors.WithStack(r.setControllerReference(cluster, deploy))

	if err == nil {
		pgbouncer.Pod(ctx, cluster, configmap, primaryCertificate, secret, &deploy.Spec.Template.Spec)
	}

	return deploy, true, err
}

// +kubebuilder:rbac:groups="apps",resources="deployments",verbs={get}
// +kubebuilder:rbac:groups="apps",resources="deployments",verbs={create,delete,patch}

// reconcilePGBouncerDeployment writes the Deployment that runs PgBouncer.
func (r *Reconciler) reconcilePGBouncerDeployment(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	primaryCertificate *corev1.SecretProjection,
	configmap *corev1.ConfigMap, secret *corev1.Secret,
) error {
	deploy, specified, err := r.generatePGBouncerDeployment(
		ctx, cluster, primaryCertificate, configmap, secret)

	// Set observations whether the deployment exists or not.
	defer func() {
		cluster.Status.Proxy.PGBouncer.Replicas = deploy.Status.Replicas
		cluster.Status.Proxy.PGBouncer.ReadyReplicas = deploy.Status.ReadyReplicas

		// NOTE(cbandy): This should be somewhere else when there is more than
		// one proxy implementation.

		var available *appsv1.DeploymentCondition
		for i := range deploy.Status.Conditions {
			if deploy.Status.Conditions[i].Type == appsv1.DeploymentAvailable {
				available = &deploy.Status.Conditions[i]
			}
		}

		if available == nil {
			meta.RemoveStatusCondition(&cluster.Status.Conditions, v1beta1.ProxyAvailable)
		} else {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				Type:    v1beta1.ProxyAvailable,
				Status:  metav1.ConditionStatus(available.Status),
				Reason:  available.Reason,
				Message: available.Message,

				LastTransitionTime: available.LastTransitionTime,
				ObservedGeneration: cluster.Generation,
			})
		}
	}()

	if err == nil && !specified {
		// PgBouncer is disabled; delete the Deployment if it exists. Check the
		// client cache first using Get.
		key := client.ObjectKeyFromObject(deploy)
		err := errors.WithStack(r.Client.Get(ctx, key, deploy))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, deploy))
		}
		return client.IgnoreNotFound(err)
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, deploy))
	}
	return err
}

// +kubebuilder:rbac:groups="policy",resources="poddisruptionbudgets",verbs={create,patch,get,delete}

// reconcilePGBouncerPodDisruptionBudget creates a PDB for the PGBouncer deployment.
// A PDB will be created when minAvailable is determined to be greater than 0 and
// a PGBouncer proxy is defined in the spec. MinAvailable can be defined in the spec
// or a default value will be set based on the number of replicas defined for PGBouncer.
func (r *Reconciler) reconcilePGBouncerPodDisruptionBudget(
	ctx context.Context,
	cluster *v1beta1.PostgresCluster,
) error {
	deleteExistingPDB := func(cluster *v1beta1.PostgresCluster) error {
		existing := &policyv1.PodDisruptionBudget{ObjectMeta: naming.ClusterPGBouncer(cluster)}
		err := errors.WithStack(r.Client.Get(ctx, client.ObjectKeyFromObject(existing), existing))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, existing))
		}
		return client.IgnoreNotFound(err)
	}

	if cluster.Spec.Proxy == nil || cluster.Spec.Proxy.PGBouncer == nil {
		return deleteExistingPDB(cluster)
	}

	if cluster.Spec.Proxy.PGBouncer.Replicas == nil {
		// Replicas should always have a value because of defaults in the spec
		return errors.New("Replicas should be defined")
	}
	minAvailable := getMinAvailable(cluster.Spec.Proxy.PGBouncer.MinAvailable,
		*cluster.Spec.Proxy.PGBouncer.Replicas)

	// If 'minAvailable' is set to '0', we will not reconcile the PDB. If one
	// already exists, we will remove it.
	scaled, err := intstr.GetScaledValueFromIntOrPercent(minAvailable,
		int(*cluster.Spec.Proxy.PGBouncer.Replicas), true)
	if err == nil && scaled <= 0 {
		return deleteExistingPDB(cluster)
	}

	meta := naming.ClusterPGBouncer(cluster)
	meta.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGBouncer,
		})
	meta.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.Proxy.PGBouncer.Metadata.GetAnnotationsOrNil())

	selector := naming.ClusterPGBouncerSelector(cluster)
	pdb := &policyv1.PodDisruptionBudget{}
	if err == nil {
		pdb, err = r.generatePodDisruptionBudget(cluster, meta, minAvailable, selector)
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, pdb))
	}
	return err
}
