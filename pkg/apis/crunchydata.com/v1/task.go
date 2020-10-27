package v1

/*
 Copyright 2017 - 2020 Crunchy Data Solutions, Inc.
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

const PgtaskUpgrade = "clusterupgrade"
const PgtaskUpgradeCreated = "cluster upgrade - task created"
const PgtaskUpgradeInProgress = "cluster upgrade - in progress"

const PgtaskPgAdminAdd = "add-pgadmin"
const PgtaskPgAdminDelete = "delete-pgadmin"

const PgtaskWorkflow = "workflow"
const PgtaskWorkflowCreateClusterType = "createcluster"
const PgtaskWorkflowBackrestRestoreType = "pgbackrestrestore"
const PgtaskWorkflowBackupType = "backupworkflow"
const PgtaskWorkflowSubmittedStatus = "task submitted"
const PgtaskWorkflowCompletedStatus = "task completed"
const PgtaskWorkflowID = "workflowid"

const PgtaskWorkflowBackrestRestorePVCCreatedStatus = "restored PVC created"
const PgtaskWorkflowBackrestRestorePrimaryCreatedStatus = "restored Primary created"
const PgtaskWorkflowBackrestRestoreJobCreatedStatus = "restore job created"

const PgtaskBackrest = "backrest"
const PgtaskBackrestBackup = "backup"
const PgtaskBackrestInfo = "info"
const PgtaskBackrestRestore = "restore"
const PgtaskBackrestStanzaCreate = "stanza-create"

const PgtaskpgDump = "pgdump"
const PgtaskpgDumpBackup = "pgdumpbackup"
const PgtaskpgDumpInfo = "pgdumpinfo"
const PgtaskpgRestore = "pgrestore"

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

// BackrestStorageTypes defines the valid types of storage that can be utilized
// with pgBackRest
var BackrestStorageTypes = []string{"local", "s3"}

// PgtaskSpec ...
// swagger:ignore
type PgtaskSpec struct {
	Namespace   string            `json:"namespace"`
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
