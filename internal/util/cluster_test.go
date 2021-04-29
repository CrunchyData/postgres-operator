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
	"errors"
	"reflect"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/config"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
)

func TestGenerateNodeAffinity(t *testing.T) {
	// presently only one rule is allowed, so we are testing for that. future
	// tests may need to expand upon that
	t.Run("preferred", func(t *testing.T) {
		affinityType := crv1.NodeAffinityTypePreferred
		key := "foo"
		values := []string{"bar", "baz"}

		affinity := GenerateNodeAffinity(affinityType, key, values)

		if affinity.RequiredDuringSchedulingIgnoredDuringExecution != nil {
			t.Fatalf("expected required node affinity to not be set")
		}

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

	t.Run("required", func(t *testing.T) {
		affinityType := crv1.NodeAffinityTypeRequired
		key := "foo"
		values := []string{"bar", "baz"}

		affinity := GenerateNodeAffinity(affinityType, key, values)

		if len(affinity.PreferredDuringSchedulingIgnoredDuringExecution) != 0 {
			t.Fatalf("expected preferred node affinity to not be set")
		}

		if affinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
			t.Fatalf("expected required node affinity to be set")
		}

		if len(affinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) == 0 {
			t.Fatalf("expected required node affinity to have at least one rule.")
		} else if len(affinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms) > 1 {
			t.Fatalf("expected required node affinity to have only one rule.")
		}

		term := affinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms[0]

		if len(term.MatchExpressions) == 0 {
			t.Fatalf("expected a match expression to be set")
		} else if len(term.MatchExpressions) > 1 {
			t.Fatalf("expected only one match expression to be set")
		}

		rule := term.MatchExpressions[0]

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

func TestGetCustomLabels(t *testing.T) {
	cluster := &crv1.Pgcluster{}

	t.Run("labels empty", func(t *testing.T) {
		if len(GetCustomLabels(cluster)) != 0 {
			t.Fatal("expected no labels to be returned")
		}

		cluster.Spec.UserLabels = map[string]string{}

		if len(GetCustomLabels(cluster)) != 0 {
			t.Fatal("expected no labels to be returned")
		}
	})

	t.Run("excluded labels are excluded", func(t *testing.T) {
		cluster.Spec.UserLabels = map[string]string{
			config.LABEL_PGO_VERSION: "0.0.1",
			config.LABEL_WORKFLOW_ID: "abcd",
			"custom":                 "label",
			"hippo":                  "cool",
		}
		expectedLen := len(cluster.Spec.UserLabels) - 2

		labels := GetCustomLabels(cluster)
		if len(labels) != expectedLen {
			t.Fatal("expected only two labels to be returned")
		}

		for k := range labels {
			switch k {
			default:
				continue
			case config.LABEL_PGO_VERSION, config.LABEL_WORKFLOW_ID:
				t.Fatalf("expected label %q to not be present", k)
			}
		}
	})

	t.Run("does not modify original map", func(t *testing.T) {
		cluster.Spec.UserLabels = map[string]string{
			config.LABEL_PGO_VERSION: "0.0.1",
			config.LABEL_WORKFLOW_ID: "abcd",
			"custom":                 "label",
			"hippo":                  "cool",
		}
		origLen := len(cluster.Spec.UserLabels)

		_ = GetCustomLabels(cluster)

		if len(cluster.Spec.UserLabels) != origLen {
			t.Fatal("expected custom labels to not be modified")
		}
	})
}

func TestValidateLabels(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		inputs := []map[string]string{
			{"key": "value"},
			{"example.com/key": "value"},
			{"key1": "value1", "key2": "value2"},
		}

		for _, input := range inputs {
			t.Run(labels.FormatLabels(input), func(*testing.T) {
				err := ValidateLabels(input)

				if err != nil {
					t.Fatalf("expected no error, got: %s", err.Error())
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		inputs := []map[string]string{
			{"key=value": "value"},
			{"key": "value", "": ""},
			{"b@d": "value"},
			{"b@d-prefix/key": "value"},
			{"really/bad/prefix/key": "value"},
			{"key": "v\\alue"},
		}

		for _, input := range inputs {
			t.Run(labels.FormatLabels(input), func(t *testing.T) {
				err := ValidateLabels(input)

				if !errors.Is(err, ErrLabelInvalid) {
					t.Fatalf("expected an ErrLabelInvalid error, got %T: %v", err, err)
				}
			})
		}
	})
}
