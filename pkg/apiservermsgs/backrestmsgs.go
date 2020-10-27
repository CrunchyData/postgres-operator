package apiservermsgs

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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

// CreateBackrestBackupResponse ...
// swagger:model
type CreateBackrestBackupResponse struct {
	Results []string
	Status
}

// CreateBackrestBackupRequest ...
// swagger:model
type CreateBackrestBackupRequest struct {
	Namespace           string
	Args                []string
	Selector            string
	BackupOpts          string
	BackrestStorageType string
}

// PgBackRestInfo and its associated structs are available for parsing the info
// that comes from the output of the "pgbackrest info --output json" command
type PgBackRestInfo struct {
	Archives []PgBackRestInfoArchive `json:"archive"`
	Backups  []PgBackRestInfoBackup  `json:"backup"`
	Cipher   string                  `json:"cipher"`
	DBs      []PgBackRestInfoDB      `json:"db"`
	Name     string                  `json:"name"`
	Status   PgBackRestInfoStatus    `json:"status"`
}

type PgBackRestInfoArchive struct {
	DB  PgBackRestInfoDB `json:"db"`
	ID  string           `json:"id"`
	Max string           `json:"max"`
	Min string           `json:"min"`
}

type PgBackRestInfoBackup struct {
	Archive   PgBackRestInfoBackupArchive   `json:"archive"`
	Backrest  PgBackRestInfoBackupBackrest  `json:"backrest"`
	Database  PgBackRestInfoDB              `json:"database"`
	Info      PgBackRestInfoBackupInfo      `json:"info"`
	Label     string                        `json:"label"`
	Prior     string                        `json:"prior"`
	Reference []string                      `json:"reference"`
	Timestamp PgBackRestInfoBackupTimestamp `json:"timestamp"`
	Type      string                        `json:"type"`
}

type PgBackRestInfoBackupArchive struct {
	Start string `json:"start"`
	Stop  string `json:"stop"`
}

type PgBackRestInfoBackupBackrest struct {
	Format  int    `json:"format"`
	Version string `json:"version"`
}

type PgBackRestInfoBackupInfo struct {
	Delta      int64                              `json:"delta"`
	Repository PgBackRestInfoBackupInfoRepository `json:"repository"`
	Size       int64                              `json:"size"`
}

type PgBackRestInfoBackupInfoRepository struct {
	Delta int64 `json:"delta"`
	Size  int64 `json:"size"`
}

type PgBackRestInfoBackupTimestamp struct {
	Start int64 `json:"start"`
	Stop  int64 `json:"stop"`
}

type PgBackRestInfoDB struct {
	ID       int    `json:"id"`
	SystemID int64  `json:"system-id,omitempty"`
	Version  string `json:"version,omitempty"`
}

type PgBackRestInfoStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ShowBackrestDetail ...
// swagger:model
type ShowBackrestDetail struct {
	Name        string
	Info        []PgBackRestInfo
	StorageType string
}

// ShowBackrestResponse ...
// swagger:model
type ShowBackrestResponse struct {
	Items []ShowBackrestDetail
	Status
}

// RestoreResponse ...
// swagger:model
type RestoreResponse struct {
	Results []string
	Status
}

// RestoreRequest ...
// swagger:model
type RestoreRequest struct {
	Namespace           string
	FromCluster         string
	RestoreOpts         string
	PITRTarget          string
	NodeLabel           string
	BackrestStorageType string
}
