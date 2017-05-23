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
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
)

const UPGRADE_COMPLETED_STATUS = "completed"
const UPGRADE_SUBMITTED_STATUS = "submitted"

type PgUpgradeSpec struct {
	Name              string `json:"name"`
	RESOURCE_TYPE     string `json:"resourcetype"`
	UPGRADE_TYPE      string `json:"upgradetype"`
	UPGRADE_STATUS    string `json:"upgradestatus"`
	PVC_ACCESS_MODE   string `json:"pvcaccessmode"`
	PVC_SIZE          string `json:"pvcsize"`
	CCP_IMAGE_TAG     string `json:"ccpimagetag"`
	OLD_DATABASE_NAME string `json:"olddatabasename"`
	NEW_DATABASE_NAME string `json:"newdatabasename"`
	OLD_VERSION       string `json:"oldversion"`
	NEW_VERSION       string `json:"newversion"`
	OLD_PVC_NAME      string `json:"oldpvcname"`
	NEW_PVC_NAME      string `json:"newpvcname"`
	BACKUP_PVC_NAME   string `json:"backuppvcname"`
}

type PgUpgrade struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec PgUpgradeSpec `json:"spec"`
}

type PgUpgradeList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []PgUpgrade `json:"items"`
}

func (f *PgUpgrade) GetObjectKind() unversioned.ObjectKind {
	return &f.TypeMeta
}

func (f *PgUpgrade) GetObjectMeta() meta.Object {
	return &f.Metadata
}

func (fl *PgUpgradeList) GetObjectKind() unversioned.ObjectKind {
	return &fl.TypeMeta
}

func (fl *PgUpgradeList) GetListMeta() unversioned.List {
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
