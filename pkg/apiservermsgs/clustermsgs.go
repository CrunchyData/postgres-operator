package apiservermsgs

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

import (
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	v1 "k8s.io/api/core/v1"
)

// ShowClusterRequest shows cluster
//
// swagger:model
type ShowClusterRequest struct {
	// Name of the cluster to show
	// required: true
	Clustername string `json:"clustername"`
	// Selector of the cluster to show
	Selector string `json:"selector"`
	// Image tag of the cluster
	Ccpimagetag string `json:"ccpimagetag"`
	// Version of API client
	// required: true
	ClientVersion string `json:"clientversion"`
	// Namespace to search
	// required: true
	Namespace string `json:"namespace"`
	// Shows all clusters
	AllFlag bool `json:"allflag"`
}

// CreateClusterRequest
//
// swagger:model
type CreateClusterRequest struct {
	Name      string `json:"Name"`
	Namespace string
	// NodeAffinityType is only considered when "NodeLabel" is also set, and is
	// either a value of "preferred" (default) or "required"
	NodeAffinityType    crv1.NodeAffinityType
	NodeLabel           string
	PasswordLength      int
	PasswordSuperuser   string
	PasswordReplication string
	Password            string
	SecretFrom          string
	UserLabels          map[string]string
	Tablespaces         []ClusterTablespaceDetail
	Policies            string
	CCPImage            string
	CCPImageTag         string
	CCPImagePrefix      string
	PGOImagePrefix      string
	ReplicaCount        int
	ServiceType         v1.ServiceType
	MetricsFlag         bool
	// ExporterCPULimit, if specified, is the value of the max CPU for a
	// Crunchy Postgres Exporter sidecar container
	ExporterCPULimit string
	// ExporterCPURequest, if specified, is the value of how much CPU should be
	// requested for a Crunchy Postgres Exporter sidecar container. Defaults to
	// not being requested
	ExporterCPURequest string
	// ExporterMemoryLimit is the value of of the limit of how much RAM a
	// Crunchy Postgres Exporter sidecar container should use
	ExporterMemoryLimit string
	// ExporterMemoryRequest, if specified, is the value of how much RAM should
	// be requested for a Crunchy Postgres Exporter sidecar container. Defaults
	// to the server specified default
	ExporterMemoryRequest string
	// ExporterCPULimit, if specified, is the value of the max amount of CPU
	// to be utilized for a Crunchy Postgres Exporter sidecar container
	BadgerFlag          bool
	AutofailFlag        bool
	ArchiveFlag         bool
	BackrestStorageType string
	// BackrestRestoreFrom  string
	PgbouncerFlag bool
	// PgBouncerReplicas represents the total number of pgBouncer pods to deploy with a
	// PostgreSQL cluster. Only works if PgbouncerFlag is set, and if so, it must
	// be at least 1. If 0 is passed in, it will automatically be set to 1
	PgBouncerReplicas int32
	// PgBouncerServiceType, if specified and if PgbouncerFlag is true, is the
	// ServiceType to use for pgBouncer. If not set, the value is defaultd to that
	// of the PostgreSQL cluster ServiceType.
	PgBouncerServiceType v1.ServiceType
	// PgBouncerTLSSecret is the name of the Secret containing the TLS keypair
	// for enabling TLS with pgBouncer. This also requires for TLSSecret and
	// CASecret to be set
	PgBouncerTLSSecret   string
	CustomConfig         string
	StorageConfig        string
	WALStorageConfig     string
	ReplicaStorageConfig string
	// Version of API client
	// required: true
	ClientVersion             string
	PasswordType              string
	PodAntiAffinity           string
	PodAntiAffinityPgBackRest string
	PodAntiAffinityPgBouncer  string
	SyncReplication           *bool
	BackrestConfig            string
	BackrestGCSBucket         string
	BackrestGCSEndpoint       string
	// BackrestGCSKey is a standard base64 encoded string that contains the
	// information found in a GCS key (typically a JSON file)
	BackrestGCSKey      string
	BackrestGCSKeyType  string
	BackrestS3Key       string
	BackrestS3KeySecret string
	BackrestS3Bucket    string
	BackrestS3Region    string
	BackrestS3Endpoint  string
	BackrestS3URIStyle  string
	BackrestS3VerifyTLS UpdateBackrestS3VerifyTLS
	Standby             bool
	BackrestRepoPath    string

	// allow the user to set custom sizes for PVCs
	// PVCSize applies to the primary/replica storage specs
	PVCSize string
	// BackrestPVCSize applies to the pgBackRest storage spec
	BackrestPVCSize string
	// WALPVCSize applies to the WAL storage spec
	WALPVCSize string

	// Username is an optional parameter that allows the user to override the
	// default user name to use for the PostgreSQL cluster
	Username string
	// ShowSystemAccounts is an optional parameter than when set to true, will
	// also show the results of the available system accounts (e.g. the PostgreSQL
	// superuser)
	ShowSystemAccounts bool
	// Database is an optional parameter that allows the user to specify the name
	// of the initial database that is created
	Database string
	// TLSOnly indicates that a PostgreSQL cluster should be deployed with only
	// TLS connections accepted. Requires that TLSSecret and CASecret are set
	TLSOnly bool
	// TLSSecret is the name of the secret that contains the keypair required to
	// deploy a TLS-enabled PostgreSQL cluster
	TLSSecret string
	// CASecret is the name of the secret that contains the CA to use along with
	// the TLS keypair for deploying a TLS-enabled PostgreSQL cluster
	CASecret string
	// ReplicationTLSSecret is the name of the secret that contains the keypair
	// used for having instances in a PostgreSQL cluster authenticate each another
	// using certificate-based authentication. The CN of the certificate must
	// either be "primaryuser" (the current name of the replication user) OR
	// have a mapping to primaryuser in the pg_ident file. The
	// ReplicationTLSSecret must be verifable by the certificate chain in the
	// CASecret
	ReplicationTLSSecret string
	// CPULimit is the value of the max CPU utilization for a Pod that has a
	// PostgreSQL cluster
	CPULimit string
	// CPURequest is the value of how much CPU should be requested for deploying
	// the PostgreSQL cluster
	CPURequest string
	// MemoryLimit is the value of of the limit of how much RAM a Pod with a
	// PostgreSQL instance should use. At this time we do not recommend setting
	// this.
	MemoryLimit string
	// MemoryRequest is the value of how much RAM should be requested for
	// deploying the PostgreSQL cluster
	MemoryRequest string
	// PgBouncerCPULimit, if specified, is the value of the max CPU for a
	// pgBouncer pod
	PgBouncerCPULimit string
	// PgBouncerCPURequest, if specified, is the value of how much CPU should be
	// requested for deploying pgBouncer instances. Defaults to not being
	// requested
	PgBouncerCPURequest string
	// PgBouncerMemoryLimit is the value of of the limit of how much RAM a Pod
	// with a pgBouncer should use
	PgBouncerMemoryLimit string
	// PgBouncerMemoryRequest, if specified, is the value of how much RAM should
	// be requested for deploying pgBouncer instances. Defaults to the server
	// specified default
	PgBouncerMemoryRequest string
	// BackrestCPULimit, if specified, is the value of the max amount of CPU
	// to be utilized for a pgBackRest Pod
	BackrestCPULimit string
	// BackrestCPURequest, if specified, is the value of how much CPU should be
	// requested the pgBackRest repository. Defaults to not being requested
	BackrestCPURequest string
	// BackrestMemoryLimit, if specified is the max amount of memory a pgBackRest
	// Pod should use
	BackrestMemoryLimit string
	// BackrestMemoryRequest, if specified, is the value of how much RAM should
	// be requested for the pgBackRest repository. Defaults to the server
	// specified default
	BackrestMemoryRequest string
	// BackrestStorageConfig sets the storage configuration to use for the
	// pgBackRest local repository. This overrides the value in pgo.yaml, though
	// the value of BackrestPVCSize can override the PVC size set in this
	// storage config
	BackrestStorageConfig string
	// BackrestS3CASecretName specifies the name of a secret to use for the
	// pgBackRest S3 CA instead of the default
	BackrestS3CASecretName string
	// PGDataSourceSpec defines the data source that should be used to populate the initial PGDATA
	// directory when bootstrapping a new PostgreSQL cluster
	PGDataSource crv1.PGDataSourceSpec
	// Annotations provide any custom annotations for a cluster
	Annotations crv1.ClusterAnnotations `json:"annotations"`
	// Tolerations allows for the setting of Pod tolerations on Postgres instances
	Tolerations []v1.Toleration `json:"tolerations"`
}

// CreateClusterDetail provides details about the PostgreSQL cluster that is
// created
//
// swagger:model
type CreateClusterDetail struct {
	// Database is the name of the database that is initially created for users to
	// connect to
	Database string
	// Name is the name of the PostgreSQL cluster
	Name string
	// Users contain an array of users along with their credentials
	Users []CreateClusterDetailUser
	// WorkflowID matches up to the WorkflowID of the cluster
	WorkflowID string
}

// CreateClusterDetailUser provides information about an individual PostgreSQL
// user, such as password
//
// swagger:model
type CreateClusterDetailUser struct {
	// Password is the password used for this username, but it may be empty based
	// on what data is allowed to be returned by the server
	Password string
	// Username is the username in PostgreSQL for the user
	Username string
}

// CreateClusterResponse
//
// swagger:model
type CreateClusterResponse struct {
	Result CreateClusterDetail `json:"result"`
	Status `json:"status"`
}

// ShowClusterService
//
// swagger:model
type ShowClusterService struct {
	Name         string
	Data         string
	ClusterIP    string
	ClusterPorts []string
	ExternalIP   string
	ClusterName  string
	Pgbouncer    bool
	BackrestRepo bool
	PGAdmin      bool
}

const (
	PodTypePrimary    = "primary"
	PodTypeReplica    = "replica"
	PodTypePgbouncer  = "pgbouncer"
	PodTypePgbackrest = "pgbackrest"
	PodTypeBackup     = "backup"
	PodTypeUnknown    = "unknown"
)

// ShowClusterPod
//
// swagger:model
type ShowClusterPod struct {
	Name        string
	Phase       string
	NodeName    string
	PVC         []ShowClusterPodPVC
	ReadyStatus string
	Ready       bool
	Primary     bool
	Type        string
}

// ShowClusterPodPVC contains information about a PVC that is bound to a Pod
//
// swagger:model
type ShowClusterPodPVC struct {
	// Capacity is the total storage capacity available. This comes from a
	// Kubernetes resource Quantity string
	Capacity string

	// Name is the name of the PVC
	Name string
}

// ShowClusterDeployment
//
// swagger:model
type ShowClusterDeployment struct {
	Name         string
	PolicyLabels []string
}

// ShowClusterReplica
//
// swagger:model
type ShowClusterReplica struct {
	Name string
}

// ShowClusterDetail ...
//
// swagger:model
type ShowClusterDetail struct {
	// Defines the Cluster using a Crunchy Pgcluster crd
	Cluster     crv1.Pgcluster `json:"cluster"`
	Deployments []ShowClusterDeployment
	Pods        []ShowClusterPod
	Services    []ShowClusterService
	Replicas    []ShowClusterReplica
	Standby     bool
}

// ShowClusterResponse ...
//
// swagger:model
type ShowClusterResponse struct {
	// results from show cluster
	Results []ShowClusterDetail
	// status of response
	Status
}

// DeleteClusterRequest ...
// swagger:model
type DeleteClusterRequest struct {
	Clustername string
	Selector    string
	// Version of API client
	// required: true
	ClientVersion string
	Namespace     string
	AllFlag       bool
	DeleteBackups bool
	DeleteData    bool
}

// DeleteClusterResponse ...
// swagger:model
type DeleteClusterResponse struct {
	Results []string
	Status
}

// set the types for updating the Autofail status
type UpdateClusterAutofailStatus int

// set the different values around updating the autofail configuration
const (
	UpdateClusterAutofailDoNothing UpdateClusterAutofailStatus = iota
	UpdateClusterAutofailEnable
	UpdateClusterAutofailDisable
)

// UpdateClusterMetrics determines whether or not to enable/disable the metrics
// collection sidecar in a cluster
type UpdateClusterMetrics int

const (
	UpdateClusterMetricsDoNothing UpdateClusterMetrics = iota
	UpdateClusterMetricsEnable
	UpdateClusterMetricsDisable
)

// UpdateClusterPGBadger determines whether or not to enable/disable the
// pgBadger sidecar in a cluster
type UpdateClusterPGBadger int

const (
	UpdateClusterPGBadgerDoNothing UpdateClusterPGBadger = iota
	UpdateClusterPGBadgerEnable
	UpdateClusterPGBadgerDisable
)

// UpdateClusterStandbyStatus defines the types for updating the Standby status
type UpdateClusterStandbyStatus int

// set the different values around updating the standby configuration
const (
	UpdateClusterStandbyDoNothing UpdateClusterStandbyStatus = iota
	UpdateClusterStandbyEnable
	UpdateClusterStandbyDisable
)

// UpdateBackrestS3VerifyTLS defines the types for updating the S3 TLS verification configuration
type UpdateBackrestS3VerifyTLS int

// set the different values around updating the S3 TLS verification configuration
const (
	UpdateBackrestS3VerifyTLSDoNothing UpdateBackrestS3VerifyTLS = iota
	UpdateBackrestS3VerifyTLSEnable
	UpdateBackrestS3VerifyTLSDisable
)

// UpdateClusterRequest ...
// swagger:model
type UpdateClusterRequest struct {
	Clustername []string
	Selector    string
	// Version of API client
	// required: true
	ClientVersion string
	Namespace     string
	AllFlag       bool
	// Annotations provide any custom annotations for a cluster
	Annotations crv1.ClusterAnnotations `json:"annotations"`
	Autofail    UpdateClusterAutofailStatus
	// BackrestCPULimit, if specified, is the value of the max amount of CPU
	// to be utilized for a pgBackRest Pod
	BackrestCPULimit string
	// BackrestCPURequest, if specified, is the value of how much CPU should be
	// requested the pgBackRest repository. Defaults to not being requested
	BackrestCPURequest string
	// BackrestMemoryLimit, if specified is the max amount of memory a pgBackRest
	// Pod should use
	BackrestMemoryLimit string
	// BackrestMemoryRequest, if specified, is the value of how much RAM should
	// be requested for the pgBackRest repository.
	BackrestMemoryRequest string
	// BackrestPVCSize if set updates the size of the pgBackRest PVC
	BackrestPVCSize string
	// CASecret is the name of the secret that contains the CA to use along with
	// the TLS keypair for deploying a TLS-enabled PostgreSQL cluster. Provide it
	// to either enable or update the Secret used in a TLS cluster
	CASecret string
	// DisableTLS, if set, will remove TLS settings from the cluster. This will
	// override the values of CASecret, ReplicationTLSSecret, TLSSecret, and
	// TLSOnly
	DisableTLS bool
	// ExporterCPULimit, if specified, is the value of the max amount of CPU
	// to be utilized for a Crunchy Postgres Exporter instance
	ExporterCPULimit string
	// ExporterCPURequest, if specified, is the value of how much CPU should be
	// requested the Crunchy Postgres Exporter. Defaults to not being requested
	ExporterCPURequest string
	// ExporterMemoryLimit, if specified is the max amount of memory a Crunchy
	// Postgres Exporter instance should use
	ExporterMemoryLimit string
	// ExporterMemoryRequest, if specified, is the value of how much RAM should
	// be requested for the Crunchy Postgres Exporter instance.
	ExporterMemoryRequest string
	// ExporterRotatePassword, if specified, rotates the password of the metrics
	// collection agent, i.e. the "ccp_monitoring" user.
	ExporterRotatePassword bool
	// CPULimit is the value of the max CPU utilization for a Pod that has a
	// PostgreSQL cluster
	CPULimit string
	// CPURequest is the value of how much CPU should be requested for deploying
	// the PostgreSQL cluster
	CPURequest string
	// MemoryLimit is the value of of the limit of how much RAM a Pod with a
	// PostgreSQL instance should use. At this time we do not recommend setting
	// this.
	MemoryLimit string
	// MemoryRequest is the value of how much RAM should be requested for
	// deploying the PostgreSQL cluster
	MemoryRequest string
	// Metrics allows for the enabling/disabling of the metrics sidecar. This can
	// cause downtime and triggers a rolling update
	Metrics UpdateClusterMetrics
	// PGBadger allows for the enabling/disabling of the pgBadger sidecar. This can
	// cause downtime and triggers a rolling update
	PGBadger UpdateClusterPGBadger
	// PVC size, if set, updates the size of the data directory.
	PVCSize string
	// ReplicationTLSSecret is the name of the secret that contains the keypair
	// used for having instances in a PostgreSQL cluster authenticate each another
	// using certificate-based authentication. The CN of the certificate must
	// either be "primaryuser" (the current name of the replication user) OR
	// have a mapping to primaryuser in the pg_ident file. The
	// ReplicationTLSSecret must be verifable by the certificate chain in the
	// CASecret
	ReplicationTLSSecret string
	// ServiceType, if specified, will change the service type of a cluster.
	ServiceType v1.ServiceType
	Standby     UpdateClusterStandbyStatus
	Startup     bool
	Shutdown    bool
	Tablespaces []ClusterTablespaceDetail
	// TLSSecret is the name of the secret that contains the keypair required to
	// deploy a TLS-enabled PostgreSQL cluster
	TLSSecret string
	// Tolerations allows for the adding of Pod tolerations on a PostgreSQL
	// cluster.
	Tolerations []v1.Toleration `json:"tolerations"`
	// TLSOnly indicates that a PostgreSQL cluster should be deployed with only
	// TLS connections accepted, or if it should be disabled
	TLSOnly UpdateClusterTLSOnly
	// TolerationsDelete  allows for the removal of Pod tolerations on a
	// PostgreSQL cluster
	TolerationsDelete []v1.Toleration `json:"tolerationsDelete"`
	// WALPVCSize updates the size of the WAL PVC, if there is a WAL PVC
	WALPVCSize string
}

// UpdateClusterResponse ...
// swagger:model
type UpdateClusterResponse struct {
	Results []string
	Status
}

type UpdateClusterTLSOnly int

const (
	UpdateClusterTLSOnlyDoNothing UpdateClusterTLSOnly = iota
	UpdateClusterTLSOnlyEnable
	UpdateClusterTLSOnlyDisable
)

// ClusterTestRequest ...
// swagger:model
type ClusterTestRequest struct {
	Clustername string
	Selector    string
	// Version of API client
	// required: true
	ClientVersion string
	Namespace     string
	AllFlag       bool
}

// a collection of constants used to enumerate the output for
// ClusterTestDetail => InstanceType
const (
	ClusterTestInstanceTypePrimary   = "primary"
	ClusterTestInstanceTypeReplica   = "replica"
	ClusterTestInstanceTypePGAdmin   = "pgadmin"
	ClusterTestInstanceTypePGBouncer = "pgbouncer"
	ClusterTestInstanceTypeBackups   = "backups"
	ClusterTestInstanceTypeUnknown   = "unknown"
)

// ClusterTestDetail provides the output of an individual test that is performed
// on either a PostgreSQL instance (i.e. pod) or a service endpoint that is used
// to connect to the instances

// swagger:model
type ClusterTestDetail struct {
	Available    bool   // true if the object being tested is available (ready)
	Message      string // a descriptive message that can be displayed with
	InstanceType string // an enumerated set of what this instance can be, e.g. "primary"
}

// ClusterTestResult contains the output for a test on a single PostgreSQL
// cluster. This includes the endpoints (i.e. how to connect to instances
// in a cluster) and the instances themselves (which are pods)
// swagger:model
type ClusterTestResult struct {
	ClusterName string
	Endpoints   []ClusterTestDetail // a list of endpoints
	Instances   []ClusterTestDetail // a list of instances (pods)
}

// ClusterTestResponse ...
// swagger:model
type ClusterTestResponse struct {
	Results []ClusterTestResult
	Status
}

// ScaleQueryTargetSpec
// swagger:model
type ScaleQueryTargetSpec struct {
	Name           string // the name of the PostgreSQL instance
	Node           string // the node that the instance is running on
	ReplicationLag int    // how far behind the instance is behind the primary, in MB
	Status         string // the current status of the instance
	Timeline       int    // the timeline the replica is on; timelines are adjusted after failover events
	PendingRestart bool   // whether or not a restart is pending for the target
}

// ScaleQueryResponse
// swagger:model
type ScaleQueryResponse struct {
	Results []ScaleQueryTargetSpec
	Status
	Standby bool
}

// ScaleDownResponse
// swagger:model
type ScaleDownResponse struct {
	Results []string
	Status
}

// ClusterScaleRequest superimposes on the legacy model of handling the ability
// to scale up the number of instances on a cluster
// swagger:model
type ClusterScaleRequest struct {
	// CCPImageTag is the image tag to use for cluster creation. If this is not
	// provided, this defaults to what the cluster is using, which is likely
	// the preferred behavior at this point.
	CCPImageTag string `json:"ccpImageTag"`
	// ClientVersion is the version of the client that is being used
	ClientVersion string `json:"clientVersion"`
	// Name is the name of the cluster to scale. This is set by the value in the
	// URL
	Name string `json:"name"`
	// Namespace is the namespace in which the queried cluster resides.
	Namespace string `json:"namespace"`
	// NodeAffinityType is only considered when "NodeLabel" is also set, and is
	// either a value of "preferred" (default) or "required"
	NodeAffinityType crv1.NodeAffinityType
	// NodeLabel if provided is a node label to use.
	NodeLabel string `json:"nodeLabel"`
	// ReplicaCount is the number of replicas to add to the cluster. This is
	// required and should be at least 1.
	ReplicaCount int `json:"replicaCount"`
	// ServiceType is the kind of Service to deploy with this instance. Defaults
	// to the value on the cluster.
	ServiceType v1.ServiceType `json:"serviceType"`
	// StorageConfig, if provided, specifies which of the storage configuration
	// options should be used. Defaults to what the main cluster definition uses.
	StorageConfig string `json:"storageConfig"`
	// Tolerations allows for the setting of Pod tolerations on Postgres instances
	Tolerations []v1.Toleration `json:"tolerations"`
}

// ClusterScaleResponse ...
// swagger:model
type ClusterScaleResponse struct {
	Results []string
	Status
}

// ClusterTablespaceDetail contains details required to create a tablespace
// swagger:model
type ClusterTablespaceDetail struct {
	// Name is the name of the tablespace. Becomes the name of the tablespace in
	// PostgreSQL
	Name string
	// optional: allows for the specification of the size of the PVC for the
	// tablespace, overriding the value that is in "StorageClass"
	PVCSize string
	// StorageConfig is the name of the storage config to use for the tablespace,
	// e.g. "nfsstorage", that is specified in the pgo.yaml configuration
	StorageConfig string
}
