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

// CloneRequest ...
// swagger:model
type CloneRequest struct {
	// BackrestPVCSize, if set, is the size of the PVC to use for the pgBackRest
	// repository if local storage is being used
	BackrestPVCSize string
	// BackrestStorageSource contains the accepted values for where pgBackRest
	// repository storage exists ("local", "s3" or both)
	BackrestStorageSource string
	ClientVersion         string
	// EnableMetrics enables metrics support in the target cluster
	EnableMetrics bool
	Namespace     string
	// PVCSize, if set, is the size of the PVC to use for the primary and any
	// replicas
	PVCSize string
	// SourceClusterName is the name of the source PostgreSQL cluster being used
	// for the clone
	SourceClusterName string
	// TargetClusterName is the name of the target PostgreSQL cluster that the
	// PostgreSQL cluster will be cloned to
	TargetClusterName string
}

// CloneReseponse
// swagger:model
type CloneResponse struct {
	Status
	TargetClusterName string
	WorkflowID        string
}
