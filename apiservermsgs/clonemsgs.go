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

import (
	"errors"
)

// CloneRequest ...
// swagger:model
type CloneRequest struct {
	Namespace             string
	SourceClusterName     string
	TargetClusterName     string
	ClientVersion         string
	BackrestStorageSource string
}

// CloneReseponse
// swagger:model
type CloneResponse struct {
	Status
	TargetClusterName string
	WorkflowID        string
}

// Validate validates that the parameters for the CloneRequest are valid for
// use int he API
func (r CloneRequest) Validate() error {
	// ensure the cluster name for the source of the clone is set
	if r.SourceClusterName == "" {
		return errors.New("the source cluster name must be set")
	}

	// ensure the cluster name for the target of the clone (the new cluster) is
	// set
	if r.TargetClusterName == "" {
		return errors.New("the target cluster name must be set")
	}

	return nil
}
