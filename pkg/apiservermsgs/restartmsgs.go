package apiservermsgs

/*
Copyright 2020 Crunchy Data Solutions, Inc.
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

// RestartResponse is the response generated for a request to restart a cluster.
// swagger:model
type RestartResponse struct {
	Result RestartDetail
	Status
}

// RestartDetail defines the details for a cluster restart request, specifically
// information about each instance restarted as a result of the request.
// swagger:model
type RestartDetail struct {
	ClusterName  string
	Instances    []InstanceDetail
	Error        bool
	ErrorMessage string
}

// InstanceDetail defines the details of an instance within a cluster restarted as a result
// of a cluster restart request.  This includes the name of each instance, along with any
// errors that may have occurred while attempting to restart an instance.
type InstanceDetail struct {
	InstanceName string
	Error        bool
	ErrorMessage string
}

// RestartRequest defines a request to restart a cluster, or one or more targets (i.e.
// instances) within a cluster
// swagger:model
type RestartRequest struct {
	Namespace     string
	ClusterName   string
	Targets       []string
	ClientVersion string
}

// QueryRestartRequest defines a request to query a specific cluster for available restart targets.
// swagger:model
type QueryRestartRequest struct {
	ClusterName   string
	ClientVersion string
}

// QueryRestartResponse is the response generated when querying the available instances within a
// cluster in order to perform a restart against a specific target.
// swagger:model
type QueryRestartResponse struct {
	Results []RestartTargetSpec
	Status
	Standby bool
}

// RestartTargetSpec defines the details for a specific restart target identified while querying a
// cluster for available targets (i.e. instances).
// swagger:model
type RestartTargetSpec struct {
	Name           string // the name of the PostgreSQL instance
	Node           string // the node that the instance is running on
	ReplicationLag int    // how far behind the instance is behind the primary, in MB
	Status         string // the current status of the instance
	Timeline       int    // the timeline the replica is on; timelines are adjusted after failover events
	PendingRestart bool   // whether or not a restart is pending for the target
	Role           string // the role of the specific instance
}
