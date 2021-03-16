package apiservermsgs

/*
Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

// CreatePgAdminRequest ...
// swagger:model
type CreatePgAdminRequest struct {
	Args          []string
	ClientVersion string
	Namespace     string
	Selector      string
	StorageConfig string
	PVCSize       string
}

// CreatePgAdminResponse ...
// swagger:model
type CreatePgAdminResponse struct {
	Results []string
	Status
}

// DeletePgAdminRequest ...
// swagger:model
type DeletePgAdminRequest struct {
	Args          []string
	Selector      string
	Namespace     string
	ClientVersion string
	Uninstall     bool
}

// DeletePgAdminResponse ...
// swagger:model
type DeletePgAdminResponse struct {
	Results []string
	Status
}

// ShowPgAdminDetail is the specific information about a pgAdmin deployment
// for a cluster
//
// swagger:model
type ShowPgAdminDetail struct {
	// ClusterName is the name of the PostgreSQL cluster associated with this
	// pgAdmin deployment
	ClusterName string
	// HasPgAdmin is set to true if there is a pgAdmin deployment with this
	// cluster, otherwise its false
	HasPgAdmin bool
	// ServiceClusterIP contains the ClusterIP address of the Service
	ServiceClusterIP string
	// ServiceExternalIP contains the external IP address of the Service, if it
	// is assigned
	ServiceExternalIP string
	// ServiceName contains the name of the Kubernetes Service
	ServiceName string
	// Users contains the list of users configured for pgAdmin login
	Users []string
}

// ShowPgAdminRequest contains the attributes for requesting information about
// a pgAdmin deployment
//
// swagger:model
type ShowPgAdminRequest struct {
	// ClientVersion is the required parameter that includes the version of the
	// Operator that is requesting
	ClientVersion string

	// ClusterNames contains one or more names of cluster to be queried to show
	// information about their pgAdmin deployment
	ClusterNames []string

	// Namespace is the namespace to perform the query in
	Namespace string

	// Selector is optional and contains a selector to gather information about
	// a PostgreSQL cluster's pgAdmin
	Selector string
}

// ShowPgAdminResponse contains the attributes that are part of the response
// from the pgAdmin request, i.e. pgAdmin information
//
// swagger:model
type ShowPgAdminResponse struct {
	Results []ShowPgAdminDetail
	Status
}
