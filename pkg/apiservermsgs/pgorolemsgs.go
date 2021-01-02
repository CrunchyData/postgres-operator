package apiservermsgs

/*
Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

// ShowPgoroleRequest ...
// swagger:model
type ShowPgoroleRequest struct {
	Namespace     string
	AllFlag       bool
	ClientVersion string
	PgoroleName   []string
}

// PgroleInfo ...
// swagger:model
type PgoroleInfo struct {
	Name        string
	Permissions string
}

// ShowPgoroleResponse ...
// swagger:model
type ShowPgoroleResponse struct {
	RoleInfo []PgoroleInfo
	Status
}

// CreatePgoroleRequest ...
// swagger:model
type CreatePgoroleRequest struct {
	PgoroleName        string
	PgorolePermissions string
	Namespace          string
	ClientVersion      string
}

// CreatePgoroleResponse ...
// swagger:model
type CreatePgoroleResponse struct {
	Status
}

// UpdatePgoroleRequest ...
// swagger:model
type UpdatePgoroleRequest struct {
	Name               string
	PgorolePermissions string
	PgoroleName        string
	ChangePermissions  bool
	Namespace          string
	ClientVersion      string
}

// ApplyPgoroleResponse ...
// swagger:model
type UpdatePgoroleResponse struct {
	Status
}

// DeletePgoroleRequest ...
// swagger:model
type DeletePgoroleRequest struct {
	PgoroleName   []string
	Namespace     string
	AllFlag       bool
	ClientVersion string
}

// DeletePgoroleResponse ...
// swagger:model
type DeletePgoroleResponse struct {
	Results []string
	Status
}
