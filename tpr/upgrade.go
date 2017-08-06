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

// Package tpr defines the ThirdPartyResources used within
// the crunchy operator, namely the PgDatabase and PgCluster
// types.
package tpr

import (
	"encoding/json"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const UPGRADE_RESOURCE = "pgupgrades"
const UPGRADE_COMPLETED_STATUS = "completed"
const UPGRADE_SUBMITTED_STATUS = "submitted"

type PgUpgradeSpec struct {
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

type PgUpgrade struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`

	Spec PgUpgradeSpec `json:"spec"`
}

type PgUpgradeList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []PgUpgrade `json:"items"`
}

func (f *PgUpgrade) GetObjectKind() schema.ObjectKind {
	return &f.TypeMeta
}

func (f *PgUpgrade) GetObjectMeta() metav1.Object {
	return &f.Metadata
}

func (fl *PgUpgradeList) GetObjectKind() schema.ObjectKind {
	return &fl.TypeMeta
}

func (fl *PgUpgradeList) GetListMeta() metav1.List {
	return &fl.Metadata
}

type PgUpgradeListCopy PgUpgradeList
type PgUpgradeCopy PgUpgrade

func (f *PgUpgrade) UnmarshalJSON(data []byte) error {
	tmp := PgUpgradeCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgUpgrade(tmp)
	*f = tmp2
	return nil
}

func (fl *PgUpgradeList) UnmarshalJSON(data []byte) error {
	tmp := PgUpgradeListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgUpgradeList(tmp)
	*fl = tmp2
	return nil
}
