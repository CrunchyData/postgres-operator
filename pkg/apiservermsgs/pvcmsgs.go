package apiservermsgs

/*
Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

// ShowPVCRequest ...
// swagger:model
type ShowPVCRequest struct {
	ClusterName   string
	Selector      string
	ClientVersion string
	Namespace     string
	AllFlag       bool
}

// ShowPVCResponse ...
// swagger:model
type ShowPVCResponse struct {
	Results []ShowPVCResponseResult
	Status
}

// ShowPVCResponseResult contains a semi structured result of information
// about a PVC in a cluster
type ShowPVCResponseResult struct {
	ClusterName string
	PVCName     string
}
