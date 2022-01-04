package cluster

/*
 Copyright 2020 - 2022 Crunchy Data Solutions, Inc.
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

	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
)

func TestMakePostgresPassword(t *testing.T) {

	t.Run("md5", func(t *testing.T) {
		t.Run("valid", func(t *testing.T) {
			passwordType := pgpassword.MD5
			password := "datalake"
			expected := "md56294153764d389dc6830b6ce4f923cdb"

			actual := makePostgresPassword(passwordType, password)

			if actual != expected {
				t.Errorf("expected: %q actual: %q", expected, actual)
			}
		})

	})
}
