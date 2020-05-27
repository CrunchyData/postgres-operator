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

// CreatePgbouncerRequest ...
// swagger:model
type CreatePgbouncerRequest struct {
	Args          []string
	ClientVersion string
	// CPULimit, if specified, is the max CPU that should be used on a pgBouncer
	// Pod. Defaults to not being set.
	CPULimit string
	// CPURequest, if specified, is the value of how much CPU should be
	// requested for deploying pgBouncer instances. Defaults to not being
	// requested
	CPURequest string
	// MemoryLimit, if specified, is the max CPU that should be used on a
	// pgBouncer Pod. Defaults to not being set.
	MemoryLimit string
	// MemoryRequest, if specified, is the value of how much RAM should
	// be requested for deploying pgBouncer instances. Defaults to the server
	// specified default
	MemoryRequest string
	Namespace     string
	// Replicas represents the total number of pgBouncer pods to deploy with a
	// PostgreSQL cluster. Must be at least 1. If 0 is passed in, it will
	// automatically be set to 1
	Replicas int32
	Selector string
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

// UpdatePgBouncerDetail is the specific information about the pgBouncer update
// request for each deployment
//
// swagger:model
type UpdatePgBouncerDetail struct {
	// ClusterName is the name of the PostgreSQL cluster associated with this
	// pgBouncer deployment
	ClusterName string
	// Error is set to true if there is an error. HasPgbouncer == false is not
	// an error
	Error bool
	// ErrorMessage contains an error message if there is an error
	ErrorMessage string
	// HasPgBouncer is set to true if there is a pgBouncer deployment with this
	// cluster, otherwise its false
	HasPgBouncer bool
}

// UpdatePgBouncerRequest contains the attributes for updating a pgBouncer
// deployment
//
// swagger:model
type UpdatePgBouncerRequest struct {
	// ClientVersion is the required parameter that includes the version of the
	// Operator that is requesting
	ClientVersion string

	// ClusterNames contains one or more names of pgBouncer deployments to be
	// updated
	ClusterNames []string

	// CPULimit, if specified, is the max CPU that should be used on a pgBouncer
	// Pod. Defaults to not being set.
	CPULimit string

	// CPURequest, if specified, is the value of how much CPU should be
	// requested for deploying pgBouncer instances. Defaults to not being
	// requested
	CPURequest string

	// MemoryLimit, if specified, is the max CPU that should be used on a
	// pgBouncer Pod. Defaults to not being set.
	MemoryLimit string

	// MemoryRequest, if specified, is the value of how much RAM should
	// be requested for deploying pgBouncer instances. Defaults to the server
	// specified default
	MemoryRequest string

	// Namespace is the namespace to perform the query in
	Namespace string

	// Replicas represents the total number of pgBouncer pods to deploy with a
	// PostgreSQL cluster. Must be at least 1. If 0 is passed in, it is ignored
	Replicas int32

	// RotatePassword is used to rotate the password for the "pgbouncer" service
	// account
	RotatePassword bool

	// Selector is optional and contains a selector for pgBouncer deployments that
	// are to be updated
	Selector string
}

// UpdatePgBouncerResponse contains the resulting output of the update request
//
// swagger:model
type UpdatePgBouncerResponse struct {
	Results []UpdatePgBouncerDetail
	Status
}
