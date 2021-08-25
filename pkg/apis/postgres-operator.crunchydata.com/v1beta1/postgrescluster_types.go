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

package v1beta1

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PostgresClusterSpec defines the desired state of PostgresCluster
type PostgresClusterSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// Specifies a data source for bootstrapping the PostgreSQL cluster.
	// +optional
	DataSource *DataSource `json:"dataSource,omitempty"`

	// PostgreSQL backup configuration
	// +kubebuilder:validation:Required
	Backups Backups `json:"backups"`

	// The secret containing the Certificates and Keys to encrypt PostgreSQL
	// traffic will need to contain the server TLS certificate, TLS key and the
	// Certificate Authority certificate with the data keys set to tls.crt,
	// tls.key and ca.crt, respectively. It will then be mounted as a volume
	// projection to the '/pgconf/tls' directory. For more information on
	// Kubernetes secret projections, please see
	// https://k8s.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths
	// NOTE: If CustomTLSSecret is provided, CustomReplicationClientTLSSecret
	// MUST be provided and the ca.crt provided must be the same.
	// +optional
	CustomTLSSecret *corev1.SecretProjection `json:"customTLSSecret,omitempty"`

	// The secret containing the replication client certificates and keys for
	// secure connections to the PostgreSQL server. It will need to contain the
	// client TLS certificate, TLS key and the Certificate Authority certificate
	// with the data keys set to tls.crt, tls.key and ca.crt, respectively.
	// NOTE: If CustomReplicationClientTLSSecret is provided, CustomTLSSecret
	// MUST be provided and the ca.crt provided must be the same.
	// +optional
	CustomReplicationClientTLSSecret *corev1.SecretProjection `json:"customReplicationTLSSecret,omitempty"`

	// The image name to use for PostgreSQL containers. When omitted, the value
	// comes from an operator environment variable. For standard PostgreSQL images,
	// the format is RELATED_IMAGE_POSTGRES_{postgresVersion},
	// e.g. RELATED_IMAGE_POSTGRES_13. For PostGIS enabled PostgreSQL images,
	// the format is RELATED_IMAGE_POSTGRES_{postgresVersion}_GIS_{postGISVersion},
	// e.g. RELATED_IMAGE_POSTGRES_13_GIS_3.1.
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	Image string `json:"image,omitempty"`

	// The image pull secrets used to pull from a private registry
	// Changing this value causes all running pods to restart.
	// https://k8s.io/docs/tasks/configure-pod-container/pull-image-private-registry/
	// +optional
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +listType=map
	// +listMapKey=name
	// +kubebuilder:validation:MinItems=1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	InstanceSets []PostgresInstanceSetSpec `json:"instances"`

	// Whether or not the PostgreSQL cluster is being deployed to an OpenShift
	// environment. If the field is unset, the operator will automatically
	// detect the environment.
	// +optional
	OpenShift *bool `json:"openshift,omitempty"`

	// +optional
	Patroni *PatroniSpec `json:"patroni,omitempty"`

	// The port on which PostgreSQL should listen.
	// +optional
	// +kubebuilder:default=5432
	// +kubebuilder:validation:Minimum=1024
	Port *int32 `json:"port,omitempty"`

	// The major version of PostgreSQL installed in the PostgreSQL image
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Minimum=10
	// +kubebuilder:validation:Maximum=13
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	PostgresVersion int `json:"postgresVersion"`

	// The PostGIS extension version installed in the PostgreSQL image.
	// When image is not set, indicates a PostGIS enabled image will be used.
	// +optional
	PostGISVersion string `json:"postGISVersion,omitempty"`

	// The specification of a proxy that connects to PostgreSQL.
	// +optional
	Proxy *PostgresProxySpec `json:"proxy,omitempty"`

	// The specification of monitoring tools that connect to PostgreSQL
	// +optional
	Monitoring *MonitoringSpec `json:"monitoring,omitempty"`

	// Specification of the service that exposes the PostgreSQL primary instance.
	// +optional
	Service *ServiceSpec `json:"service,omitempty"`

	// Whether or not the PostgreSQL cluster should be stopped.
	// When this is true, workloads are scaled to zero and CronJobs
	// are suspended.
	// Other resources, such as Services and Volumes, remain in place.
	// +optional
	Shutdown *bool `json:"shutdown,omitempty"`

	// Run this cluster as a read-only copy of an existing cluster or archive.
	// +optional
	Standby *PostgresStandbySpec `json:"standby,omitempty"`

	// A list of group IDs applied to the process of a container. These can be
	// useful when accessing shared file systems with constrained permissions.
	// More info: https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context
	// +optional
	SupplementalGroups []int64 `json:"supplementalGroups,omitempty"`

	// Users to create inside PostgreSQL and the databases they should access.
	// The default creates one user that can access one database matching the
	// PostgresCluster name. An empty list creates no users. Removing a user
	// from this list does NOT drop the user nor revoke their access.
	// +listType=map
	// +listMapKey=name
	// +optional
	Users []PostgresUserSpec `json:"users,omitempty"`
}

// DataSource defines the source of the PostgreSQL data directory for a new PostgresCluster.
type DataSource struct {
	// Defines a pgBackRest data source that can be used to pre-populate the PostgreSQL data
	// directory for a new PostgreSQL cluster using a pgBackRest restore.
	// +optional
	PostgresCluster *PostgresClusterDataSource `json:"postgresCluster,omitempty"`
}

// PostgresClusterDataSource defines a data source for bootstrapping PostgreSQL clusters using a
// an existing PostgresCluster.
type PostgresClusterDataSource struct {

	// The name of an existing PostgresCluster to use as the data source for the new PostgresCluster.
	// Defaults to the name of the PostgresCluster being created if not provided.
	// +optional
	ClusterName string `json:"clusterName,omitempty"`

	// The namespace of the cluster specified as the data source using the clusterName field.
	// Defaults to the namespace of the PostgresCluster being created if not provided.
	// +optional
	ClusterNamespace string `json:"clusterNamespace,omitempty"`

	// The name of the pgBackRest repo within the source PostgresCluster that contains the backups
	// that should be utilized to perform a pgBackRest restore when initializing the data source
	// for the new PostgresCluster.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^repo[1-4]
	RepoName string `json:"repoName"`

	// Command line options to include when running the pgBackRest restore command.
	// https://pgbackrest.org/command.html#command-restore
	// +optional
	Options []string `json:"options,omitempty"`

	// Resource requirements for the pgBackRest restore Job.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Scheduling constraints of the pgBackRest restore Job.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Tolerations of the pgBackRest restore Job.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
}

// Default defines several key default values for a Postgres cluster.
func (s *PostgresClusterSpec) Default() {
	for i := range s.InstanceSets {
		s.InstanceSets[i].Default(i)
	}

	if s.Patroni == nil {
		s.Patroni = new(PatroniSpec)
	}
	s.Patroni.Default()

	if s.Port == nil {
		s.Port = new(int32)
		*s.Port = 5432
	}

	if s.Proxy != nil {
		s.Proxy.Default()
	}
}

// Backups defines a PostgreSQL archive configuration
type Backups struct {

	// pgBackRest archive configuration
	// +kubebuilder:validation:Required
	PGBackRest PGBackRestArchive `json:"pgbackrest"`
}

// PostgresClusterStatus defines the observed state of PostgresCluster
type PostgresClusterStatus struct {

	// Identifies the databases that have been installed into PostgreSQL.
	DatabaseRevision string `json:"databaseRevision,omitempty"`

	// Current state of PostgreSQL instances.
	// +listType=map
	// +listMapKey=name
	// +optional
	InstanceSets []PostgresInstanceSetStatus `json:"instances,omitempty"`

	// +optional
	Patroni *PatroniStatus `json:"patroni,omitempty"`

	// Status information for pgBackRest
	// +optional
	PGBackRest *PGBackRestStatus `json:"pgbackrest,omitempty"`

	// Current state of the PostgreSQL proxy.
	// +optional
	Proxy PostgresProxyStatus `json:"proxy,omitempty"`

	// The instance that should be started first when bootstrapping and/or starting a
	// PostgresCluster.
	// +optional
	StartupInstance string `json:"startupInstance,omitempty"`

	// The instance set associated with the startupInstance
	// +optional
	StartupInstanceSet string `json:"startupInstanceSet,omitempty"`

	// Identifies the users that have been installed into PostgreSQL.
	UsersRevision string `json:"usersRevision,omitempty"`

	// Current state of PostgreSQL cluster monitoring tool configuration
	// +optional
	Monitoring MonitoringStatus `json:"monitoring,omitempty"`

	// observedGeneration represents the .metadata.generation on which the status was based.
	// +optional
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// conditions represent the observations of postgrescluster's current state.
	// Known .status.conditions.type are: "PersistentVolumeResizing",
	// "ProxyAvailable"
	// +optional
	// +listType=map
	// +listMapKey=type
	// +operator-sdk:csv:customresourcedefinitions:type=status,xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// PostgresClusterStatus condition types.
const (
	PersistentVolumeResizing = "PersistentVolumeResizing"
	ProxyAvailable           = "ProxyAvailable"
)

type PostgresInstanceSetSpec struct {
	// +optional
	Metadata *Metadata `json:"metadata,omitempty"`

	// +optional
	// +kubebuilder:default=""
	Name string `json:"name"`

	// Scheduling constraints of a PostgreSQL pod. Changing this value causes
	// PostgreSQL to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`

	// Defines a PersistentVolumeClaim for PostgreSQL data.
	// More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes
	// +kubebuilder:validation:Required
	DataVolumeClaimSpec corev1.PersistentVolumeClaimSpec `json:"dataVolumeClaimSpec"`

	// +optional
	// +kubebuilder:default=1
	// +kubebuilder:validation:Minimum=1
	Replicas *int32 `json:"replicas,omitempty"`

	// Compute resources of a PostgreSQL container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Tolerations of a PostgreSQL pod. Changing this value causes PostgreSQL to restart.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Defines a separate PersistentVolumeClaim for PostgreSQL's write-ahead log.
	// More info: https://www.postgresql.org/docs/current/wal.html
	// +optional
	WALVolumeClaimSpec *corev1.PersistentVolumeClaimSpec `json:"walVolumeClaimSpec,omitempty"`
}

// Default sets the default values for an instance set spec, including the name
// suffix and number of replicas.
func (s *PostgresInstanceSetSpec) Default(i int) {
	if s.Name == "" {
		s.Name = fmt.Sprintf("%02d", i)
	}
	if s.Replicas == nil {
		s.Replicas = new(int32)
		*s.Replicas = 1
	}
}

type PostgresInstanceSetStatus struct {
	Name string `json:"name"`

	// Total number of ready pods.
	// +optional
	ReadyReplicas int32 `json:"readyReplicas,omitempty"`

	// Total number of non-terminated pods.
	// +optional
	Replicas int32 `json:"replicas,omitempty"`

	// Total number of non-terminated pods that have the desired specification.
	// +optional
	UpdatedReplicas int32 `json:"updatedReplicas,omitempty"`
}

// PostgresProxySpec is a union of the supported PostgreSQL proxies.
type PostgresProxySpec struct {

	// Defines a PgBouncer proxy and connection pooler.
	PGBouncer *PGBouncerPodSpec `json:"pgBouncer"`
}

// Default sets the defaults for any proxies that are set.
func (s *PostgresProxySpec) Default() {
	if s.PGBouncer != nil {
		s.PGBouncer.Default()
	}
}

type PostgresProxyStatus struct {
	PGBouncer PGBouncerPodStatus `json:"pgBouncer,omitempty"`
}

// PostgresStandbySpec defines if/how the cluster should be a hot standby.
type PostgresStandbySpec struct {
	// Whether or not the PostgreSQL cluster should be read-only. When this is
	// true, WAL files are applied from the pgBackRest repository.
	// +optional
	// +kubebuilder:default=true
	Enabled bool `json:"enabled"`

	// The name of the pgBackRest repository to follow for WAL files.
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Pattern=^repo[1-4]
	RepoName string `json:"repoName"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +operator-sdk:csv:customresourcedefinitions:resources={{ConfigMap,v1},{Secret,v1},{Service,v1},{CronJob,v1beta1},{Deployment,v1},{Job,v1},{StatefulSet,v1},{PersistentVolumeClaim,v1}}

// PostgresCluster is the Schema for the postgresclusters API
type PostgresCluster struct {
	// ObjectMeta.Name is a DNS subdomain.
	// - https://docs.k8s.io/concepts/overview/working-with-objects/names/#dns-subdomain-names
	// - https://releases.k8s.io/v1.21.0/staging/src/k8s.io/apiextensions-apiserver/pkg/registry/customresource/validator.go#L60

	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// NOTE(cbandy): Every PostgresCluster needs a Spec, but it is optional here
	// so ObjectMeta can be managed independently.

	Spec   PostgresClusterSpec   `json:"spec,omitempty"`
	Status PostgresClusterStatus `json:"status,omitempty"`
}

// Default implements "sigs.k8s.io/controller-runtime/pkg/webhook.Defaulter" so
// a webhook can be registered for the type.
// - https://book.kubebuilder.io/reference/webhook-overview.html
func (c *PostgresCluster) Default() {
	if len(c.APIVersion) == 0 {
		c.APIVersion = GroupVersion.String()
	}
	if len(c.Kind) == 0 {
		c.Kind = "PostgresCluster"
	}
	c.Spec.Default()
}

// +kubebuilder:object:root=true

// PostgresClusterList contains a list of PostgresCluster
type PostgresClusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PostgresCluster `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PostgresCluster{}, &PostgresClusterList{})
}

// Metadata contains metadata for PostgresCluster resources
type Metadata struct {
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GetLabelsOrNil gets labels from a Metadata pointer, if Metadata
// hasn't been set return nil
func (meta *Metadata) GetLabelsOrNil() map[string]string {
	if meta == nil {
		return nil
	}
	return meta.Labels
}

// GetAnnotationsOrNil gets annotations from a Metadata pointer, if Metadata
// hasn't been set return nil
func (meta *Metadata) GetAnnotationsOrNil() map[string]string {
	if meta == nil {
		return nil
	}
	return meta.Annotations
}

// MonitoringSpec is a union of the supported PostgreSQL Monitoring tools
type MonitoringSpec struct {
	// +optional
	PGMonitor *PGMonitorSpec `json:"pgmonitor,omitempty"`
}

// MonitoringStatus is the current state of PostgreSQL cluster monitoring tool
// configuration
type MonitoringStatus struct {
	// +optional
	ExporterConfiguration string `json:"exporterConfiguration,omitempty"`
}

// PGMonitorSpec defines the desired state of the pgMonitor tool suite
type PGMonitorSpec struct {
	// +optional
	Exporter *ExporterSpec `json:"exporter,omitempty"`
}

type ExporterSpec struct {

	// Projected volumes containing custom PostgreSQL Exporter configuration.  Currently supports
	// the customization of PostgreSQL Exporter queries. If a "queries.yaml" file is detected in
	// any volume projected using this field, it will be loaded using the "extend.query-path" flag:
	// https://github.com/prometheus-community/postgres_exporter#flags
	// Changing the values of field causes PostgreSQL and the exporter to restart.
	// +optional
	Configuration []corev1.VolumeProjection `json:"configuration,omitempty"`

	// The image name to use for crunchy-postgres-exporter containers. The image may
	// also be set using the RELATED_IMAGE_PGEXPORTER environment variable.
	// +optional
	Image string `json:"image,omitempty"`

	// Changing this value causes PostgreSQL and the exporter to restart.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`
}
