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

// CreateScheduleRequest ...
// swagger:model
type CreateScheduleRequest struct {
	ClusterName         string
	Name                string
	Namespace           string
	Schedule            string
	ScheduleType        string
	Selector            string
	PGBackRestType      string
	BackrestStorageType string
	PVCName             string
	ScheduleOptions     string
	StorageConfig       string
	PolicyName          string
	Database            string
	Secret              string
}

// CreateScheduleResponse ...
// swagger:model
type CreateScheduleResponse struct {
	Results []string
	Status
}

// DeleteScheduleResponse ...
// swagger:model
type DeleteScheduleRequest struct {
	Namespace    string
	ScheduleName string
	ClusterName  string
	Selector     string
}

// ShowScheduleRequest ...
// swagger:model
type ShowScheduleRequest struct {
	Namespace    string
	ScheduleName string
	ClusterName  string
	Selector     string
}

// DeleteScheduleResponse ...
// swagger:model
type DeleteScheduleResponse struct {
	Results []string
	Status
}

// ShowSchewduleResponse ...
// swagger:model
type ShowScheduleResponse struct {
	Results []string
	Status
}
