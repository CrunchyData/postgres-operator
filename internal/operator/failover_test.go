package operator

/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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
	"reflect"
	"testing"
)

func TestGeneratePostgresFailoverCommand(t *testing.T) {
	clusterName := "hippo"
	candidate := ""

	t.Run("no candidate", func(t *testing.T) {
		expected := []string{"patronictl", "failover", "--force", clusterName}
		actual := generatePostgresFailoverCommand(clusterName, candidate)

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("expected: %v actual: %v", expected, actual)
		}
	})

	t.Run("candidate", func(t *testing.T) {
		candidate = "hippo-abc-123"
		expected := []string{"patronictl", "failover", "--force", clusterName, "--candidate", candidate}
		actual := generatePostgresFailoverCommand(clusterName, candidate)

		if !reflect.DeepEqual(expected, actual) {
			t.Fatalf("expected: %v actual: %v", expected, actual)
		}
	})
}
