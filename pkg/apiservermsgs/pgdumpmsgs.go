package apiservermsgs

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
)

// CreatepgDumpBackupResponse ...
// swagger:model
type CreatepgDumpBackupResponse struct {
	Results []string
	Status
}

// CreatepgDumpBackup ...
// swagger:model
type CreatepgDumpBackupRequest struct {
	Namespace     string
	Args          []string
	Selector      string
	PGDumpDB      string
	PVCName       string
	StorageConfig string
	BackupOpts    string
}

// ShowpgDumpDetail
// swagger:model
type ShowpgDumpDetail struct {
	Name string
	Info string
}

// PgRestoreResponse
// swagger:model
type PgRestoreResponse struct {
	Results []string
	Status
}

// PgRestoreRequest ...
// swagger:model
type PgRestoreRequest struct {
	Namespace   string
	FromCluster string
	FromPVC     string
	PGDumpDB    string
	RestoreOpts string
	PITRTarget  string
	// NodeAffinityType is only considered when "NodeLabel" is also set, and is
	// either a value of "preferred" (default) or "required"
	NodeAffinityType crv1.NodeAffinityType
	NodeLabel        string
}

// NOTE: these are ported over from legacy functionality

// ShowBackupResponse ...
// swagger:model
type ShowBackupResponse struct {
	BackupList PgbackupList
	Status
}

// PgbackupList ...
// swagger:model
type PgbackupList struct {
	Items []Pgbackup `json:"items"`
}

// Pgbackup ...
// swagger:model
type Pgbackup struct {
	CreationTimestamp string
	Namespace         string             `json:"namespace"`
	Name              string             `json:"name"`
	StorageSpec       crv1.PgStorageSpec `json:"storagespec"`
	CCPImageTag       string             `json:"ccpimagetag"`
	BackupHost        string             `json:"backuphost"`
	BackupUserSecret  string             `json:"backupusersecret"`
	BackupPort        string             `json:"backupport"`
	BackupStatus      string             `json:"backupstatus"`
	BackupPVC         string             `json:"backuppvc"`
	BackupOpts        string             `json:"backupopts"`
	Toc               map[string]string  `json:"toc"`
}
