
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

import ()

// FailoverTargetSpec ...
// swagger:model
type FailoverTargetSpec struct {
	Name            string
	ReadyStatus     string
	Node            string
	PreferredNode   bool
	RepStatus       string
	ReceiveLocation uint64
	ReplayLocation  uint64
}

// QueryFailoverResponse ...
// swagger:model
type QueryFailoverResponse struct {
	Results []string
	Targets []FailoverTargetSpec
	Status
}

// CreateFailoverResponse ...
// swagger:model
type CreateFailoverResponse struct {
	Results []string
	Targets string
	Status
}

// CreateFailoverRequest ...
// swagger:model
type CreateFailoverRequest struct {
	Namespace              string
	ClusterName            string
	AutofailReplaceReplica string
	Target                 string
	ClientVersion          string
}

// QueryFailoverRequest ...
// swagger:model
type QueryFailoverRequest struct {
	ClusterName   string
	ClientVersion string
}
