package apiserver

/*
Copyright 2018 - 2022 Crunchy Data Solutions, Inc.
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
	"fmt"
	"strings"
	"testing"

	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"

	"k8s.io/apimachinery/pkg/api/resource"
)

func TestGetPasswordType(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		tests := map[string]pgpassword.PasswordType{
			"":              pgpassword.MD5,
			"md5":           pgpassword.MD5,
			"scram":         pgpassword.SCRAM,
			"scram-sha-256": pgpassword.SCRAM,
		}

		for passwordTypeStr, expected := range tests {
			t.Run(passwordTypeStr, func(t *testing.T) {
				passwordType, err := GetPasswordType(passwordTypeStr)
				if err != nil {
					t.Error(err)
					return
				}

				if passwordType != expected {
					t.Errorf("password type %q should yield %d", passwordTypeStr, expected)
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		tests := map[string]error{
			"magic":         ErrPasswordTypeInvalid,
			"scram-sha-512": ErrPasswordTypeInvalid,
		}

		for passwordTypeStr, expected := range tests {
			t.Run(passwordTypeStr, func(t *testing.T) {
				if _, err := GetPasswordType(passwordTypeStr); !errors.Is(err, expected) {
					t.Errorf("password type %q should yield error %q", passwordTypeStr, expected.Error())
				}
			})
		}
	})
}

func TestValidateLabel(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		inputs := []map[string]string{
			map[string]string{"key": "value"},
			map[string]string{"example.com/key": "value"},
			map[string]string{"key1": "value1", "key2": "value2"},
			map[string]string{"": ""},
		}

		for _, input := range inputs {
			labelStr := ""

			for k, v := range input {
				if k == "" && v == "" {
					continue
				}

				labelStr += fmt.Sprintf("%s=%s,", k, v)
			}

			labelStr = strings.Trim(labelStr, ",")

			t.Run(labelStr, func(*testing.T) {
				labels, err := ValidateLabel(labelStr)

				if err != nil {
					t.Fatalf("expected no error, got: %s", err.Error())
				}

				for k := range labels {
					if v, ok := input[k]; !(ok || v == labels[k]) {
						t.Fatalf("label values do not matched (%s vs. %s)", input[k], labels[k])
					}
				}
			})
		}
	})

	t.Run("invalid", func(t *testing.T) {
		inputs := []string{
			"key",
			"key=value=value",
			"key=value,",
			"b@d=value",
			"b@d-prefix/key=value",
			"really/bad/prefix/key=value",
			"key=v\\alue",
		}

		for _, input := range inputs {
			t.Run(input, func(t *testing.T) {
				_, err := ValidateLabel(input)

				if err == nil {
					t.Fatalf("expected an invalid input error.")
				}

				if !errors.Is(err, ErrLabelInvalid) {
					t.Fatalf("expected an ErrLabelInvalid error.")
				}
			})
		}
	})
}

func TestValidateResourceRequestLimit(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		resources := []struct{ request, limit, defaultRequest string }{
			{"", "", "0"},
			{"256Mi", "256Mi", "128Mi"},
			{"", "256Mi", "128Mi"},
			{"", "256Mi", "0"},
			{"256Mi", "", "128Mi"},
			{"64Mi", "", "128Mi"},
			{"256Mi", "", "0"},
			{"", "", "128Mi"},
		}

		for _, r := range resources {
			defaultQuantity := resource.MustParse(r.defaultRequest)

			if err := ValidateResourceRequestLimit(r.request, r.limit, defaultQuantity); err != nil {
				t.Fatal(err)
				return
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		resources := []struct{ request, limit, defaultRequest string }{
			{"broken", "3000 Gigabytes", "128Mi"},
			{"256Mi", "3000 Gigabytes", "128Mi"},
			{"broken", "256Mi", "128Mi"},
			{"256Mi", "128Mi", "512Mi"},
			{"", "128Mi", "512Mi"},
		}

		for _, r := range resources {
			defaultQuantity := resource.MustParse(r.defaultRequest)

			if err := ValidateResourceRequestLimit(r.request, r.limit, defaultQuantity); err == nil {
				t.Fatalf("expected error with values %v", r)
				return
			}
		}
	})
}

func TestValidateQuantity(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		quantities := []string{
			"",
			"100Mi",
			"100M",
			"250Gi",
			"25G",
			"0.1",
			"1.2",
			"150m",
		}

		for _, quantity := range quantities {
			if err := ValidateQuantity(quantity); err != nil {
				t.Fatal(err)
				return
			}
		}
	})

	t.Run("invalid", func(t *testing.T) {
		quantities := []string{
			"broken",
			"3000 Gigabytes",
		}

		for _, quantity := range quantities {
			if err := ValidateQuantity(quantity); err == nil {
				t.Fatalf("expected error with value %q", quantity)
				return
			}
		}
	})
}
