package apiservermsgs

/*
Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

// NodeInfo ...
// swagger:model
type NodeInfo struct {
	Name   string
	Status string
	Labels map[string]string
}

// KeyValue ...
// swagger:model
type KeyValue struct {
	Key   string
	Value int
}

// StatusDetail ...
// this aggregated status comes from the pgo-status container
// by means of a volume mounted json blob it generates
// swagger:model
type StatusDetail struct {
	NumDatabases int
	NumClaims    int
	VolumeCap    string
	DbTags       map[string]int
	NotReady     []string
	Nodes        []NodeInfo
	Labels       []KeyValue
}

// ShowClusterResponse ...
// swagger:model
type StatusResponse struct {
	Result StatusDetail
	Status
}
