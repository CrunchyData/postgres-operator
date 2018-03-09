package apiservermsgs

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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

// CreateIngestRequest ...
type CreateIngestRequest struct {
	Name            string
	WatchDir        string
	DBHost          string
	DBPort          string
	DBName          string
	DBSecret        string
	DBTable         string
	DBColumn        string
	PVCName         string
	SecurityContext string
	MaxJobs         int
}

// CreateIngestResponse ...
type CreateIngestResponse struct {
	Results []string
	Status
}

// ShowIngestResponseDetail ...
type ShowIngestResponseDetail struct {
	Ingest            crv1.Pgingest
	JobCountRunning   int
	JobCountCompleted int
}

// ShowIngestResponse ...
type ShowIngestResponse struct {
	Details []ShowIngestResponseDetail
	Status
}

// DeleteIngestResponse ...
type DeleteIngestResponse struct {
	Results []string
	Status
}
