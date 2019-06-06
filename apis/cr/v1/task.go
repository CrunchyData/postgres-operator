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

// PgtaskResourcePlural ...
const PgtaskResourcePlural = "pgtasks"

const PgtaskAddPgbouncer = "add-pgbouncer"
const PgtaskDeletePgbouncer = "delete-pgbouncer"
const PgtaskReconfigurePgbouncer = "reconfigure-pgbouncer"
const PgtaskUpdatePgbouncerAuths = "update-pgbouncer-auths"
const PgtaskAddPgpool = "add-pgpool"
const PgtaskDeletePgpool = "delete-pgpool"
const PgtaskReconfigurePgpool = "reconfigure-pgpool"
const PgtaskDeleteBackups = "delete-backups"
const PgtaskDeleteData = "delete-data"
const PgtaskFailover = "failover"
const PgtaskAutoFailover = "autofailover"
const PgtaskAddPolicies = "addpolicies"
const PgtaskMinorUpgrade = "minorupgradecluster"

const PgtaskWorkflow = "workflow"
const PgtaskWorkflowCreateClusterType = "createcluster"
const PgtaskWorkflowCreateBenchmarkType = "createbenchmark"
const PgtaskWorkflowBackrestRestoreType = "pgbackrestrestore"
const PgtaskWorkflowPgbasebackupRestoreType = "pgbasebackuprestore"
const PgtaskWorkflowBackupType = "backupworkflow"
const PgtaskWorkflowSubmittedStatus = "task submitted"
const PgtaskWorkflowCompletedStatus = "task completed"
const PgtaskWorkflowID = "workflowid"

const PgtaskWorkflowBackrestRestorePVCCreatedStatus = "restored PVC created"
const PgtaskWorkflowBackrestRestorePrimaryCreatedStatus = "restored Primary created"
const PgtaskWorkflowBackrestRestoreJobCreatedStatus = "restore job created"

const PgtaskWorkflowPgbasebackupRestorePVCCreatedStatus = "restored PVC created"
const PgtaskWorkflowPgbasebackupRestorePrimaryCreatedStatus = "restored Primary created"
const PgtaskWorkflowPgbasebackupRestoreJobCreatedStatus = "restore job created"

const PgtaskBackrest = "backrest"
const PgtaskBackrestBackup = "backup"
const PgtaskBackrestInfo = "info"
const PgtaskBackrestRestore = "restore"
const PgtaskBackrestStanzaCreate = "stanza-create"

const PgtaskpgDump = "pgdump"
const PgtaskpgDumpBackup = "pgdumpbackup"
const PgtaskpgDumpInfo = "pgdumpinfo"
const PgtaskpgRestore = "pgrestore"

const PgtaskpgBasebackupRestore = "pgbasebackuprestore"

const PgtaskBenchmark = "benchmark"

// PgtaskSpec ...
type PgtaskSpec struct {
	Namespace   string            `json:"namespace"`
	Name        string            `json:"name"`
	StorageSpec PgStorageSpec     `json:"storagespec"`
	TaskType    string            `json:"tasktype"`
	Status      string            `json:"status"`
	Parameters  map[string]string `json:"parameters"`
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
	PgtaskStateCreated PgtaskState = "pgtask Created"
	// PgtaskStateProcessed ...
	PgtaskStateProcessed PgtaskState = "pgtask Processed"
)
