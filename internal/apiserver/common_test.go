package apiserver

/*
Copyright 2018 - 2020 Crunchy Data Solutions, Inc.
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

	"k8s.io/apimachinery/pkg/api/resource"
)

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
