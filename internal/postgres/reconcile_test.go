/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgres

import (
	"fmt"
	"testing"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
	"gotest.tools/v3/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAddPGDATAVolumeToPod(t *testing.T) {

	postgresClusterBase := &v1alpha1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}
	testClaimName := "test-claim-name"

	testsCases := []struct {
		claimName  string
		containers []v1.Container
	}{{
		claimName:  testClaimName,
		containers: []v1.Container{{Name: "database"}, {Name: "pgbackrest"}},
	}, {
		claimName:  testClaimName,
		containers: []v1.Container{{Name: "database"}},
	}, {
		claimName:  testClaimName,
		containers: []v1.Container{},
	}, {
		claimName:  "", // should cause error
		containers: []v1.Container{},
	}}

	for _, tc := range testsCases {
		t.Run(fmt.Sprintf("claimName=%s, containers=%d", tc.claimName, len(tc.containers)), func(t *testing.T) {

			postgresCluster := postgresClusterBase.DeepCopy()
			template := &v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					Containers: tc.containers,
				},
			}

			err := AddPGDATAVolumeToPod(postgresCluster, template, tc.claimName,
				getContainerNames(tc.containers)...)
			if tc.claimName == "" {
				assert.ErrorContains(t, err, "must not be empty")
				return
			}
			assert.NilError(t, err)

			var foundPGDATAVol bool
			var pgdataVol *v1.Volume
			for i, v := range template.Spec.Volumes {
				if v.Name == naming.PGDATAVolume {
					foundPGDATAVol = true
					pgdataVol = &template.Spec.Volumes[i]
					break
				}
			}
			assert.Assert(t, foundPGDATAVol)
			assert.Assert(t, pgdataVol.PersistentVolumeClaim != nil)
			assert.Assert(t, (pgdataVol.PersistentVolumeClaim.ClaimName == tc.claimName))
		})
	}
}

func getContainerNames(containers []v1.Container) []string {
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	return names
}
