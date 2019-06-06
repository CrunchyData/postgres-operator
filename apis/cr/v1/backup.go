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

// PgbackupResourcePlural ...
const PgbackupResourcePlural = "pgbackups"

// Backup job processing states - used by pgdump
const PgBackupJobReSubmitted = "Backup Job Re-Submitted"
const PgBackupJobSubmitted = "Backup Job Submitted"
const PgBackupJobInProgress = "Backup Job In Progress"
const PgBackupJobCompleted = "Backup Job Completed"
const PgBackupJobError = "Backup Job Error"

// PgbackupSpec ...
type PgbackupSpec struct {
	Namespace        string            `json:"namespace"`
	Name             string            `json:"name"`
	StorageSpec      PgStorageSpec     `json:"storagespec"`
	CCPImageTag      string            `json:"ccpimagetag"`
	BackupHost       string            `json:"backuphost"`
	BackupUserSecret string            `json:"backupusersecret"`
	BackupPort       string            `json:"backupport"`
	BackupStatus     string            `json:"backupstatus"`
	BackupPVC        string            `json:"backuppvc"`
	BackupOpts       string            `json:"backupopts"`
	Toc              map[string]string `json:"toc"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Pgbackup ...
type Pgbackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgbackupSpec   `json:"spec"`
	Status PgbackupStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PgbackupList ...
type PgbackupList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgbackup `json:"items"`
}

// PgbackupStatus ...
type PgbackupStatus struct {
	State   PgbackupState `json:"state,omitempty"`
	Message string        `json:"message,omitempty"`
}

// PgbackupState ...
type PgbackupState string

const (
	// PgbackupStateCreated ...
	PgbackupStateCreated PgbackupState = "pgbackup Created"
	// PgbackupStateProcessed ...
	PgbackupStateProcessed PgbackupState = "pgbackup Processed"
)
