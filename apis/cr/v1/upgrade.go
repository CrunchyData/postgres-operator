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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const UPGRADE_COMPLETED_STATUS = "completed"
const UPGRADE_SUBMITTED_STATUS = "submitted"

const PgupgradeResourcePlural = "pgupgrades"

type PgupgradeSpec struct {
	Name              string        `json:"name"`
	RESOURCE_TYPE     string        `json:"resourcetype"`
	UPGRADE_TYPE      string        `json:"upgradetype"`
	UPGRADE_STATUS    string        `json:"upgradestatus"`
	StorageSpec       PgStorageSpec `json:"storagespec"`
	CCP_IMAGE_TAG     string        `json:"ccpimagetag"`
	OLD_DATABASE_NAME string        `json:"olddatabasename"`
	NEW_DATABASE_NAME string        `json:"newdatabasename"`
	OLD_VERSION       string        `json:"oldversion"`
	NEW_VERSION       string        `json:"newversion"`
	OLD_PVC_NAME      string        `json:"oldpvcname"`
	NEW_PVC_NAME      string        `json:"newpvcname"`
	BACKUP_PVC_NAME   string        `json:"backuppvcname"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type Pgupgrade struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata"`

	Spec   PgupgradeSpec   `json:"spec"`
	Status PgupgradeStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type PgupgradeList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []Pgupgrade `json:"items"`
}

type PgupgradeStatus struct {
	State   PgupgradeState `json:"state,omitempty"`
	Message string         `json:"message,omitempty"`
}

type PgupgradeState string

const (
	PgupgradeStateCreated   PgupgradeState = "Created"
	PgupgradeStateProcessed PgupgradeState = "Processed"
)
