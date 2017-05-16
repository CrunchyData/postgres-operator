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
	Name                  string `json:"name"`
	PVC_NAME              string `json:"pvcname"`
	PVC_ACCESS_MODE       string `json:"pvcaccessmode"`
	PVC_SIZE              string `json:"pvcsize"`
	Port                  string `json:"port"`
	CCP_IMAGE_TAG         string `json:"ccpimagetag"`
	POSTGRES_FULL_VERSION string `json:"postgresfullversion"`
	PG_MASTER_USER        string `json:"pgmasteruser"`
	PG_MASTER_PASSWORD    string `json:"pgmasterpassword"`
	PG_USER               string `json:"pguser"`
	PG_PASSWORD           string `json:"pgpassword"`
	PG_DATABASE           string `json:"pgdatabase"`
	PG_ROOT_PASSWORD      string `json:"pgrootpassword"`
	BACKUP_PVC_NAME       string `json:"backuppvcname"`
	BACKUP_PATH           string `json:"backuppath"`
	FS_GROUP              string `json:"fsgroup"`
	SUPPLEMENTAL_GROUPS   string `json:"supplementalgroups"`
	STRATEGY              string `json:"strategy"`
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
