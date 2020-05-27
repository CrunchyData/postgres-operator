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

// FailoverTargetSpec
// swagger:model
type FailoverTargetSpec struct {
	Name           string // the name of the PostgreSQL instance
	Node           string // the node that the instance is running on
	ReplicationLag int    // how far behind the instance is behind the primary, in MB
	Status         string // the current status of the instance
	Timeline       int    // the timeline the replica is on; timelines are adjusted after failover events
	PendingRestart bool   // whether or not a restart is pending for the target
}

// QueryFailoverResponse ...
// swagger:model
type QueryFailoverResponse struct {
	Results []FailoverTargetSpec
	Status
	Standby bool
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
	Namespace     string
	ClusterName   string
	Target        string
	ClientVersion string
}

// QueryFailoverRequest ...
// swagger:model
type QueryFailoverRequest struct {
	ClusterName   string
	ClientVersion string
}
