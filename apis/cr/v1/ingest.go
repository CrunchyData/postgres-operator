package v1

/*
 Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

// PgingestResourcePlural ..
const PgingestResourcePlural = "pgingests"

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Pgingest ..
type Pgingest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`
	Spec              PgingestSpec   `json:"spec"`
	Status            PgingestStatus `json:"status,omitempty"`
}

// PgingestSpec ...
type PgingestSpec struct {
	Name            string `json:"name"`
	WatchDir        string `json:"watchdir"`
	DBHost          string `json:"dbhost"`
	DBPort          string `json:"dbport"`
	DBName          string `json:"dbname"`
	DBSecret        string `json:"dbsecret"`
	DBTable         string `json:"dbtable"`
	DBColumn        string `json:"dbcolumn"`
	MaxJobs         int    `json:"maxjobs"`
	PVCName         string `json:"pvcname"`
	SecurityContext string `json:"securitycontext"`
	Status          string `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PgingestList ...
type PgingestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgingest `json:"items"`
}

// PgingestStatus ...
type PgingestStatus struct {
	State   PgingestState `json:"state,omitempty"`
	Message string        `json:"message,omitempty"`
}

// PgingestState ...
type PgingestState string

const (
	// PgingestStateCreated ...
	PgingestStateCreated PgingestState = "Created"
	// PgingestStateProcessed ...
	PgingestStateProcessed PgingestState = "Processed"
)
