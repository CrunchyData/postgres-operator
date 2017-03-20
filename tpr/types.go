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
// the crunchy operator, namely the CrunchyDatabase and CrunchyCluster
// types.
package tpr

import (
	"encoding/json"
	"k8s.io/client-go/pkg/api"
	"k8s.io/client-go/pkg/api/meta"
	"k8s.io/client-go/pkg/api/unversioned"
)

type CrunchyDatabaseSpec struct {
	Name               string `json:"name"`
	PVC_NAME           string `json:"pvcname"`
	Port               string `json:"port"`
	CCP_IMAGE_TAG      string `json:"ccpimagetag"`
	PG_MASTER_USER     string `json:"pgmasteruser"`
	PG_MASTER_PASSWORD string `json:"pgmasterpassword"`
	PG_USER            string `json:"pguser"`
	PG_PASSWORD        string `json:"pgpassword"`
	PG_DATABASE        string `json:"pgdatabase"`
	PG_ROOT_PASSWORD   string `json:"pgrootpassword"`
}

type CrunchyDatabase struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec CrunchyDatabaseSpec `json:"spec"`
}

type CrunchyDatabaseList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []CrunchyDatabase `json:"items"`
}

func (e *CrunchyDatabase) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *CrunchyDatabase) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *CrunchyDatabaseList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *CrunchyDatabaseList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type CrunchyDatabaseListCopy CrunchyDatabaseList
type CrunchyDatabaseCopy CrunchyDatabase

func (e *CrunchyDatabase) UnmarshalJSON(data []byte) error {
	tmp := CrunchyDatabaseCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyDatabase(tmp)
	*e = tmp2
	return nil
}
func (el *CrunchyDatabaseList) UnmarshalJSON(data []byte) error {
	tmp := CrunchyDatabaseListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyDatabaseList(tmp)
	*el = tmp2
	return nil
}

type CrunchyClusterSpec struct {
	Name               string `json:"name"`
	ClusterName        string `json:"clustername"`
	CCP_IMAGE_TAG      string `json:"ccpimagetag"`
	Port               string `json:"port"`
	PVC_NAME           string `json:"pvcname"`
	PG_MASTER_HOST     string `json:"pgmasterhost"`
	PG_MASTER_USER     string `json:"pgmasteruser"`
	PG_MASTER_PASSWORD string `json:"pgmasterpassword"`
	PG_USER            string `json:"pguser"`
	PG_PASSWORD        string `json:"pgpassword"`
	PG_DATABASE        string `json:"pgdatabase"`
	PG_ROOT_PASSWORD   string `json:"pgrootpassword"`
	REPLICAS           string `json:"replicas"`
}

type CrunchyCluster struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec CrunchyClusterSpec `json:"spec"`
}

type CrunchyClusterList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []CrunchyCluster `json:"items"`
}

func (e *CrunchyCluster) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *CrunchyCluster) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *CrunchyClusterList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *CrunchyClusterList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type CrunchyClusterListCopy CrunchyClusterList
type CrunchyClusterCopy CrunchyCluster

func (e *CrunchyCluster) UnmarshalJSON(data []byte) error {
	tmp := CrunchyClusterCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyCluster(tmp)
	*e = tmp2
	return nil
}
func (el *CrunchyClusterList) UnmarshalJSON(data []byte) error {
	tmp := CrunchyClusterListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyClusterList(tmp)
	*el = tmp2
	return nil
}

type CrunchyBackupSpec struct {
	Name          string `json:"name"`
	PVC_NAME      string `json:"pvcname"`
	CCP_IMAGE_TAG string `json:"ccpimagetag"`
	BACKUP_HOST   string `json:"backuphost"`
	BACKUP_USER   string `json:"backupuser"`
	BACKUP_PASS   string `json:"backuppass"`
	BACKUP_PORT   string `json:"backupport"`
}

type CrunchyBackup struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec CrunchyBackupSpec `json:"spec"`
}

type CrunchyBackupList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []CrunchyBackup `json:"items"`
}

func (e *CrunchyBackup) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *CrunchyBackup) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *CrunchyBackupList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *CrunchyBackupList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type CrunchyBackupListCopy CrunchyBackupList
type CrunchyBackupCopy CrunchyBackup

func (e *CrunchyBackup) UnmarshalJSON(data []byte) error {
	tmp := CrunchyBackupCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyBackup(tmp)
	*e = tmp2
	return nil
}

func (el *CrunchyBackupList) UnmarshalJSON(data []byte) error {
	tmp := CrunchyBackupListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := CrunchyBackupList(tmp)
	*el = tmp2
	return nil
}
