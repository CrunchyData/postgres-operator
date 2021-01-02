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

type DfPVCType int

// DfShowAllSelector is a value that is used to represent "all"
const DfShowAllSelector = "*"

// the DfPVCType selectors help to display determine what type of PVC is being
// analyzed as part of the DF command
const (
	PVCTypePostgreSQL DfPVCType = iota
	PVCTypepgBackRest
	PVCTypeTablespace
	PVCTypeWriteAheadLog
)

// DfRequest contains the parameters that can be used to get disk utilization
// for PostgreSQL clusters
// swagger:model
type DfRequest struct {
	ClientVersion string
	Namespace     string
	Selector      string
}

// DfDetail returns specific information about the utilization of a PVC
// swagger:model
type DfDetail struct {
	InstanceName string
	PodName      string
	PVCType      DfPVCType
	PVCName      string
	PVCUsed      int64
	PVCCapacity  int64
}

// DfResponse returns the results of how PVCs are being utilized, or an error
// message
// swagger:model
type DfResponse struct {
	Results []DfDetail
	Status
}
