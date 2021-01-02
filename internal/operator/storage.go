package operator

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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
	"bytes"
	"encoding/json"

	v1 "k8s.io/api/core/v1"
)

// StorageResult is a resolved PgStorageSpec. The zero value is an emptyDir.
type StorageResult struct {
	PersistentVolumeClaimName string
	SupplementalGroups        []int64
}

// InlineVolumeSource returns the key and value of a k8s.io/api/core/v1.VolumeSource.
func (s StorageResult) InlineVolumeSource() string {
	b := new(bytes.Buffer)
	e := json.NewEncoder(b)
	e.SetEscapeHTML(false)
	_ = e.Encode(s.VolumeSource())

	// remove trailing newline and surrounding brackets
	return b.String()[1 : b.Len()-2]
}

// VolumeSource returns the VolumeSource equivalent of s.
func (s StorageResult) VolumeSource() v1.VolumeSource {
	if s.PersistentVolumeClaimName != "" {
		return v1.VolumeSource{
			PersistentVolumeClaim: &v1.PersistentVolumeClaimVolumeSource{
				ClaimName: s.PersistentVolumeClaimName,
			},
		}
	}

	return v1.VolumeSource{EmptyDir: &v1.EmptyDirVolumeSource{}}
}
