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

// UpgradeCompletedStatus ....
const UpgradeCompletedStatus = "upgrade completed"

// UpgradeSubmittedStatus ....
const UpgradeSubmittedStatus = "upgrade submitted"

// PgupgradeResourcePlural ...
const PgupgradeResourcePlural = "pgupgrades"

// PgupgradeSpec ...
type PgupgradeSpec struct {
	Name            string        `json:"name"`
	ResourceType    string        `json:"resourcetype"`
	UpgradeType     string        `json:"upgradetype"`
	UpgradeStatus   string        `json:"upgradestatus"`
	StorageSpec     PgStorageSpec `json:"storagespec"`
	CCPImageTag     string        `json:"ccpimagetag"`
	OldDatabaseName string        `json:"olddatabasename"`
	NewDatabaseName string        `json:"newdatabasename"`
	OldVersion      string        `json:"oldversion"`
	NewVersion      string        `json:"newversion"`
	OldPVCName      string        `json:"oldpvcname"`
	NewPVCName      string        `json:"newpvcname"`
	BackupPVCName   string        `json:"backuppvcname"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// Pgupgrade ...
type Pgupgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgupgradeSpec   `json:"spec"`
	Status PgupgradeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// PgupgradeList ...
type PgupgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgupgrade `json:"items"`
}

// PgupgradeStatus  ...
type PgupgradeStatus struct {
	State   PgupgradeState `json:"state,omitempty"`
	Message string         `json:"message,omitempty"`
}

// PgupgradeState ...
type PgupgradeState string

// PgupgradeStateCreated  ...
const PgupgradeStateCreated PgupgradeState = "Created"

// PgupgradeStateProcessed ...
const PgupgradeStateProcessed PgupgradeState = "Processed"
