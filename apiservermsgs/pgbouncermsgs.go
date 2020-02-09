package apiservermsgs

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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

// CreatePgbouncerRequest ...
// swagger:model
type CreatePgbouncerRequest struct {
	Args          []string
	Selector      string
	Namespace     string
	ClientVersion string
}

// CreatePgbouncerResponse ...
// swagger:model
type CreatePgbouncerResponse struct {
	Results []string
	Status
}

// DeletePgbouncerRequest ...
// swagger:model
type DeletePgbouncerRequest struct {
	Args          []string
	Selector      string
	Namespace     string
	ClientVersion string
	Uninstall     bool
}

// DeletePgbouncerResponse ...
// swagger:model
type DeletePgbouncerResponse struct {
	Results []string
	Status
}

// ShowPgBouncerDetail is the specific information about a pgBouncer deployment
// for a cluster
//
// swagger:model
type ShowPgBouncerDetail struct {
	// ClusterName is the name of the PostgreSQL cluster associated with this
	// pgBouncer deployment
	ClusterName string
	// HasPgBouncer is set to true if there is a pgBouncer deployment with this
	// cluster, otherwise its false
	HasPgBouncer bool
	// Password contains the password for the pgBouncer service account
	Password string
	// ServiceClusterIP contains the ClusterIP address of the Service
	ServiceClusterIP string
	// ServiceExternalIP contains the external IP address of the Service, if it
	// is assigned
	ServiceExternalIP string
	// ServiceName contains the name of the Kubernetes Service
	ServiceName string
	// Username is the username for the pgBouncer service account
	Username string
}

// ShowPgBouncerRequest contains the attributes for requesting information about
// a pgBouncer deployment
//
// swagger:model
type ShowPgBouncerRequest struct {
	// ClientVersion is the required parameter that includes the version of the
	// Operator that is requesting
	ClientVersion string

	// ClusterNames contains one or more names of cluster to be queried to show
	// information about their pgBouncer deployment
	ClusterNames []string

	// Namespace is the namespace to perform the query in
	Namespace string

	// Selector is optional and contains a selector to gather information about
	// a PostgreSQL cluster's pgBouncer
	Selector string
}

// ShowPgBouncerResponse contains the attributes that are part of the response
// from the pgBouncer request, i.e. pgBouncer information
//
// swagger:model
type ShowPgBouncerResponse struct {
	Results []ShowPgBouncerDetail
	Status
}
