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

// CreateBackrestBackupResponse ...
type CreateBackrestBackupResponse struct {
	Results []string
	Status
}

// CreateBackrestBackupRequest ...
type CreateBackrestBackupRequest struct {
	Namespace           string
	Args                []string
	Selector            string
	BackupOpts          string
	BackrestStorageType string
}

type ShowBackrestDetail struct {
	Name string
	Info string
}

// ShowBackrestResponse ...
type ShowBackrestResponse struct {
	Items []ShowBackrestDetail
	Status
}

// RestoreResponse ...
type RestoreResponse struct {
	Results []string
	Status
}

// RestoreRequest ...
type RestoreRequest struct {
	Namespace           string
	FromCluster         string
	ToPVC               string
	RestoreOpts         string
	PITRTarget          string
	NodeLabel           string
	BackrestStorageType string
}
