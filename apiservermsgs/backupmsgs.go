package apiservermsgs

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

// CreateBackupResponse ...
type CreateBackupResponse struct {
	Results []string
	Status
}

// CreateBackupRequest ...
type CreateBackupRequest struct {
	Namespace     string
	Args          []string
	Selector      string
	PVCName       string
	StorageConfig string
	BackupOpts    string
}

// ShowBackupResponse ...
type ShowBackupResponse struct {
	BackupList crv1.PgbackupList
	Status
}

// DeleteBackupResponse ...
type DeleteBackupResponse struct {
	Results []string
	Status
}

// PgbasebackupRestoreRequest ...
type PgbasebackupRestoreRequest struct {
	Namespace   string
	FromCluster string
	ToPVC       string
	FromPVC     string
	BackupPath  string
	NodeLabel   string
}

// PgbasebackupRestoreResponse ...
type PgbasebackupRestoreResponse struct {
	Results []string
	Status
}
