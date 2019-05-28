package v1

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgclusterResourcePlural ..
const PgclusterResourcePlural = "pgclusters"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Pgcluster ..
type Pgcluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PgclusterSpec   `json:"spec"`
	Status            PgclusterStatus `json:"status,omitempty"`
}

// PgclusterSpec ...
type PgclusterSpec struct {
	Namespace          string               `json:"namespace"`
	Name               string               `json:"name"`
	ClusterName        string               `json:"clustername"`
	Policies           string               `json:"policies"`
	CCPImage           string               `json:"ccpimage"`
	CCPImageTag        string               `json:"ccpimagetag"`
	Port               string               `json:"port"`
	NodeName           string               `json:"nodename"`
	PrimaryStorage     PgStorageSpec        `json:primarystorage`
	ArchiveStorage     PgStorageSpec        `json:archivestorage`
	ReplicaStorage     PgStorageSpec        `json:replicastorage`
	BackrestStorage    PgStorageSpec        `json:backreststorage`
	ContainerResources PgContainerResources `json:containerresources`
	PrimaryHost        string               `json:"primaryhost"`
	User               string               `json:"user"`
	Database           string               `json:"database"`
	Replicas           string               `json:"replicas"`
	Strategy           string               `json:"strategy"`
	SecretFrom         string               `json:"secretfrom"`
	UserSecretName     string               `json:"usersecretname"`
	RootSecretName     string               `json:"rootsecretname"`
	PrimarySecretName  string               `json:"primarysecretname"`
	Status             string               `json:"status"`
	PswLastUpdate      string               `json:"pswlastupdate"`
	CustomConfig       string               `json:"customconfig"`
	UserLabels         map[string]string    `json:"userlabels"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PgclusterList ...
type PgclusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgcluster `json:"items"`
}

// PgclusterStatus ...
type PgclusterStatus struct {
	State   PgclusterState `json:"state,omitempty"`
	Message string         `json:"message,omitempty"`
}

// PgclusterState ...
type PgclusterState string

const (
	// PgclusterStateCreated ...
	PgclusterStateCreated PgclusterState = "pgcluster Created"
	// PgclusterStateProcessed ...
	PgclusterStateProcessed PgclusterState = "pgcluster Processed"
)
