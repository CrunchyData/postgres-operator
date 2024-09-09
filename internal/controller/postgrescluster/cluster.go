// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={create,patch}

// reconcileClusterConfigMap writes the ConfigMap that contains generated
// files (etc) that apply to the entire cluster.
func (r *Reconciler) reconcileClusterConfigMap(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	pgHBAs postgres.HBAs, pgParameters postgres.Parameters,
) (*corev1.ConfigMap, error) {
	clusterConfigMap := &corev1.ConfigMap{ObjectMeta: naming.ClusterConfigMap(cluster)}
	clusterConfigMap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterConfigMap))

	clusterConfigMap.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	clusterConfigMap.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
		})

	if err == nil {
		err = patroni.ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters,
			clusterConfigMap)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterConfigMap))
	}

	return clusterConfigMap, err
}

// +kubebuilder:rbac:groups="",resources="services",verbs={create,patch}

// reconcileClusterPodService writes the Service that can provide stable DNS
// names to Pods related to cluster.
func (r *Reconciler) reconcileClusterPodService(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.Service, error) {
	clusterPodService := &corev1.Service{ObjectMeta: naming.ClusterPodService(cluster)}
	clusterPodService.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterPodService))

	clusterPodService.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	clusterPodService.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
		})

	// Allocate no IP address (headless) and match any Pod with the cluster
	// label, regardless of its readiness. Not particularly useful by itself, but
	// this allows a properly configured Pod to get a DNS record based on its name.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	// - https://docs.k8s.io/concepts/services-networking/dns-pod-service/#pods
	clusterPodService.Spec.ClusterIP = corev1.ClusterIPNone
	clusterPodService.Spec.PublishNotReadyAddresses = true
	clusterPodService.Spec.Selector = map[string]string{
		naming.LabelCluster: cluster.Name,
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterPodService))
	}

	return clusterPodService, err
}

// generateClusterPrimaryService returns a v1.Service and v1.Endpoints that
// resolve to the PostgreSQL primary instance.
func (r *Reconciler) generateClusterPrimaryService(
	cluster *v1beta1.PostgresCluster, leader *corev1.Service,
) (*corev1.Service, *corev1.Endpoints, error) {
	// We want to name and label our primary Service consistently. When Patroni is
	// using Endpoints for its DCS, however, they and any Service that uses them
	// must use the same name as the Patroni "scope" which has its own constraints.
	//
	// To stay free from those constraints, our primary Service resolves to the
	// ClusterIP of the Service created in Reconciler.reconcilePatroniLeaderLease
	// when Patroni is using Endpoints.

	service := &corev1.Service{ObjectMeta: naming.ClusterPrimaryService(cluster)}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	service.Annotations = naming.Merge(
		cluster.Spec.Metadata.GetAnnotationsOrNil())
	service.Labels = naming.Merge(
		cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePrimary,
		})

	err := errors.WithStack(r.setControllerReference(cluster, service))

	// Endpoints for a Service have the same name as the Service. Copy labels,
	// annotations, and ownership, too.
	endpoints := &corev1.Endpoints{}
	service.ObjectMeta.DeepCopyInto(&endpoints.ObjectMeta)
	endpoints.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Endpoints"))

	if leader == nil {
		// TODO(cbandy): We need to build a different kind of Service here.
		return nil, nil, errors.New("Patroni DCS other than Kubernetes Endpoints is not implemented")
	}

	// Allocate no IP address (headless) and manage the Endpoints ourselves.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	// - https://docs.k8s.io/concepts/services-networking/service/#services-without-selectors
	service.Spec.ClusterIP = corev1.ClusterIPNone
	service.Spec.Selector = nil

	service.Spec.Ports = []corev1.ServicePort{{
		Name:       naming.PortPostgreSQL,
		Port:       *cluster.Spec.Port,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPostgreSQL),
	}}

	// Resolve to the ClusterIP for which Patroni has configured the Endpoints.
	endpoints.Subsets = []corev1.EndpointSubset{{
		Addresses: []corev1.EndpointAddress{{IP: leader.Spec.ClusterIP}},
	}}

	// Copy the EndpointPorts from the ServicePorts.
	for _, sp := range service.Spec.Ports {
		endpoints.Subsets[0].Ports = append(endpoints.Subsets[0].Ports,
			corev1.EndpointPort{
				Name:     sp.Name,
				Port:     sp.Port,
				Protocol: sp.Protocol,
			})
	}

	return service, endpoints, err
}

// +kubebuilder:rbac:groups="",resources="endpoints",verbs={create,patch}
// +kubebuilder:rbac:groups="",resources="services",verbs={create,patch}

// The OpenShift RestrictedEndpointsAdmission plugin requires special
// authorization to create Endpoints that contain ClusterIPs.
// - https://github.com/openshift/origin/pull/9383
// +kubebuilder:rbac:groups="",resources="endpoints/restricted",verbs={create}

// reconcileClusterPrimaryService writes the Service and Endpoints that resolve
// to the PostgreSQL primary instance.
func (r *Reconciler) reconcileClusterPrimaryService(
	ctx context.Context, cluster *v1beta1.PostgresCluster, leader *corev1.Service,
) (*corev1.Service, error) {
	service, endpoints, err := r.generateClusterPrimaryService(cluster, leader)

	if err == nil {
		err = errors.WithStack(r.apply(ctx, service))
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, endpoints))
	}
	return service, err
}

// generateClusterReplicaService returns a v1.Service that exposes PostgreSQL
// replica instances.
func (r *Reconciler) generateClusterReplicaService(
	cluster *v1beta1.PostgresCluster) (*corev1.Service, error,
) {
	service := &corev1.Service{ObjectMeta: naming.ClusterReplicaService(cluster)}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))

	service.Annotations = cluster.Spec.Metadata.GetAnnotationsOrNil()
	service.Labels = cluster.Spec.Metadata.GetLabelsOrNil()

	if spec := cluster.Spec.ReplicaService; spec != nil {
		service.Annotations = naming.Merge(service.Annotations,
			spec.Metadata.GetAnnotationsOrNil())
		service.Labels = naming.Merge(service.Labels,
			spec.Metadata.GetLabelsOrNil())
	}

	// add our labels last so they aren't overwritten
	service.Labels = naming.Merge(
		service.Labels,
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RoleReplica,
		})

	// The TargetPort must be the name (not the number) of the PostgreSQL
	// ContainerPort. This name allows the port number to differ between Pods,
	// which can happen during a rolling update.
	servicePort := corev1.ServicePort{
		Name:       naming.PortPostgreSQL,
		Port:       *cluster.Spec.Port,
		Protocol:   corev1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPostgreSQL),
	}

	// Default to a service type of ClusterIP
	service.Spec.Type = corev1.ServiceTypeClusterIP

	// Check user provided spec for a specified type
	if spec := cluster.Spec.ReplicaService; spec != nil {
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
				return nil, fmt.Errorf("NodePort cannot be set with type ClusterIP on Service %q", service.Name)
			}
			servicePort.NodePort = *spec.NodePort
		}
	}
	service.Spec.Ports = []corev1.ServicePort{servicePort}

	// Allocate an IP address and let Kubernetes manage the Endpoints by
	// selecting Pods with the Patroni replica role.
	// - https://docs.k8s.io/concepts/services-networking/service/#defining-a-service
	service.Spec.Selector = map[string]string{
		naming.LabelCluster: cluster.Name,
		naming.LabelRole:    naming.RolePatroniReplica,
	}

	err := errors.WithStack(r.setControllerReference(cluster, service))

	return service, err
}

// +kubebuilder:rbac:groups="",resources="services",verbs={create,patch}

// reconcileClusterReplicaService writes the Service that exposes PostgreSQL
// replica instances.
func (r *Reconciler) reconcileClusterReplicaService(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*corev1.Service, error) {
	service, err := r.generateClusterReplicaService(cluster)

	if err == nil {
		err = errors.WithStack(r.apply(ctx, service))
	}
	return service, err
}

// reconcileDataSource is responsible for reconciling the data source for a PostgreSQL cluster.
// This involves ensuring the PostgreSQL data directory for the cluster is properly populated
// prior to bootstrapping the cluster, specifically according to any data source configured in the
// PostgresCluster spec.
// TODO(benjaminjb): Right now the spec will accept a dataSource with both a PostgresCluster and
// a PGBackRest section, but the code will only honor the PostgresCluster in that case; this would
// be better handled with a webhook to reject a spec with both `dataSource.postgresCluster` and
// `dataSource.pgbackrest` fields
func (r *Reconciler) reconcileDataSource(ctx context.Context,
	cluster *v1beta1.PostgresCluster, observed *observedInstances,
	clusterVolumes []corev1.PersistentVolumeClaim,
	rootCA *pki.RootCertificateAuthority,
	backupsSpecFound bool,
) (bool, error) {

	// a hash func to hash the pgBackRest restore options
	hashFunc := func(jobConfigs []string) (string, error) {
		return safeHash32(func(w io.Writer) (err error) {
			for _, o := range jobConfigs {
				_, err = w.Write([]byte(o))
			}
			return
		})
	}

	// observe all resources currently relevant to reconciling data sources, and update status
	// accordingly
	endpoints, restoreJob, err := r.observeRestoreEnv(ctx, cluster)
	if err != nil {
		return false, errors.WithStack(err)
	}

	// determine if the user wants to initialize the PG data directory
	postgresDataInitRequested := cluster.Spec.DataSource != nil &&
		(cluster.Spec.DataSource.PostgresCluster != nil ||
			cluster.Spec.DataSource.PGBackRest != nil)

	// determine if the user has requested an in-place restore
	restoreID := cluster.GetAnnotations()[naming.PGBackRestRestore]
	restoreInPlaceRequested := restoreID != "" &&
		cluster.Spec.Backups.PGBackRest.Restore != nil &&
		*cluster.Spec.Backups.PGBackRest.Restore.Enabled

	// Set the proper data source for the restore based on whether we're initializing the PG
	// data directory (e.g. for a new PostgreSQL cluster), or restoring an existing cluster
	// in place (and therefore recreating the data directory).  If the user hasn't requested
	// PG data initialization or an in-place restore, then simply return.
	var dataSource *v1beta1.PostgresClusterDataSource
	var cloudDataSource *v1beta1.PGBackRestDataSource
	switch {
	case restoreInPlaceRequested:
		dataSource = cluster.Spec.Backups.PGBackRest.Restore.PostgresClusterDataSource
	case postgresDataInitRequested:
		// there is no restore annotation when initializing a new cluster, so we create a
		// restore ID for bootstrap
		restoreID = "~pgo-bootstrap-" + cluster.GetName()
		dataSource = cluster.Spec.DataSource.PostgresCluster
		if dataSource == nil {
			cloudDataSource = cluster.Spec.DataSource.PGBackRest
		}
	default:
		return false, nil
	}

	// check the cluster's conditions to determine if the PG data for the cluster has been
	// initialized
	dataSourceCondition := meta.FindStatusCondition(cluster.Status.Conditions,
		ConditionPostgresDataInitialized)
	postgresDataInitialized := dataSourceCondition != nil &&
		(dataSourceCondition.Status == metav1.ConditionTrue)

	// check the cluster's conditions to determine if an in-place restore is in progress,
	// and if the reason for that condition indicates that the cluster has been prepared for
	// restore
	restoreCondition := meta.FindStatusCondition(cluster.Status.Conditions,
		ConditionPGBackRestRestoreProgressing)
	restoringInPlace := restoreCondition != nil &&
		(restoreCondition.Status == metav1.ConditionTrue)
	readyForRestore := restoreCondition != nil &&
		restoringInPlace &&
		(restoreCondition.Reason == ReasonReadyForRestore)

	// check the restore status to see if the ID for the restore currently being requested (as
	// provided by the user via annotation) has changed
	var restoreIDStatus string
	if cluster.Status.PGBackRest != nil && cluster.Status.PGBackRest.Restore != nil {
		restoreIDStatus = cluster.Status.PGBackRest.Restore.ID
	}
	restoreIDChanged := (restoreID != restoreIDStatus)

	// calculate the configHash for the options in the current data source, and if an existing
	// restore Job exists, determine if the config has changed
	var configs []string
	switch {
	case dataSource != nil:
		configs = []string{dataSource.ClusterName, dataSource.RepoName}
		configs = append(configs, dataSource.Options...)
	case cloudDataSource != nil:
		configs = []string{cloudDataSource.Stanza, cloudDataSource.Repo.Name}
		configs = append(configs, cloudDataSource.Options...)
	}
	configHash, err := hashFunc(configs)
	if err != nil {
		return false, errors.WithStack(err)
	}
	var configChanged bool
	if restoreJob != nil {
		configChanged =
			(configHash != restoreJob.GetAnnotations()[naming.PGBackRestConfigHash])
	}

	// Proceed with preparing the cluster for restore (e.g. tearing down runners, the DCS,
	// etc.) if:
	// - A restore is already in progress, but the cluster has not yet been prepared
	// - A restore is already in progress, but the config hash changed
	// - The restore ID has changed (i.e. the user provide a new value for the restore
	//   annotation, indicating they want a new in-place restore)
	if (restoringInPlace && (!readyForRestore || configChanged)) || restoreIDChanged {
		if err := r.prepareForRestore(ctx, cluster, observed, endpoints,
			restoreJob, restoreID); err != nil {
			return true, err
		}
		// return early and don't restore (i.e. populate the data dir) until the cluster is
		// prepared for restore
		return true, nil
	}

	// simply return if data is already initialized
	if postgresDataInitialized {
		return false, nil
	}

	// proceed with initializing the PG data directory if not already initialized
	switch {
	case dataSource != nil:
		if err := r.reconcilePostgresClusterDataSource(ctx, cluster, dataSource,
			configHash, clusterVolumes, rootCA,
			backupsSpecFound); err != nil {
			return true, err
		}
	case cloudDataSource != nil:
		if err := r.reconcileCloudBasedDataSource(ctx, cluster, cloudDataSource,
			configHash, clusterVolumes); err != nil {
			return true, err
		}
	}
	// return early until the PG data directory is initialized
	return true, nil
}
