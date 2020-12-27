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
	"fmt"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
