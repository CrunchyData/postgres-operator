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
	"testing"

	pgpassword "github.com/crunchydata/postgres-operator/internal/postgres/password"
	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"

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

func TestValidateBackrestStorageTypeForCommand(t *testing.T) {
	cluster := &crv1.Pgcluster{
		Spec: crv1.PgclusterSpec{},
	}

	t.Run("empty repo type", func(t *testing.T) {
		err := ValidateBackrestStorageTypeForCommand(cluster, "")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("invalid repo type", func(t *testing.T) {
		err := ValidateBackrestStorageTypeForCommand(cluster, "bad")

		if err == nil {
			t.Fatalf("expected invalid repo type to return an error, no error returned")
		}
	})

	t.Run("multiple repo types", func(t *testing.T) {
		err := ValidateBackrestStorageTypeForCommand(cluster, "posix,s3")

		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("posix repo, no repo types on resource", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{}
		err := ValidateBackrestStorageTypeForCommand(cluster, "posix")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("local repo, no repo types on resource", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{}
		err := ValidateBackrestStorageTypeForCommand(cluster, "local")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("s3 repo, no repo types on resource", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{}
		err := ValidateBackrestStorageTypeForCommand(cluster, "s3")

		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("posix repo, posix repo type available", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypePosix}
		err := ValidateBackrestStorageTypeForCommand(cluster, "posix")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("posix repo, posix repo type unavailable", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypeS3}
		err := ValidateBackrestStorageTypeForCommand(cluster, "posix")

		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("posix repo, local repo type available", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypeLocal}
		err := ValidateBackrestStorageTypeForCommand(cluster, "posix")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("posix repo, multi-repo", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeS3,
			crv1.BackrestStorageTypePosix,
		}
		err := ValidateBackrestStorageTypeForCommand(cluster, "posix")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("local repo, local repo type available", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypeLocal}
		err := ValidateBackrestStorageTypeForCommand(cluster, "local")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("local repo, local repo type unavailable", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypeS3}
		err := ValidateBackrestStorageTypeForCommand(cluster, "local")

		if err == nil {
			t.Fatalf("expected error")
		}
	})

	t.Run("local repo, posix repo type available", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypePosix}
		err := ValidateBackrestStorageTypeForCommand(cluster, "local")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("s3 repo, s3 repo type available", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypeS3}
		err := ValidateBackrestStorageTypeForCommand(cluster, "s3")

		if err != nil {
			t.Fatalf("expected no error, actual error: %s", err.Error())
		}
	})

	t.Run("s3 repo, s3 repo type unavailable", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{crv1.BackrestStorageTypePosix}
		err := ValidateBackrestStorageTypeForCommand(cluster, "s3")

		if err == nil {
			t.Fatalf("expected error")
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
