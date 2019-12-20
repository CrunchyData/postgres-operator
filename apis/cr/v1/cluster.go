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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgclusterResourcePlural ..
const PgclusterResourcePlural = "pgclusters"

// Pgcluster is the CRD that defines a Crunchy PG Cluster
//
// swagger:ignore Pgcluster
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Pgcluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PgclusterSpec   `json:"spec"`
	Status            PgclusterStatus `json:"status,omitempty"`
}

// PgclusterSpec is the CRD that defines a Crunchy PG Cluster Spec
// swagger:ignore
type PgclusterSpec struct {
	Namespace          string               `json:"namespace"`
	Name               string               `json:"name"`
	ClusterName        string               `json:"clustername"`
	Policies           string               `json:"policies"`
	CCPImage           string               `json:"ccpimage"`
	CCPImageTag        string               `json:"ccpimagetag"`
	Port               string               `json:"port"`
	PGBadgerPort       string               `json:"pgbadgerport"`
	ExporterPort       string               `json:"exporterport"`
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
	CollectSecretName  string               `json:"collectSecretName"`
	Status             string               `json:"status"`
	PswLastUpdate      string               `json:"pswlastupdate"`
	CustomConfig       string               `json:"customconfig"`
	UserLabels         map[string]string    `json:"userlabels"`
	PodAntiAffinity    string               `json:"podPodAntiAffinity"`
	SyncReplication    *bool                `json:"syncReplication"`
	BackrestS3Bucket   string               `json:"backrestS3Bucket"`
	BackrestS3Region   string               `json:"backrestS3Region"`
	BackrestS3Endpoint string               `json:"backrestS3Endpoint"`
}

// PgclusterList is the CRD that defines a Crunchy PG Cluster List
// swagger:ignore
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgclusterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgcluster `json:"items"`
}

// PgclusterStatus is the CRD that defines PG Cluster Status
// swagger:ignore
type PgclusterStatus struct {
	State   PgclusterState `json:"state,omitempty"`
	Message string         `json:"message,omitempty"`
}

// PgclusterState is the crd that defines PG Cluster Stage
// swagger:ignore
type PgclusterState string

// PodAntiAffinityType defines the different types of type of anti-affinity rules applied to pg
// clusters when utilizing the default pod anti-affinity rules provided by the PostgreSQL Operator,
// which are enabled for a new pg cluster by default.  Valid Values include "required" for
// requiredDuringSchedulingIgnoredDuringExecution anti-affinity, "preferred" for
// preferredDuringSchedulingIgnoredDuringExecution anti-affinity, and "disabled" to disable the
// default pod anti-affinity rules for the pg cluster all together.
type PodAntiAffinityType string

const (
	// PgclusterStateCreated ...
	PgclusterStateCreated PgclusterState = "pgcluster Created"
	// PgclusterStateProcessed ...
	PgclusterStateProcessed PgclusterState = "pgcluster Processed"
	// PgclusterStateInitialized ...
	PgclusterStateInitialized PgclusterState = "pgcluster Initialized"
	// PgclusterStateRestore ...
	PgclusterStateRestore PgclusterState = "pgcluster Restoring"

	// PodAntiAffinityRequired results in requiredDuringSchedulingIgnoredDuringExecution for any
	// default pod anti-affinity rules applied to pg custers
	PodAntiAffinityRequired PodAntiAffinityType = "required"

	// PodAntiAffinityPreffered results in preferredDuringSchedulingIgnoredDuringExecution for any
	// default pod anti-affinity rules applied to pg custers
	PodAntiAffinityPreffered PodAntiAffinityType = "preferred"

	// PodAntiAffinityDisabled disables any default pod anti-affinity rules applied to pg custers
	PodAntiAffinityDisabled PodAntiAffinityType = "disabled"
)

// ValidatePodAntiAffinityType is responsible for validating whether or not the type of pod
// anti-affinity specified is valid
func (p PodAntiAffinityType) Validate() error {
	switch p {
	case
		PodAntiAffinityRequired,
		PodAntiAffinityPreffered,
		PodAntiAffinityDisabled:
		return nil
	}
	return fmt.Errorf("Invalid pod anti-affinity type.  Valid values are '%s', '%s' or '%s'",
		PodAntiAffinityRequired, PodAntiAffinityPreffered, PodAntiAffinityDisabled)
}
