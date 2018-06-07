package v1

/*
Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

import (
	"fmt"
)

// RootSecretSuffix ...
const RootSecretSuffix = "-postgres-secret"

// PrimarySecretSuffix ...
const PrimarySecretSuffix = "-primaryuser-secret"

// StorageExisting ...
const StorageExisting = "existing"

// StorageCreate ...
const StorageCreate = "create"

// StorageEmptydir ...
const StorageEmptydir = "emptydir"

// StorageDynamic ...
const StorageDynamic = "dynamic"

// PgStorageSpec ...
type PgStorageSpec struct {
	Name               string `json:"name"`
	StorageClass       string `json:"storageclass"`
	AccessMode         string `json:"accessmode"`
	Size               string `json:"size"`
	StorageType        string `json:"storagetype"`
	Fsgroup            string `json:"fsgroup"`
	SupplementalGroups string `json:"supplementalgroups"`
}

// PgContainerResource ...
type PgContainerResources struct {
	RequestsMemory string `json:"requestsmemory"`
	RequestsCPU    string `json:"requestscpu"`
	LimitsMemory   string `json:"limitsmemory"`
	LimitsCPU      string `json:"limitscpu"`
}

// UserSecretSuffix ...
func UserSecretSuffix(user string) string {
	if user == "" {
		user = "testuser"
	}
	return fmt.Sprintf("-%s-secret", user)
}
