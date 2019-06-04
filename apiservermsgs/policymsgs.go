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
	crv1 "github.com/crunchydata/postgres-operator/apis/cr/v1"
)

// ShowPolicyRequest ...
type ShowPolicyRequest struct {
	Selector      string
	Namespace     string
	AllFlag       bool
	ClientVersion string
	Policyname    string
}

// CreatePolicyRequest ...
type CreatePolicyRequest struct {
	Name          string
	URL           string
	SQL           string
	Namespace     string
	ClientVersion string
}

// CreatePolicyResponse ...
type CreatePolicyResponse struct {
	Status
}

// ApplyPolicyRequest ...
type ApplyPolicyRequest struct {
	Name          string
	Selector      string
	DryRun        bool
	Namespace     string
	ClientVersion string
}

// ApplyPolicyResponse ...
type ApplyPolicyResponse struct {
	Name []string
	Status
}

// ApplyResults ...
type ApplyResults struct {
	Results []string
}

// ShowPolicyResponse ...
type ShowPolicyResponse struct {
	PolicyList crv1.PgpolicyList
	Status
}

// DeletePolicyRequest ...
type DeletePolicyRequest struct {
	Selector      string
	Namespace     string
	AllFlag       bool
	ClientVersion string
	PolicyName    string
}

// DeletePolicyResponse ...
type DeletePolicyResponse struct {
	Results []string
	Status
}
