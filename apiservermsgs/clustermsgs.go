package apiservermsgs

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
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
	Name                string `json:"Name"`
	Namespace           string
	NodeLabel           string
	Password            string
	SecretFrom          string
	UserLabels          string
	Policies            string
	CCPImage            string
	CCPImageTag         string
	Series              int
	ReplicaCount        int
	ServiceType         string
	MetricsFlag         bool
	BadgerFlag          bool
	AutofailFlag        bool
	ArchiveFlag         bool
	BackrestStorageType string
	//BackrestRestoreFrom  string
	PgbouncerFlag        bool
	PgbouncerSecret      string
	PgbouncerPass        string
	PgbouncerUser        string
	CustomConfig         string
	StorageConfig        string
	ReplicaStorageConfig string
	ContainerResources   string
	// Version of API client
	// required: true
	ClientVersion       string
	PodAntiAffinity     string
	SyncReplication     *bool
	BackrestS3Key       string
	BackrestS3KeySecret string
	BackrestS3Bucket    string
	BackrestS3Region    string
	BackrestS3Endpoint  string
}

// CreateClusterResponse
//
// swagger:model
type CreateClusterResponse struct {
	Results []string `json:"results"`
	Status  `json:"status"`
}

// ShowClusterService
//
// swagger:model
type ShowClusterService struct {
	Name         string
	Data         string
	ClusterIP    string
	ExternalIP   string
	ClusterName  string
	Pgbouncer    bool
	BackrestRepo bool
}

const PodTypePrimary = "primary"
const PodTypeReplica = "replica"
const PodTypePgbouncer = "pgbouncer"
const PodTypePgbackrest = "pgbackrest"
const PodTypeBackup = "backup"
const PodTypeUnknown = "unknown"

// ShowClusterPod
//
// swagger:model
type ShowClusterPod struct {
	Name        string
	Phase       string
	NodeName    string
	PVCName     map[string]string
	ReadyStatus string
	Ready       bool
	Primary     bool
	Type        string
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
	Autofail      bool
}

// UpdateClusterResponse ...
// swagger:model
type UpdateClusterResponse struct {
	Results []string
	Status
}

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
}

// ScaleQueryResponse
// swagger:model
type ScaleQueryResponse struct {
	Results []ScaleQueryTargetSpec
	Status
}

// ScaleDownResponse
// swagger:model
type ScaleDownResponse struct {
	Results []string
	Status
}

// ClusterScaleResponse ...
// swagger:model
type ClusterScaleResponse struct {
	Results []string
	Status
}
