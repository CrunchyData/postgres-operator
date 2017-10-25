package apiservermsgs

/*
Copyright 2017 Crunchy Data Solutions, Inc.
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
	Name        string
	Namespace   string
	NodeName    string
	Password    string
	SecretFrom  string
	BackupPVC   string
	UserLabels  string
	BackupPath  string
	Policies    string
	CCPImageTag string
	Series      int
}

// CreateClusterResponse ...
type CreateClusterResponse struct {
	Results []string
	Status
}

// ShowClusterService
type ShowClusterService struct {
	Name      string
	ClusterIP string
}

// ShowClusterPod
type ShowClusterPod struct {
	Name        string
	Phase       string
	NodeName    string
	ReadyStatus string
}

// ShowClusterDeployment
type ShowClusterDeployment struct {
	Name         string
	PolicyLabels []string
}

// ShowClusterSecret
type ShowClusterSecret struct {
	Name     string
	Username string
	Password string
}

// ShowClusterDetail ...
type ShowClusterDetail struct {
	Cluster     crv1.Pgcluster
	Deployments []ShowClusterDeployment
	Pods        []ShowClusterPod
	Services    []ShowClusterService
	Secrets     []ShowClusterSecret
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

// ClusterTestResponse ...
type ClusterTestResponse struct {
	Items []ClusterTestDetail
	Status
}

// ClusterScaleResponse ...
type ClusterScaleResponse struct {
	Status
}
