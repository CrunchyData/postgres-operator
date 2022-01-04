package operator

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

	crv1 "github.com/crunchydata/postgres-operator/pkg/apis/crunchydata.com/v1"
)

func TestGetRepoType(t *testing.T) {
	cluster := &crv1.Pgcluster{
		Spec: crv1.PgclusterSpec{},
	}

	t.Run("empty list returns posix", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = make([]crv1.BackrestStorageType, 0)

		expected := crv1.BackrestStorageTypePosix
		actual := GetRepoType(cluster)
		if expected != actual {
			t.Fatalf("expected %q, actual %q", expected, actual)
		}
	})

	t.Run("multiple list returns posix", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeS3,
			crv1.BackrestStorageTypePosix,
		}

		expected := crv1.BackrestStorageTypePosix
		actual := GetRepoType(cluster)
		if expected != actual {
			t.Fatalf("expected %q, actual %q", expected, actual)
		}
	})

	t.Run("local returns posix", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
		}

		expected := crv1.BackrestStorageTypePosix
		actual := GetRepoType(cluster)
		if expected != actual {
			t.Fatalf("expected %q, actual %q", expected, actual)
		}
	})

	t.Run("posix returns posix", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
		}

		expected := crv1.BackrestStorageTypePosix
		actual := GetRepoType(cluster)
		if expected != actual {
			t.Fatalf("expected %q, actual %q", expected, actual)
		}
	})

	t.Run("s3 returns s3", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeS3,
		}

		expected := crv1.BackrestStorageTypeS3
		actual := GetRepoType(cluster)
		if expected != actual {
			t.Fatalf("expected %q, actual %q", expected, actual)
		}
	})
}

func TestIsLocalAndGCSStorage(t *testing.T) {
	cluster := &crv1.Pgcluster{
		Spec: crv1.PgclusterSpec{},
	}

	t.Run("empty list returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = make([]crv1.BackrestStorageType, 0)

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("posix only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
		}

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("local only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
		}

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("s3 only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeS3,
		}

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("gcs only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeGCS,
		}

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("posix and s3 returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
			crv1.BackrestStorageTypeS3,
		}

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("local and s3 returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
			crv1.BackrestStorageTypeS3,
		}

		expected := false
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("posix and gcs returns true", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
			crv1.BackrestStorageTypeGCS,
		}

		expected := true
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("local and gcs returns true", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
			crv1.BackrestStorageTypeGCS,
		}

		expected := true
		actual := IsLocalAndGCSStorage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})
}

func TestIsLocalAndS3Storage(t *testing.T) {
	cluster := &crv1.Pgcluster{
		Spec: crv1.PgclusterSpec{},
	}

	t.Run("empty list returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = make([]crv1.BackrestStorageType, 0)

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("posix only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
		}

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("local only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
		}

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("s3 only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeS3,
		}

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("gcs only returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeGCS,
		}

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("posix and s3 returns true", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
			crv1.BackrestStorageTypeS3,
		}

		expected := true
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("local and s3 returns true", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
			crv1.BackrestStorageTypeS3,
		}

		expected := true
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("posix and gcs returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypePosix,
			crv1.BackrestStorageTypeGCS,
		}

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})

	t.Run("local and gcs returns false", func(t *testing.T) {
		cluster.Spec.BackrestStorageTypes = []crv1.BackrestStorageType{
			crv1.BackrestStorageTypeLocal,
			crv1.BackrestStorageTypeGCS,
		}

		expected := false
		actual := IsLocalAndS3Storage(cluster)
		if expected != actual {
			t.Fatalf("expected %t, actual %t", expected, actual)
		}
	})
}
