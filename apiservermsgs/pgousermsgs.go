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

import ()

// ShowPgoUserRequest ...
// swagger:model
type ShowPgoUserRequest struct {
	Namespace     string
	AllFlag       bool
	ClientVersion string
	PgoUserName   []string
}

// PgoUserInfo ...
// swagger:model
type PgoUserInfo struct {
	Username  string
	Role      []string
	Namespace []string
}

// ShowPgoUserResponse ...
// swagger:model
type ShowPgoUserResponse struct {
	UserInfo []PgoUserInfo
	Status
}

// CreatePgoUserRequest ...
// swagger:model
type CreatePgoUserRequest struct {
	PgoUserName       string
	PgoUserPassword   string
	PgoUserRoles      string
	AllNamespaces     bool
	PgoUserNamespaces string
	Namespace         string
	ClientVersion     string
}

// CreatePgoUserResponse ...
// swagger:model
type CreatePgoUserResponse struct {
	Status
}

// UpdatePgoUserRequest ...
// swagger:model
type UpdatePgoUserRequest struct {
	Name              string
	PgoUserRoles      string
	PgoUserNamespaces string
	AllNamespaces     bool
	PgoUserPassword   string
	PgoUserName       string
	Namespace         string
	ClientVersion     string
}

// ApplyPgoUserResponse ...
// swagger:model
type UpdatePgoUserResponse struct {
	Status
}

// DeletePgoUserRequest ...
// swagger:model
type DeletePgoUserRequest struct {
	PgoUserName   []string
	Namespace     string
	AllFlag       bool
	ClientVersion string
}

// DeletePgoUserResponse ...
// swagger:model
type DeletePgoUserResponse struct {
	Results []string
	Status
}
