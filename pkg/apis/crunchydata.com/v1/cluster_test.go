package v1

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
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestParseBackrestStorageTypes(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		_, err := ParseBackrestStorageTypes("")

		if !errors.Is(err, ErrStorageTypesEmpty) {
			t.Fatalf("expected ErrStorageTypesEmpty actual %q", err.Error())
		}
	})

	t.Run("invalid", func(t *testing.T) {
		_, err := ParseBackrestStorageTypes("bad bad bad")

		if !errors.Is(err, ErrInvalidStorageType) {
			t.Fatalf("expected ErrInvalidStorageType actual %q", err.Error())
		}

		_, err = ParseBackrestStorageTypes("posix,bad")

		if !errors.Is(err, ErrInvalidStorageType) {
			t.Fatalf("expected ErrInvalidStorageType actual %q", err.Error())
		}
	})

	t.Run("local should be posix", func(t *testing.T) {
		storageTypes, err := ParseBackrestStorageTypes("local")

		if err != nil {
			t.Fatalf("expected no error actual %q", err.Error())
		}

		if len(storageTypes) != 1 {
			t.Fatalf("multiple storage types returned, expected 1")
		}

		if storageTypes[0] != BackrestStorageTypePosix {
			t.Fatalf("posix expected but not found")
		}
	})

	t.Run("posix", func(t *testing.T) {
		storageTypes, err := ParseBackrestStorageTypes("posix")

		if err != nil {
			t.Fatalf("expected no error actual %q", err.Error())
		}

		if len(storageTypes) != 1 {
			t.Fatalf("multiple storage types returned, expected 1")
		}

		if storageTypes[0] != BackrestStorageTypePosix {
			t.Fatalf("posix expected but not found")
		}
	})

	t.Run("gcs", func(t *testing.T) {
		storageTypes, err := ParseBackrestStorageTypes("gcs")

		if err != nil {
			t.Fatalf("expected no error actual %q", err.Error())
		}

		if len(storageTypes) != 1 {
			t.Fatalf("multiple storage types returned, expected 1")
		}

		if storageTypes[0] != BackrestStorageTypeGCS {
			t.Fatalf("gcs expected but not found")
		}
	})

	t.Run("s3", func(t *testing.T) {
		storageTypes, err := ParseBackrestStorageTypes("s3")

		if err != nil {
			t.Fatalf("expected no error actual %q", err.Error())
		}

		if len(storageTypes) != 1 {
			t.Fatalf("multiple storage types returned, expected 1")
		}

		if storageTypes[0] != BackrestStorageTypeS3 {
			t.Fatalf("s3 expected but not found")
		}
	})

	t.Run("posix and s3", func(t *testing.T) {
		storageTypes, err := ParseBackrestStorageTypes("posix,s3")

		if err != nil {
			t.Fatalf("expected no error actual %q", err.Error())
		}

		if len(storageTypes) != 2 {
			t.Fatalf("expected 2 storage types, actual %d", len(storageTypes))
		}

		posix := false
		s3 := false
		for _, storageType := range storageTypes {
			posix = posix || (storageType == BackrestStorageTypePosix)
			s3 = s3 || (storageType == BackrestStorageTypeS3)
		}

		if !(posix && s3) {
			t.Fatalf("posix and s3 expected but not found")
		}
	})

	t.Run("local and s3", func(t *testing.T) {
		storageTypes, err := ParseBackrestStorageTypes("local,s3")

		if err != nil {
			t.Fatalf("expected no error actual %q", err.Error())
		}

		if len(storageTypes) != 2 {
			t.Fatalf("expected 2 storage types, actual %d", len(storageTypes))
		}

		posix := false
		s3 := false
		for _, storageType := range storageTypes {
			posix = posix || (storageType == BackrestStorageTypePosix)
			s3 = s3 || (storageType == BackrestStorageTypeS3)
		}

		if !(posix && s3) {
			t.Fatalf("posix and s3 expected but not found")
		}
	})
}

func TestUserSecretName(t *testing.T) {
	cluster := &Pgcluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "second-pick",
		},
		Spec: PgclusterSpec{
			ClusterName: "second-pick",
			User:        "puppy",
		},
	}

	t.Run(PGUserMonitor, func(t *testing.T) {
		expected := fmt.Sprintf("%s-%s-secret", cluster.Name, "exporter")
		actual := UserSecretName(cluster, PGUserMonitor)
		if expected != actual {
			t.Fatalf("expected %q, got %q", expected, actual)
		}
	})

	t.Run("any other user", func(t *testing.T) {
		for _, user := range []string{PGUserSuperuser, PGUserReplication, cluster.Spec.User} {
			expected := fmt.Sprintf("%s-%s-secret", cluster.Name, user)
			actual := UserSecretName(cluster, user)
			if expected != actual {
				t.Fatalf("expected %q, got %q", expected, actual)
			}
		}
	})
}

func TestUserSecretNameFromClusterName(t *testing.T) {
	clusterName := "second-pick"

	t.Run(PGUserMonitor, func(t *testing.T) {
		expected := fmt.Sprintf("%s-%s-secret", clusterName, "exporter")
		actual := UserSecretNameFromClusterName(clusterName, PGUserMonitor)
		if expected != actual {
			t.Fatalf("expected %q, got %q", expected, actual)
		}
	})

	t.Run("any other user", func(t *testing.T) {
		for _, user := range []string{PGUserSuperuser, PGUserReplication, "puppy"} {
			expected := fmt.Sprintf("%s-%s-secret", clusterName, user)
			actual := UserSecretNameFromClusterName(clusterName, user)
			if expected != actual {
				t.Fatalf("expected %q, got %q", expected, actual)
			}
		}
	})
}
