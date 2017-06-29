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

type PgClusterSpec struct {
	Name                  string `json:"name"`
	ClusterName           string `json:"clustername"`
	Policies              string `json:"policies"`
	CCP_IMAGE_TAG         string `json:"ccpimagetag"`
	POSTGRES_FULL_VERSION string `json:"postgresfullversion"`
	Port                  string `json:"port"`
	PVC_NAME              string `json:"pvcname"`
	PVC_SIZE              string `json:"pvcsize"`
	PVC_ACCESS_MODE       string `json:"pvcaccessmode"`
	PG_MASTER_HOST        string `json:"pgmasterhost"`
	PG_MASTER_USER        string `json:"pgmasteruser"`
	PG_MASTER_PASSWORD    string `json:"pgmasterpassword"`
	PG_USER               string `json:"pguser"`
	PG_PASSWORD           string `json:"pgpassword"`
	PG_DATABASE           string `json:"pgdatabase"`
	PG_ROOT_PASSWORD      string `json:"pgrootpassword"`
	REPLICAS              string `json:"replicas"`
	FS_GROUP              string `json:"fsgroup"`
	SUPPLEMENTAL_GROUPS   string `json:"supplementalgroups"`
	STRATEGY              string `json:"strategy"`
	SECRET_FROM           string `json:"secretfrom"`
	BACKUP_PVC_NAME       string `json:"backuppvcname"`
	BACKUP_PATH           string `json:"backuppath"`
	PGUSER_SECRET_NAME    string `json:"pgusersecretname"`
	PGROOT_SECRET_NAME    string `json:"pgrootsecretname"`
	PGMASTER_SECRET_NAME  string `json:"pgmastersecretname"`
	STATUS                string `json:"status"`
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
