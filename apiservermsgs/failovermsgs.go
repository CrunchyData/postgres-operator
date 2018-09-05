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

import ()

type FailoverTargetSpec struct {
	Name            string
	ReadyStatus     string
	Node            string
	RepStatus       string
	ReceiveLocation uint64
	ReplayLocation  uint64
}

// QueryFailoverResponse ...
type QueryFailoverResponse struct {
	Results []string
	Targets []FailoverTargetSpec
	Status
}

// CreateFailoverResponse ...
type CreateFailoverResponse struct {
	Results []string
	Targets string
	Status
}

// CreateFailoverRequest ...
type CreateFailoverRequest struct {
	ClusterName   string
	Target        string
	ClientVersion string
}

// QueryFailoverRequest ...
type QueryFailoverRequest struct {
	ClusterName   string
	ClientVersion string
}
