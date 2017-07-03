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

const POLICY_LOG_RESOURCE = "pgpolicylogs"

type PgPolicylogSpec struct {
	PolicyName  string `json:"policyname"`
	Status      string `json:"status"`
	ApplyDate   string `json:"applydate"`
	ClusterName string `json:"clustername"`
	Username    string `json:"username"`
}

type PgPolicylog struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             api.ObjectMeta `json:"metadata"`

	Spec PgPolicylogSpec `json:"spec"`
}

type PgPolicylogList struct {
	unversioned.TypeMeta `json:",inline"`
	Metadata             unversioned.ListMeta `json:"metadata"`

	Items []PgPolicylog `json:"items"`
}

func (e *PgPolicylog) GetObjectKind() unversioned.ObjectKind {
	return &e.TypeMeta
}

func (e *PgPolicylog) GetObjectMeta() meta.Object {
	return &e.Metadata
}

func (el *PgPolicylogList) GetObjectKind() unversioned.ObjectKind {
	return &el.TypeMeta
}

func (el *PgPolicylogList) GetListMeta() unversioned.List {
	return &el.Metadata
}

type PgPolicylogListCopy PgPolicylogList
type PgPolicylogCopy PgPolicylog

func (e *PgPolicylog) UnmarshalJSON(data []byte) error {
	tmp := PgPolicylogCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgPolicylog(tmp)
	*e = tmp2
	return nil
}

func (el *PgPolicylogList) UnmarshalJSON(data []byte) error {
	tmp := PgPolicylogListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgPolicylogList(tmp)
	*el = tmp2
	return nil
}
