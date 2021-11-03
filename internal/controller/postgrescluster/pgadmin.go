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

	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pgadmin"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// reconcilePGAdmin writes the objects necessary to run a pgAdmin Pod.
func (r *Reconciler) reconcilePGAdmin(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) error {

	// TODO(tjmoore4): Currently, the returned service is only used in tests,
	// but it may be useful during upcoming feature enhancements. If not, we
	// may consider removing the service return altogether and refactoring
	// this function to only return errors.
	_, err := r.reconcilePGAdminService(ctx, cluster)
	var dataVolume *corev1.PersistentVolumeClaim
	if err == nil {
		dataVolume, err = r.reconcilePGAdminDataVolume(ctx, cluster)
	}
	if err == nil {
		err = r.reconcilePGAdminStatefulSet(ctx, cluster, dataVolume)
	}
	return err
}

// generatePGAdminService returns a v1.Service that exposes pgAdmin pods.
// The ServiceType comes from the cluster user interface spec.
func (r *Reconciler) generatePGAdminService(
	cluster *v1beta1.PostgresCluster) (*corev1.Service, bool, error,
) {
	service := &corev1.Service{ObjectMeta: naming.ClusterPGAdmin(cluster)}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	if cluster.Spec.UserInterface == nil || cluster.Spec.UserInterface.PGAdmin == nil {
		return service, false, nil
	}

	service.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.UserInterface.PGAdmin.Metadata.GetAnnotationsOrNil())
	service.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.UserInterface.PGAdmin.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGAdmin,
		})

	// Allocate an IP address and/or node port and let Kubernetes manage the
	// Endpoints by selecting Pods with the pgAdmin role.
	// - https://docs.k8s.io/concepts/services-networking/service/#defining-a-service
	service.Spec.Selector = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RolePGAdmin,
	}
	if spec := cluster.Spec.UserInterface.PGAdmin.Service; spec != nil {
		service.Spec.Type = corev1.ServiceType(spec.Type)
	} else {
		service.Spec.Type = corev1.ServiceTypeClusterIP
	}

	// The TargetPort must be the name (not the number) of the pgAdmin
	// ContainerPort. This name allows the port number to differ between Pods,
	// which can happen during a rolling update.
	//
	// TODO(tjmoore4): A custom service port is not currently supported as this
	// requires updates to the pgAdmin service configuration, but the spec
	// structures are in place to facilitate further enhancement.
	service.Spec.Ports = []corev1.ServicePort{{
		Name:       naming.PortPGAdmin,
		Port:       *cluster.Spec.UserInterface.PGAdmin.Port,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPGAdmin),
	}}

	err := errors.WithStack(r.setControllerReference(cluster, service))

	return service, true, err
}

// +kubebuilder:rbac:groups="",resources="services",verbs={get}
// +kubebuilder:rbac:groups="",resources="services",verbs={create,delete,patch}

// reconcilePGAdminService writes the Service that resolves to pgAdmin.
func (r *Reconciler) reconcilePGAdminService(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.Service, error) {
	service, specified, err := r.generatePGAdminService(cluster)

	if err == nil && !specified {
		// pgAdmin is disabled; delete the Service if it exists. Check the client
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

// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=create;delete;patch

// reconcilePGAdminStatefulSet writes the StatefulSet that runs pgAdmin.
func (r *Reconciler) reconcilePGAdminStatefulSet(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	dataVolume *corev1.PersistentVolumeClaim,
) error {
	sts := &appsv1.StatefulSet{ObjectMeta: naming.ClusterPGAdmin(cluster)}
	sts.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("StatefulSet"))

	if cluster.Spec.UserInterface == nil || cluster.Spec.UserInterface.PGAdmin == nil {
		// pgAdmin is disabled; delete the Deployment if it exists. Check the
		// client cache first using Get.
		key := client.ObjectKeyFromObject(sts)
		err := errors.WithStack(r.Client.Get(ctx, key, sts))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, sts))
		}
		return client.IgnoreNotFound(err)
	}

	sts.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.UserInterface.PGAdmin.Metadata.GetAnnotationsOrNil())
	sts.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.UserInterface.PGAdmin.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGAdmin,
			naming.LabelData:    naming.DataPGAdmin,
		})
	sts.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGAdmin,
		},
	}
	sts.Spec.Template.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
		cluster.Spec.UserInterface.PGAdmin.Metadata.GetAnnotationsOrNil())
	sts.Spec.Template.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		cluster.Spec.UserInterface.PGAdmin.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePGAdmin,
			naming.LabelData:    naming.DataPGAdmin,
		})

	// if the shutdown flag is set, set pgAdmin replicas to 0
	if cluster.Spec.Shutdown != nil && *cluster.Spec.Shutdown {
		sts.Spec.Replicas = initialize.Int32(0)
	} else {
		sts.Spec.Replicas = cluster.Spec.UserInterface.PGAdmin.Replicas
	}

	// Don't clutter the namespace with extra ControllerRevisions.
	sts.Spec.RevisionHistoryLimit = initialize.Int32(0)

	// Give the Pod a stable DNS record based on its name.
	// - https://docs.k8s.io/concepts/workloads/controllers/statefulset/#stable-network-id
	// - https://docs.k8s.io/concepts/services-networking/dns-pod-service/#pods
	sts.Spec.ServiceName = naming.ClusterPodService(cluster).Name

	// Set the StatefulSet update strategy to "RollingUpdate", and the Partition size for the
	// update strategy to 0 (note that these are the defaults for a StatefulSet).  This means
	// every pod of the StatefulSet will be deleted and recreated when the Pod template changes.
	// - https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#rolling-updates
	// - https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/#forced-rollback
	sts.Spec.UpdateStrategy.Type = appsv1.RollingUpdateStatefulSetStrategyType
	sts.Spec.UpdateStrategy.RollingUpdate = &appsv1.RollingUpdateStatefulSetStrategy{
		Partition: initialize.Int32(0),
	}

	// Use scheduling constraints from the cluster spec.
	sts.Spec.Template.Spec.Affinity = cluster.Spec.UserInterface.PGAdmin.Affinity
	sts.Spec.Template.Spec.Tolerations = cluster.Spec.UserInterface.PGAdmin.Tolerations

	if cluster.Spec.UserInterface.PGAdmin.PriorityClassName != nil {
		sts.Spec.Template.Spec.PriorityClassName = *cluster.Spec.UserInterface.PGAdmin.PriorityClassName
	}

	sts.Spec.Template.Spec.TopologySpreadConstraints =
		cluster.Spec.UserInterface.PGAdmin.TopologySpreadConstraints

	// Restart containers any time they stop, die, are killed, etc.
	// - https://docs.k8s.io/concepts/workloads/pods/pod-lifecycle/#restart-policy
	sts.Spec.Template.Spec.RestartPolicy = corev1.RestartPolicyAlways

	// pgAdmin does not make any Kubernetes API calls. Use the default
	// ServiceAccount and do not mount its credentials.
	sts.Spec.Template.Spec.AutomountServiceAccountToken = initialize.Bool(false)

	sts.Spec.Template.Spec.SecurityContext = initialize.RestrictedPodSecurityContext()

	// set the image pull secrets, if any exist
	sts.Spec.Template.Spec.ImagePullSecrets = cluster.Spec.ImagePullSecrets

	err := errors.WithStack(r.setControllerReference(cluster, sts))

	if err == nil {
		pgadmin.Pod(cluster, &sts.Spec.Template.Spec, dataVolume)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, sts))
	}

	return err
}

// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=create;patch

// reconcilePGAdminDataVolume writes the PersistentVolumeClaim for instance's
// pgAdmin data volume.
func (r *Reconciler) reconcilePGAdminDataVolume(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.PersistentVolumeClaim, error) {

	labelMap := map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RolePGAdmin,
		naming.LabelData:    naming.DataPGAdmin,
	}

	pvc := &corev1.PersistentVolumeClaim{ObjectMeta: naming.ClusterPGAdmin(cluster)}
	pvc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"))

	if cluster.Spec.UserInterface == nil || cluster.Spec.UserInterface.PGAdmin == nil {
		// pgAdmin is disabled; delete the PVC if it exists. Check the client
		// cache first using Get.
		key := client.ObjectKeyFromObject(pvc)
		err := errors.WithStack(r.Client.Get(ctx, key, pvc))
		if err == nil {
			err = errors.WithStack(r.deleteControlled(ctx, cluster, pvc))
		}
		return nil, client.IgnoreNotFound(err)
	}

	pvc.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil(),
	)
	pvc.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		labelMap,
	)
	pvc.Spec = cluster.Spec.UserInterface.PGAdmin.DataVolumeClaimSpec

	err := errors.WithStack(r.setControllerReference(cluster, pvc))

	if err == nil {
		err = r.handlePersistentVolumeClaimError(cluster,
			errors.WithStack(r.apply(ctx, pvc)))
	}

	return pvc, err
}
