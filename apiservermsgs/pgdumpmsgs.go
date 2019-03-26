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

type CreatepgDumpBackupResponse struct {
	Results []string
	Status
}

type CreatepgDumpBackupRequest struct {
	Namespace     string
	Args          []string
	Selector      string
	PVCName       string
	StorageConfig string
	BackupOpts    string
}

type ShowpgDumpDetail struct {
	Name string
	Info string
}

type PgRestoreResponse struct {
	Results []string
	Status
}

type PgRestoreRequest struct {
	Namespace   string
	FromCluster string
	FromPVC     string
	RestoreOpts string
	PITRTarget  string
	NodeLabel   string
}
