package util

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
	"reflect"
	"testing"

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
	v1 "k8s.io/api/core/v1"
)

func TestGenerateNodeAffinity(t *testing.T) {
	// presently this test is really strict. as we allow for more options, we will
	// need to add more tests.
	t.Run("valid", func(t *testing.T) {
		key := "foo"
		values := []string{"bar", "baz"}

		affinity := GenerateNodeAffinity(key, values)

		if len(affinity.PreferredDuringSchedulingIgnoredDuringExecution) == 0 {
			t.Fatalf("expected preferred node affinity to be set")
		} else if len(affinity.PreferredDuringSchedulingIgnoredDuringExecution) > 1 {
			t.Fatalf("only expected one rule to be set")
		}

		term := affinity.PreferredDuringSchedulingIgnoredDuringExecution[0]

		if term.Weight != crv1.NodeAffinityDefaultWeight {
			t.Fatalf("expected weight %d actual %d", crv1.NodeAffinityDefaultWeight, term.Weight)
		}

		if len(term.Preference.MatchExpressions) == 0 {
			t.Fatalf("expected a match expression to be set")
		} else if len(term.Preference.MatchExpressions) > 1 {
			t.Fatalf("expected only one match expression to be set")
		}

		rule := term.Preference.MatchExpressions[0]

		if rule.Operator != v1.NodeSelectorOpIn {
			t.Fatalf("operator expected %s actual %s", v1.NodeSelectorOpIn, rule.Operator)
		}

		if rule.Key != key {
			t.Fatalf("key expected %s actual %s", key, rule.Key)
		}

		if !reflect.DeepEqual(rule.Values, values) {
			t.Fatalf("values expected %v actual %v", values, rule.Values)
		}
	})
}
