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

// NamespaceResult ...
// swagger:model
type NamespaceResult struct {
	Namespace          string
	InstallationAccess bool
	UserAccess         bool
}

// ShowNamespaceRequest ...
// swagger:model
type ShowNamespaceRequest struct {
	Args          []string
	AllFlag       bool
	ClientVersion string
}

// ShowNamespaceResponse ...
// swagger:model
type ShowNamespaceResponse struct {
	Username string
	Results  []NamespaceResult
	Status
}

// UpdateNamespaceRequest ...
// swagger:model
type UpdateNamespaceRequest struct {
	Args          []string
	ClientVersion string
}

// UpdateNamespaceResponse ...
// swagger:model
type UpdateNamespaceResponse struct {
	Results []string
	Status
}

// CreateNamespaceRequest ...
// swagger:model
type CreateNamespaceRequest struct {
	Args          []string
	Namespace     string
	ClientVersion string
}

// CreateNamespaceResponse ...
// swagger:model
type CreateNamespaceResponse struct {
	Results []string
	Status
}

// DeleteNamespaceRequest ...
// swagger:model
type DeleteNamespaceRequest struct {
	Args          []string
	Selector      string
	Namespace     string
	AllFlag       bool
	ClientVersion string
}

// DeleteNamespaceResponse ...
// swagger:model
type DeleteNamespaceResponse struct {
	Results []string
	Status
}
