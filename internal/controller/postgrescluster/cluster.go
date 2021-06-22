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
	"net/url"

	"github.com/pkg/errors"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/patroni"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	"github.com/crunchydata/postgres-operator/internal/util"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources=configmaps,verbs=create;patch

// reconcileClusterConfigMap writes the ConfigMap that contains generated
// files (etc) that apply to the entire cluster.
func (r *Reconciler) reconcileClusterConfigMap(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
	pgHBAs postgres.HBAs, pgParameters postgres.Parameters, pgUser *v1.Secret,
) (*v1.ConfigMap, error) {
	clusterConfigMap := &v1.ConfigMap{ObjectMeta: naming.ClusterConfigMap(cluster)}
	clusterConfigMap.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("ConfigMap"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterConfigMap))

	clusterConfigMap.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	clusterConfigMap.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
		})

	if err == nil {
		err = patroni.ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters, pgUser,
			clusterConfigMap)
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterConfigMap))
	}

	return clusterConfigMap, err
}

// +kubebuilder:rbac:groups="",resources=services,verbs=create;patch

// reconcileClusterPodService writes the Service that can provide stable DNS
// names to Pods related to cluster.
func (r *Reconciler) reconcileClusterPodService(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*v1.Service, error) {
	clusterPodService := &v1.Service{ObjectMeta: naming.ClusterPodService(cluster)}
	clusterPodService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

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
	clusterPodService.Spec.ClusterIP = v1.ClusterIPNone
	clusterPodService.Spec.PublishNotReadyAddresses = true
	clusterPodService.Spec.Selector = map[string]string{
		naming.LabelCluster: cluster.Name,
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterPodService))
	}

	return clusterPodService, err
}

// +kubebuilder:rbac:groups="",resources=endpoints,verbs=create;patch
// +kubebuilder:rbac:groups="",resources=services,verbs=create;patch

// The OpenShift RestrictedEndpointsAdmission plugin requires special
// authorization to create Endpoints that contain ClusterIPs.
// - https://github.com/openshift/origin/pull/9383
// +kubebuilder:rbac:groups="",resources=endpoints/restricted,verbs=create

// reconcileClusterPrimaryService writes the Service and Endpoints that resolve
// to the PostgreSQL primary instance.
func (r *Reconciler) reconcileClusterPrimaryService(
	ctx context.Context, cluster *v1beta1.PostgresCluster, leader *v1.Service,
) error {
	clusterPrimaryService := &v1.Service{ObjectMeta: naming.ClusterPrimaryService(cluster)}
	clusterPrimaryService.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Service"))

	err := errors.WithStack(r.setControllerReference(cluster, clusterPrimaryService))

	clusterPrimaryService.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	clusterPrimaryService.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePrimary,
		})

	if err == nil && leader == nil {
		// TODO(cbandy): We need to build a different kind of Service here.
		err = errors.New("Patroni DCS other than Kubernetes Endpoints is not implemented")
	}

	// We want to name and label our primary Service consistently. When Patroni is
	// using Endpoints for its DCS, however, they and any Service that uses them
	// must use the same name as the Patroni "scope" which has its own constraints.
	//
	// To stay free from those constraints, our primary Service will resolve to
	// the ClusterIP of the Service created in the reconcilePatroniLeaderLease
	// method when Patroni is using Endpoints.

	// Allocate no IP address (headless) and manage the Endpoints ourselves.
	// - https://docs.k8s.io/concepts/services-networking/service/#headless-services
	// - https://docs.k8s.io/concepts/services-networking/service/#services-without-selectors
	clusterPrimaryService.Spec.ClusterIP = v1.ClusterIPNone
	clusterPrimaryService.Spec.Selector = nil

	clusterPrimaryService.Spec.Ports = []v1.ServicePort{{
		Name:       naming.PortPostgreSQL,
		Port:       *cluster.Spec.Port,
		Protocol:   v1.ProtocolTCP,
		TargetPort: intstr.FromString(naming.PortPostgreSQL),
	}}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, clusterPrimaryService))
	}

	// Endpoints for a Service have the same name as the Service.
	endpoints := &v1.Endpoints{ObjectMeta: naming.ClusterPrimaryService(cluster)}
	endpoints.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Endpoints"))

	if err == nil {
		err = errors.WithStack(r.setControllerReference(cluster, endpoints))
	}

	endpoints.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	endpoints.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster: cluster.Name,
			naming.LabelRole:    naming.RolePrimary,
		})

	// Resolve to the ClusterIP for which Patroni has configured the Endpoints.
	endpoints.Subsets = []v1.EndpointSubset{{
		Addresses: []v1.EndpointAddress{{IP: leader.Spec.ClusterIP}},
	}}

	// Copy the EndpointPorts from the ServicePorts.
	for _, sp := range clusterPrimaryService.Spec.Ports {
		endpoints.Subsets[0].Ports = append(endpoints.Subsets[0].Ports,
			v1.EndpointPort{
				Name:     sp.Name,
				Port:     sp.Port,
				Protocol: sp.Protocol,
			})
	}

	if err == nil {
		err = errors.WithStack(r.apply(ctx, endpoints))
	}

	return err
}

// reconcileDataSource is responsible for reconciling the data source for a PostgreSQL cluster.
// This involves ensuring the PostgreSQL data directory for the cluster is properly populated
// prior to bootstrapping the cluster, specifically according to any data source configured in the
// PostgresCluster spec.
func (r *Reconciler) reconcileDataSource(ctx context.Context,
	cluster *v1beta1.PostgresCluster, observed *observedInstances) (bool, error) {

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
		cluster.Spec.DataSource.PostgresCluster != nil

	// determine if the user has requested an in-place restore
	restoreID := cluster.GetAnnotations()[naming.PGBackRestRestore]
	restoreInPlaceRequested := restoreID != "" &&
		cluster.Spec.Archive.PGBackRest.Restore != nil &&
		*cluster.Spec.Archive.PGBackRest.Restore.Enabled

	// Set the proper data source for the restore based on whether we're initializing the PG
	// data directory (e.g. for a new PostgreSQL cluster), or restoring an existing cluster
	// in place (and therefore recreating the data directory).  If the user hasn't requested
	// PG data initialization or an in-place restore, then simply return.
	var dataSource *v1beta1.PostgresClusterDataSource
	switch {
	case restoreInPlaceRequested:
		dataSource = cluster.Spec.Archive.PGBackRest.Restore.PostgresClusterDataSource
	case postgresDataInitRequested:
		// there is no restore annotation when initializing a new cluster, so we create a
		// restore ID for bootstrap
		restoreID = "~pgo-bootstrap-" + cluster.GetName()
		dataSource = cluster.Spec.DataSource.PostgresCluster
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
	configs := []string{dataSource.ClusterName, dataSource.RepoName}
	configs = append(configs, dataSource.Options...)
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
	if err := r.reconcilePostgresClusterDataSource(ctx, cluster, dataSource,
		configHash); err != nil {
		return true, err
	}
	// return early until the PG data directory is initialized
	return true, nil
}

// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;patch

// reconcilePGUserSecret creates the secret that contains the default
// connection information to use with the postgrescluster
// TODO(tjmoore4): add updated reconciliation logic
func (r *Reconciler) reconcilePGUserSecret(
	ctx context.Context, cluster *v1beta1.PostgresCluster,
) (*v1.Secret, error) {
	existing := &v1.Secret{ObjectMeta: naming.PostgresUserSecret(cluster)}
	err := errors.WithStack(client.IgnoreNotFound(r.Client.Get(ctx,
		client.ObjectKeyFromObject(existing), existing)))
	if err != nil {
		return nil, err
	}

	intent := &v1.Secret{ObjectMeta: naming.PostgresUserSecret(cluster)}
	intent.SetGroupVersionKind(v1.SchemeGroupVersion.WithKind("Secret"))

	intent.Data = make(map[string][]byte)

	// TODO(jkatz): user as cluster name? could there be a different default here?
	intent.Data["user"] = []byte(cluster.Name)
	intent.Data["dbname"] = []byte(cluster.Name)
	intent.Data["port"] = []byte(fmt.Sprint(*cluster.Spec.Port))

	hostname := naming.ClusterPrimaryService(cluster).Name + "." +
		naming.ClusterPrimaryService(cluster).Namespace + ".svc"
	intent.Data["host"] = []byte(hostname)

	// if the password is not set, generate a new one
	if _, ok := existing.Data["password"]; !ok {
		password, err := util.GeneratePassword(util.DefaultGeneratedPasswordLength)
		if err != nil {
			return nil, err
		}
		// Generate the SCRAM verifier now and store alongside the plaintext
		// password so that later reconciles don't generate it repeatedly.
		// NOTE(cbandy): We don't have a function to compare a plaintext password
		// to a SCRAM verifier.
		verifier, err := pgpassword.NewSCRAMPassword(password).Build()
		if err != nil {
			return nil, err
		}
		intent.Data["password"] = []byte(password)
		intent.Data["verifier"] = []byte(verifier)
	} else {
		intent.Data["password"] = existing.Data["password"]
		intent.Data["verifier"] = existing.Data["verifier"]
	}

	// The stored connection string follows the PostgreSQL format.
	// The example followed is
	// postgresql://user:secret@localhost:port/mydb
	// where 'user' is the username, 'secret' is the password, 'localhost'
	// is the hostname, 'port' is the port and 'mydb' is the database name
	// https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING
	connectionString := (&url.URL{
		Scheme: "postgresql",
		Host:   fmt.Sprintf("%s:%d", hostname, *cluster.Spec.Port),
		User:   url.UserPassword(string(intent.Data["user"]), string(intent.Data["password"])),
		Path:   string(intent.Data["dbname"]),
	}).String()
	intent.Data["uri"] = []byte(connectionString)

	// if there is a pgBouncer instance, apply the pgBouncer settings. Otherwise
	// remove the pgBouncer settings
	if cluster.Spec.Proxy != nil && cluster.Spec.Proxy.PGBouncer != nil {
		pgBouncerHostname := naming.ClusterPGBouncer(cluster).Name + "." +
			naming.ClusterPGBouncer(cluster).Namespace + ".svc"
		intent.Data["pgbouncer-host"] = []byte(pgBouncerHostname)
		intent.Data["pgbouncer-port"] = []byte(fmt.Sprint(*cluster.Spec.Proxy.PGBouncer.Port))

		pgBouncerConnectionString := (&url.URL{
			Scheme: "postgresql",
			Host:   fmt.Sprintf("%s:%d", pgBouncerHostname, *cluster.Spec.Proxy.PGBouncer.Port),
			User:   url.UserPassword(string(intent.Data["user"]), string(intent.Data["password"])),
			Path:   string(intent.Data["dbname"]),
		}).String()
		intent.Data["pgbouncer-uri"] = []byte(pgBouncerConnectionString)
	}

	intent.Annotations = naming.Merge(cluster.Spec.Metadata.GetAnnotationsOrNil())
	intent.Labels = naming.Merge(cluster.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelCluster:    cluster.Name,
			naming.LabelUserSecret: cluster.Name,
		})

	err = errors.WithStack(r.setControllerReference(cluster, intent))

	if err == nil {
		err = errors.WithStack(r.apply(ctx, intent))
	}

	// if no error, return intent
	if err == nil {
		return intent, err
	}

	// do not return the intent if there was an error during apply
	return nil, err
}
