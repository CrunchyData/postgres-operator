package v1

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PgpolicyResourcePlural ...
const PgpolicyResourcePlural = "pgpolicies"

// PgpolicySpec ...
type PgpolicySpec struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	SQL       string `json:"sql"`
	Status    string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Pgpolicy ...
type Pgpolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgpolicySpec   `json:"spec"`
	Status PgpolicyStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PgpolicyList ...
type PgpolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgpolicy `json:"items"`
}

// PgpolicyStatus ...
type PgpolicyStatus struct {
	State   PgpolicyState `json:"state,omitempty"`
	Message string        `json:"message,omitempty"`
}

// PgpolicyState ...
type PgpolicyState string

const (
	// PgpolicyStateCreated ...
	PgpolicyStateCreated PgpolicyState = "pgpolicy Created"
	// PgpolicyStateProcessed ...
	PgpolicyStateProcessed PgpolicyState = "pgpolicy Processed"
)
