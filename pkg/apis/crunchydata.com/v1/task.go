package v1

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgtaskResourcePlural ...
const PgtaskResourcePlural = "pgtasks"

const (
	PgtaskDeleteData    = "delete-data"
	PgtaskAddPolicies   = "addpolicies"
	PgtaskRollingUpdate = "rolling update"
)

const (
	PgtaskUpgrade           = "clusterupgrade"
	PgtaskUpgradeCreated    = "cluster upgrade - task created"
	PgtaskUpgradeInProgress = "cluster upgrade - in progress"
)

const (
	PgtaskPgAdminAdd    = "add-pgadmin"
	PgtaskPgAdminDelete = "delete-pgadmin"
)

const (
	PgtaskWorkflow                    = "workflow"
	PgtaskWorkflowCreateClusterType   = "createcluster"
	PgtaskWorkflowBackrestRestoreType = "pgbackrestrestore"
	PgtaskWorkflowBackupType          = "backupworkflow"
	PgtaskWorkflowSubmittedStatus     = "task submitted"
	PgtaskWorkflowCompletedStatus     = "task completed"
	PgtaskWorkflowID                  = "workflowid"
)

const (
	PgtaskWorkflowBackrestRestorePVCCreatedStatus     = "restored PVC created"
	PgtaskWorkflowBackrestRestorePrimaryCreatedStatus = "restored Primary created"
	PgtaskWorkflowBackrestRestoreJobCreatedStatus     = "restore job created"
)

const (
	PgtaskBackrest             = "backrest"
	PgtaskBackrestBackup       = "backup"
	PgtaskBackrestInfo         = "info"
	PgtaskBackrestRestore      = "restore"
	PgtaskBackrestStanzaCreate = "stanza-create"
)

const (
	PgtaskpgDump       = "pgdump"
	PgtaskpgDumpBackup = "pgdumpbackup"
	PgtaskpgDumpInfo   = "pgdumpinfo"
	PgtaskpgRestore    = "pgrestore"
)

// this is ported over from legacy backup code
const PgBackupJobSubmitted = "Backup Job Submitted"

// Defines the types of pgBackRest backups that are taken throughout a clusters
// lifecycle
const (
	// this type of backup is taken following a failover event
	BackupTypeFailover string = "failover"
	// this type of backup is taken when a new cluster is being bootstrapped
	BackupTypeBootstrap string = "bootstrap"
)

// PgtaskSpec ...
// swagger:ignore
type PgtaskSpec struct {
	Name        string            `json:"name"`
	StorageSpec PgStorageSpec     `json:"storagespec"`
	TaskType    string            `json:"tasktype"`
	Status      string            `json:"status"`
	Parameters  map[string]string `json:"parameters"`
}

// Pgtask ...
// swagger:ignore
// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Pgtask struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgtaskSpec   `json:"spec"`
	Status PgtaskStatus `json:"status,omitempty"`
}

// PgtaskList ...
// swagger:ignore
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgtaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgtask `json:"items"`
}

// PgtaskStatus ...
// swagger:ignore
type PgtaskStatus struct {
	State   PgtaskState `json:"state,omitempty"`
	Message string      `json:"message,omitempty"`
}

// PgtaskState ...
// swagger:ignore
type PgtaskState string

const (
	// PgtaskStateCreated ...
	PgtaskStateCreated PgtaskState = "pgtask Created"
	// PgtaskStateProcessed ...
	PgtaskStateProcessed PgtaskState = "pgtask Processed"
)
