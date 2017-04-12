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

type PgDatabaseSpec struct {
	Name                string `json:"name"`
	PVC_NAME            string `json:"pvcname"`
	PVC_ACCESS_MODE     string `json:"pvcaccessmode"`
	PVC_SIZE            string `json:"pvcsize"`
	Port                string `json:"port"`
	CCP_IMAGE_TAG       string `json:"ccpimagetag"`
	PG_MASTER_USER      string `json:"pgmasteruser"`
	PG_MASTER_PASSWORD  string `json:"pgmasterpassword"`
	PG_USER             string `json:"pguser"`
	PG_PASSWORD         string `json:"pgpassword"`
	PG_DATABASE         string `json:"pgdatabase"`
	PG_ROOT_PASSWORD    string `json:"pgrootpassword"`
	BACKUP_PVC_NAME     string `json:"backuppvcname"`
	BACKUP_PATH         string `json:"backuppath"`
	FS_GROUP            string `json:"fsgroup"`
	SUPPLEMENTAL_GROUPS string `json:"supplementalgroups"`
	STRATEGY            string `json:"strategy"`
}

type PgDatabase struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec PgDatabaseSpec `json:"spec"`
}

type PgDatabaseList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []PgDatabase `json:"items"`
}

func (e *PgDatabase) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *PgDatabase) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *PgDatabaseList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *PgDatabaseList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type PgDatabaseListCopy PgDatabaseList
type PgDatabaseCopy PgDatabase

func (e *PgDatabase) UnmarshalJSON(data []byte) error {
	tmp := PgDatabaseCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgDatabase(tmp)
	*e = tmp2
	return nil
}
func (el *PgDatabaseList) UnmarshalJSON(data []byte) error {
	tmp := PgDatabaseListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgDatabaseList(tmp)
	*el = tmp2
	return nil
}

type PgClusterSpec struct {
	Name                string `json:"name"`
	ClusterName         string `json:"clustername"`
	CCP_IMAGE_TAG       string `json:"ccpimagetag"`
	Port                string `json:"port"`
	PVC_NAME            string `json:"pvcname"`
	PVC_SIZE            string `json:"pvcsize"`
	PVC_ACCESS_MODE     string `json:"pvcaccessmode"`
	PG_MASTER_HOST      string `json:"pgmasterhost"`
	PG_MASTER_USER      string `json:"pgmasteruser"`
	PG_MASTER_PASSWORD  string `json:"pgmasterpassword"`
	PG_USER             string `json:"pguser"`
	PG_PASSWORD         string `json:"pgpassword"`
	PG_DATABASE         string `json:"pgdatabase"`
	PG_ROOT_PASSWORD    string `json:"pgrootpassword"`
	REPLICAS            string `json:"replicas"`
	FS_GROUP            string `json:"fsgroup"`
	SUPPLEMENTAL_GROUPS string `json:"supplementalgroups"`
	STRATEGY            string `json:"strategy"`
}

type PgCluster struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec PgClusterSpec `json:"spec"`
}

type PgClusterList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []PgCluster `json:"items"`
}

func (e *PgCluster) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *PgCluster) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *PgClusterList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *PgClusterList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type PgClusterListCopy PgClusterList
type PgClusterCopy PgCluster

func (e *PgCluster) UnmarshalJSON(data []byte) error {
	tmp := PgClusterCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgCluster(tmp)
	*e = tmp2
	return nil
}
func (el *PgClusterList) UnmarshalJSON(data []byte) error {
	tmp := PgClusterListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgClusterList(tmp)
	*el = tmp2
	return nil
}

type PgBackupSpec struct {
	Name            string `json:"name"`
	PVC_NAME        string `json:"pvcname"`
	PVC_ACCESS_MODE string `json:"pvcaccessmode"`
	PVC_SIZE        string `json:"pvcsize"`
	CCP_IMAGE_TAG   string `json:"ccpimagetag"`
	BACKUP_HOST     string `json:"backuphost"`
	BACKUP_USER     string `json:"backupuser"`
	BACKUP_PASS     string `json:"backuppass"`
	BACKUP_PORT     string `json:"backupport"`
}

type PgBackup struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec PgBackupSpec `json:"spec"`
}

type PgBackupList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []PgBackup `json:"items"`
}

func (e *PgBackup) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *PgBackup) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *PgBackupList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *PgBackupList) GetListMeta() unversioned.List {
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

type PgUpgradeSpec struct {
	Name            string `json:"name"`
	PVC_NAME        string `json:"pvcname"`
	PVC_ACCESS_MODE string `json:"pvcaccessmode"`
	PVC_SIZE        string `json:"pvcsize"`
	CCP_IMAGE_TAG   string `json:"ccpimagetag"`
	BACKUP_HOST     string `json:"backuphost"`
	BACKUP_USER     string `json:"backupuser"`
	BACKUP_PASS     string `json:"backuppass"`
	BACKUP_PORT     string `json:"backupport"`
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

func (e *PgUpgrade) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *PgUpgrade) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *PgUpgradeList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *PgUpgradeList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type PgUpgradeListCopy PgUpgradeList
type PgUpgradeCopy PgUpgrade

func (e *PgUpgrade) UnmarshalJSON(data []byte) error {
	tmp := PgUpgradeCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgUpgrade(tmp)
	*e = tmp2
	return nil
}

func (el *PgUpgradeList) UnmarshalJSON(data []byte) error {
	tmp := PgUpgradeListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgUpgradeList(tmp)
	*el = tmp2
	return nil
}
