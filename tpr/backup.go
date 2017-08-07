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

const BACKUP_RESOURCE = "pgbackups"

type PgBackupSpec struct {
	Name          string        `json:"name"`
	StorageSpec   PgStorageSpec `json:"storagespec"`
	CCP_IMAGE_TAG string        `json:"ccpimagetag"`
	BACKUP_HOST   string        `json:"backuphost"`
	BACKUP_USER   string        `json:"backupuser"`
	BACKUP_PASS   string        `json:"backuppass"`
	BACKUP_PORT   string        `json:"backupport"`
	BACKUP_STATUS string        `json:"backupstatus"`
}

type PgBackup struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`

	Spec PgBackupSpec `json:"spec"`
}

type PgBackupList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []PgBackup `json:"items"`
}

func (e *PgBackup) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

func (e *PgBackup) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

func (el *PgBackupList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

func (el *PgBackupList) GetListMeta() metav1.List {
	return &el.Metadata
}

type PgBackupListCopy PgBackupList
type PgBackupCopy PgBackup

func (e *PgBackup) UnmarshalJSON(data []byte) error {
	tmp := PgBackupCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgBackup(tmp)
	*e = tmp2
	return nil
}

func (el *PgBackupList) UnmarshalJSON(data []byte) error {
	tmp := PgBackupListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgBackupList(tmp)
	*el = tmp2
	return nil
}
