package util

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
	"testing"
)

func TestGenerateExporterSecretName(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		clusterName := "hippo"
		expected := clusterName + "-exporter-secret"
		actual := GenerateExporterSecretName(clusterName)

		if expected != actual {
			t.Fatalf("expected %q actual %q", expected, actual)
		}
	})
}
