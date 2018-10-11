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

// CreateScheduleRequest ...
type CreateScheduleRequest struct {
	ClusterName     string
	Name            string
	Schedule        string
	ScheduleType    string
	Selector        string
	PGBackRestType  string
	PVCName         string
	CCPImageTag     string
	ScheduleOptions string
}

type CreateScheduleResponse struct {
	Results []string
	Status
}

type DeleteScheduleRequest struct {
	ScheduleName string
	ClusterName  string
	Selector     string
}

type ShowScheduleRequest struct {
	ScheduleName string
	ClusterName  string
	Selector     string
}

type DeleteScheduleResponse struct {
	Results []string
	Status
}

type ShowScheduleResponse struct {
	Results []string
	Status
}
