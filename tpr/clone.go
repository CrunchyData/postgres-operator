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
// the crunchy operator, namely the PgDatabase and PgClone
// types.
package tpr

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

const CLONE_RESOURCE = "pgclones"

type PgCloneSpec struct {
	Name        string `json:"name"`
	ClusterName string `json:"clustername"`
	Status      string `json:"status"`
}

type PgClone struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ObjectMeta `json:"metadata"`

	Spec PgCloneSpec `json:"spec"`
}

type PgCloneList struct {
	metav1.TypeMeta `json:",inline"`
	Metadata        metav1.ListMeta `json:"metadata"`

	Items []PgClone `json:"items"`
}

func (e *PgClone) GetObjectKind() schema.ObjectKind {
	return &e.TypeMeta
}

func (e *PgClone) GetObjectMeta() metav1.Object {
	return &e.Metadata
}

func (el *PgCloneList) GetObjectKind() schema.ObjectKind {
	return &el.TypeMeta
}

func (el *PgCloneList) GetListMeta() metav1.List {
	return &el.Metadata
}

type PgCloneListCopy PgCloneList
type PgCloneCopy PgClone

func (e *PgClone) UnmarshalJSON(data []byte) error {
	tmp := PgCloneCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgClone(tmp)
	*e = tmp2
	return nil
}
func (el *PgCloneList) UnmarshalJSON(data []byte) error {
	tmp := PgCloneListCopy{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	tmp2 := PgCloneList(tmp)
	*el = tmp2
	return nil
}
