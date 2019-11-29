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

// PgreplicaResourcePlural ..
const PgreplicaResourcePlural = "pgreplicas"

// Pgreplica ..
// swagger:ignore
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Pgreplica struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PgreplicaSpec   `json:"spec"`
	Status            PgreplicaStatus `json:"status,omitempty"`
}

// PgreplicaSpec ...
// swagger:ignore
type PgreplicaSpec struct {
	Namespace          string               `json:"namespace"`
	Name               string               `json:"name"`
	ClusterName        string               `json:"clustername"`
	ReplicaStorage     PgStorageSpec        `json:"replicastorage"`
	ContainerResources PgContainerResources `json:"containerresources"`
	Status             string               `json:"status"`
	UserLabels         map[string]string    `json:"userlabels"`
}

// PgreplicaList ...
// swagger:ignore
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgreplicaList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgreplica `json:"items"`
}

// PgreplicaStatus ...
// swagger:ignore
type PgreplicaStatus struct {
	State   PgreplicaState `json:"state,omitempty"`
	Message string         `json:"message,omitempty"`
}

// PgreplicaState ...
// swagger:ignore
type PgreplicaState string

const (
	// PgreplicaStateCreated ...
	PgreplicaStateCreated PgreplicaState = "pgreplica Created"
	// PgreplicaStatePending ...
	PgreplicaStatePendingInit PgreplicaState = "pgreplica Pending init"
	// PgreplicaStatePendingRestore ...
	PgreplicaStatePendingRestore PgreplicaState = "pgreplica Pending restore"
	// PgreplicaStateProcessed ...
	PgreplicaStateProcessed PgreplicaState = "pgreplica Processed"
)
