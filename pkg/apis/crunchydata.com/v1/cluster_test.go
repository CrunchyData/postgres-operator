package v1

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPgClusterCurrentPrimary(t *testing.T) {
	t.Run("is set", func(t *testing.T) {
		currentPrimary := "abc"
		cluster := &Pgcluster{
			ObjectMeta: meta_v1.ObjectMeta{
				Annotations: map[string]string{
					PgclusterCurrentPrimary: currentPrimary,
				},
			},
		}

		if cluster.CurrentPrimary() != currentPrimary {
			t.Errorf("expected CurrentPrimary to return %q", currentPrimary)
		}
	})

	t.Run("is unset", func(t *testing.T) {
		cluster := &Pgcluster{
			ObjectMeta: meta_v1.ObjectMeta{
				Annotations: map[string]string{},
			},
		}

		if cluster.CurrentPrimary() != "" {
			t.Error("expected CurrentPrimary to return nothing")
		}
	})
}
