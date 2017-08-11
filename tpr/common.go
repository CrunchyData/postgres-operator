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

import ()

const PGROOT_SECRET_SUFFIX = "-pgroot-secret"
const PGUSER_SECRET_SUFFIX = "-pguser-secret"
const PGMASTER_SECRET_SUFFIX = "-pgmaster-secret"

const STORAGE_EXISTING = "existing"
const STORAGE_CREATE = "create"
const STORAGE_EMPTYDIR = "emptydir"
const STORAGE_DYNAMIC = "dynamic"

type PgStorageSpec struct {
	PvcName             string `json:"pvcname"`
	StorageClass        string `json:"storageclass"`
	PvcAccessMode       string `json:"pvcaccessmode"`
	PvcSize             string `json:"pvcsize"`
	StorageType         string `json:"storagetype"`
	FSGROUP             string `json:"fsgroup"`
	SUPPLEMENTAL_GROUPS string `json:"supplementalgroups"`
}
