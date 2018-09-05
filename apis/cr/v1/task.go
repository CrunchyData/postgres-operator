package v1

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgtaskResourcePlural ...
const PgtaskResourcePlural = "pgtasks"

const PgtaskDeleteBackups = "delete-backups"
const PgtaskDeleteData = "delete-data"
const PgtaskFailover = "failover"
const PgtaskAutoFailover = "autofailover"
const PgtaskAddPolicies = "addpolicies"

const PgtaskBackrest = "backrest"
const PgtaskBackrestBackup = "backup"
const PgtaskBackrestInfo = "info"
const PgtaskBackrestRestore = "restore"

// PgtaskSpec ...
type PgtaskSpec struct {
	Name        string        `json:"name"`
	StorageSpec PgStorageSpec `json:"storagespec"`
	TaskType    string        `json:"tasktype"`
	Status      string        `json:"status"`
	//Parameters  string            `json:"parameters"`
	Parameters map[string]string `json:"parameters"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Pgtask ...
type Pgtask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgtaskSpec   `json:"spec"`
	Status PgtaskStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PgtaskList ...
type PgtaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgtask `json:"items"`
}

// PgtaskStatus ...
type PgtaskStatus struct {
	State   PgtaskState `json:"state,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PgtaskState ...
type PgtaskState string

const (
	// PgtaskStateCreated ...
	PgtaskStateCreated PgtaskState = "Created"
	// PgtaskStateProcessed ...
	PgtaskStateProcessed PgtaskState = "Processed"
)
