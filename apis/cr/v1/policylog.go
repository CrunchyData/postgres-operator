package v1

/*
 Copyright 2017 Crunchy Data Solutions, Inc.
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

// PgpolicylogResourcePlural ...
const PgpolicylogResourcePlural = "pgpolicylogs"

// PgpolicylogSpec ...
type PgpolicylogSpec struct {
	PolicyName  string `json:"policyname"`
	Status      string `json:"status"`
	ApplyDate   string `json:"applydate"`
	ClusterName string `json:"clustername"`
	Username    string `json:"username"`
}

// Pgpolicylog ...
type Pgpolicylog struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgpolicylogSpec   `json:"spec"`
	Status PgpolicylogStatus `json:"status,omitempty"`
}

// PgpolicylogList ...
type PgpolicylogList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgpolicylog `json:"items"`
}

// PgpolicylogStatus ...
type PgpolicylogStatus struct {
	State   PgpolicylogState `json:"state,omitempty"`
	Message string           `json:"message,omitempty"`
}

// PgpolicylogState ...
type PgpolicylogState string

const (
	// PgpolicylogStateCreated ...
	PgpolicylogStateCreated PgpolicylogState = "Created"
	// PgpolicylogStateProcessed ...
	PgpolicylogStateProcessed PgpolicylogState = "Processed"
)
