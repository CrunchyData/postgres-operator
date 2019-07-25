package apiservermsgs

/*
Copyright 2019 Crunchy Data Solutions, Inc.
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

type NamespaceResult struct {
	Namespace          string
	InstallationAccess bool
	UserAccess         bool
}

// ShowNamespaceRequest ...
type ShowNamespaceRequest struct {
	Args          []string
	AllFlag       bool
	ClientVersion string
}

// ShowNamespaceResponse ...
type ShowNamespaceResponse struct {
	Username string
	Results  []NamespaceResult
	Status
}

// UpdateNamespaceRequest ...
type UpdateNamespaceRequest struct {
	Args          []string
	ClientVersion string
}

// UpdateNamespaceResponse ...
type UpdateNamespaceResponse struct {
	Results []string
	Status
}

// CreateNamespaceRequest ...
type CreateNamespaceRequest struct {
	Args          []string
	Namespace     string
	ClientVersion string
}

// CreateNamespaceResponse ...
type CreateNamespaceResponse struct {
	Results []string
	Status
}

// DeleteNamespaceRequest ...
type DeleteNamespaceRequest struct {
	Args          []string
	Selector      string
	Namespace     string
	AllFlag       bool
	ClientVersion string
}

// DeletePgouserResponse ...
type DeleteNamespaceResponse struct {
	Results []string
	Status
}
