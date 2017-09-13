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
	"time"
)

const CLUSTER_RESOURCE = "pgclusters"

type PgClusterSpec struct {
	Name                  string        `json:"name"`
	ClusterName           string        `json:"clustername"`
	Policies              string        `json:"policies"`
	CCP_IMAGE_TAG         string        `json:"ccpimagetag"`
	POSTGRES_FULL_VERSION string        `json:"postgresfullversion"`
	Port                  string        `json:"port"`
	NodeName              string        `json:"nodename"`
	MasterStorage         PgStorageSpec `json:masterstorage`
	ReplicaStorage        PgStorageSpec `json:replicastorage`
	PG_MASTER_HOST        string        `json:"pgmasterhost"`
	PG_MASTER_USER        string        `json:"pgmasteruser"`
	PG_MASTER_PASSWORD    string        `json:"pgmasterpassword"`
	PG_USER               string        `json:"pguser"`
	PG_PASSWORD           string        `json:"pgpassword"`
	PG_DATABASE           string        `json:"pgdatabase"`
	PG_ROOT_PASSWORD      string        `json:"pgrootpassword"`
	REPLICAS              string        `json:"replicas"`
	STRATEGY              string        `json:"strategy"`
	SECRET_FROM           string        `json:"secretfrom"`
	BACKUP_PVC_NAME       string        `json:"backuppvcname"`
	BACKUP_PATH           string        `json:"backuppath"`
	PGUSER_SECRET_NAME    string        `json:"pgusersecretname"`
	PGROOT_SECRET_NAME    string        `json:"pgrootsecretname"`
	PGMASTER_SECRET_NAME  string        `json:"pgmastersecretname"`
	STATUS                string        `json:"status"`
	PSW_LAST_UPDATE       time.Time     `json:"pswlastupdate"`
}

type PgCluster struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`

	Spec PgClusterSpec `json:"spec"`
}

type PgClusterList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []PgCluster `json:"items"`
}

func (e *PgCluster) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

func (e *PgCluster) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

func (el *PgClusterList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

func (el *PgClusterList) GetListMeta() metav1.List {
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
