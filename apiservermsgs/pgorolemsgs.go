package apiservermsgs

/*
Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

// ShowPgoRoleRequest ...
// swagger:model
type ShowPgoRoleRequest struct {
	Namespace     string
	AllFlag       bool
	ClientVersion string
	PgoroleName   []string
}

// PgoRoleInfo ...
// swagger:model
type PgoRoleInfo struct {
	Name        string
	Permissions string
}

// ShowPgoRoleResponse ...
// swagger:model
type ShowPgoRoleResponse struct {
	RoleInfo []PgoRoleInfo
	Status
}

// CreatePgoRoleRequest ...
// swagger:model
type CreatePgoRoleRequest struct {
	PgoroleName        string
	PgorolePermissions string
	Namespace          string
	ClientVersion      string
}

// CreatePgoRoleResponse ...
// swagger:model
type CreatePgoRoleResponse struct {
	Status
}

// UpdatePgoRoleRequest ...
// swagger:model
type UpdatePgoRoleRequest struct {
	Name               string
	PgorolePermissions string
	PgoroleName        string
	ChangePermissions  bool
	Namespace          string
	ClientVersion      string
}

// UpdatePgoRoleResponse ...
// swagger:model
type UpdatePgoRoleResponse struct {
	Status
}

// DeletePgoRoleRequest ...
// swagger:model
type DeletePgoRoleRequest struct {
	PgoroleName   []string
	Namespace     string
	AllFlag       bool
	ClientVersion string
}

// DeletePgoRoleResponse ...
// swagger:model
type DeletePgoRoleResponse struct {
	Results []string
	Status
}
