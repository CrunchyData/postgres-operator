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
	"encoding/json"
	"testing"

	v1 "k8s.io/api/core/v1"
)

func TestStorageResultInlineVolumeSource(t *testing.T) {
	if b, _ := json.Marshal(v1.VolumeSource{}); string(b) != "{}" {
		t.Logf("expected VolumeSource to always marshal with brackets, got %q", b)
	}

	for _, tt := range []struct {
		value    StorageResult
		expected string
	}{
		{StorageResult{}, `"emptyDir":{}`},
		{
			StorageResult{PersistentVolumeClaimName: "<\x00"},
			`"persistentVolumeClaim":{"claimName":"<\u0000"}`,
		},
		{
			StorageResult{PersistentVolumeClaimName: "some-name"},
			`"persistentVolumeClaim":{"claimName":"some-name"}`,
		},
	} {
		if actual := tt.value.InlineVolumeSource(); actual != tt.expected {
			t.Errorf("expected %q for %v, got %q", tt.expected, tt.value, actual)
		}
	}
}
