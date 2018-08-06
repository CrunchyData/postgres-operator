package apiservermsgs

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

// CreateClusterRequest ...
type CreateClusterRequest struct {
	Name                 string
	Namespace            string
	NodeLabel            string
	Password             string
	SecretFrom           string
	BackupPVC            string
	UserLabels           string
	BackupPath           string
	Policies             string
	CCPImageTag          string
	Series               int
	ServiceType          string
	MetricsFlag          bool
	BadgerFlag           bool
	AutofailFlag         bool
	ArchiveFlag          bool
	BackrestFlag         bool
	PgpoolFlag           bool
	PgpoolSecret         string
	CustomConfig         string
	StorageConfig        string
	ReplicaStorageConfig string
	ContainerResources   string
	ClientVersion        string
}

// CreateClusterResponse ...
type CreateClusterResponse struct {
	Results []string
	Status
}

// ShowClusterService
type ShowClusterService struct {
	Name       string
	Data       string
	ClusterIP  string
	ExternalIP string
}

const PodTypePrimary = "primary"
const PodTypeReplica = "replica"
const PodTypeBackup = "backup"
const PodTypeUnknown = "unknown"

// ShowClusterPod
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
type ShowClusterDeployment struct {
	Name         string
	PolicyLabels []string
}

// ShowClusterReplica
type ShowClusterReplica struct {
	Name string
}

// ShowClusterDetail ...
type ShowClusterDetail struct {
	Cluster     crv1.Pgcluster
	Deployments []ShowClusterDeployment
	Pods        []ShowClusterPod
	Services    []ShowClusterService
	Replicas    []ShowClusterReplica
}

// ShowClusterResponse ...
type ShowClusterResponse struct {
	Results []ShowClusterDetail
	Status
}

// DeleteClusterResponse ...
type DeleteClusterResponse struct {
	Results []string
	Status
}

// ClusterTestDetail ...
type ClusterTestDetail struct {
	PsqlString string
	Working    bool
}

// ClusterTestResult ...
type ClusterTestResult struct {
	ClusterName string
	Items       []ClusterTestDetail
}

// ClusterTestResponse ...
type ClusterTestResponse struct {
	Results []ClusterTestResult
	Status
}

type ScaleQueryTargetSpec struct {
	Name        string
	ReadyStatus string
	Node        string
	RepStatus   string
}

type ScaleQueryResponse struct {
	Results []string
	Targets []ScaleQueryTargetSpec
	Status
}

type ScaleDownResponse struct {
	Results []string
	Status
}

// ClusterScaleResponse ...
type ClusterScaleResponse struct {
	Results []string
	Status
}
