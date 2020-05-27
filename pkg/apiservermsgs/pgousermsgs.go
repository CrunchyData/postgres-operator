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

// ShowPgouserRequest ...
// swagger:model
type ShowPgouserRequest struct {
	Namespace     string
	AllFlag       bool
	ClientVersion string
	PgouserName   []string
}

// PgouserInfo ...
// swagger:model
type PgouserInfo struct {
	Username  string
	Role      []string
	Namespace []string
}

// ShowPgouserResponse ...
// swagger:model
type ShowPgouserResponse struct {
	UserInfo []PgouserInfo
	Status
}

// CreatePgouserRequest ...
// swagger:model
type CreatePgouserRequest struct {
	PgouserName       string
	PgouserPassword   string
	PgouserRoles      string
	AllNamespaces     bool
	PgouserNamespaces string
	Namespace         string
	ClientVersion     string
}

// CreatePgouserResponse ...
// swagger:model
type CreatePgouserResponse struct {
	Status
}

// UpdatePgouserRequest ...
// swagger:model
type UpdatePgouserRequest struct {
	Name              string
	PgouserRoles      string
	PgouserNamespaces string
	AllNamespaces     bool
	PgouserPassword   string
	PgouserName       string
	Namespace         string
	ClientVersion     string
}

// ApplyPgouserResponse ...
// swagger:model
type UpdatePgouserResponse struct {
	Status
}

// DeletePgouserRequest ...
// swagger:model
type DeletePgouserRequest struct {
	PgouserName   []string
	Namespace     string
	AllFlag       bool
	ClientVersion string
}

// DeletePgouserResponse ...
// swagger:model
type DeletePgouserResponse struct {
	Results []string
	Status
}
