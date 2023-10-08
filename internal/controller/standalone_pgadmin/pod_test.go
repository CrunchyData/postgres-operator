// Copyright 2023 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package standalone_pgadmin

import (
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPod(t *testing.T) {
	t.Parallel()

	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Name = "pgadmin"
	pgadmin.Spec.AdminUsername = "admin@pgo.com"
	config := new(corev1.ConfigMap)
	testpod := new(corev1.PodSpec)
	pvc := new(corev1.PersistentVolumeClaim)

	call := func() { pod(pgadmin, config, testpod, pvc) }

	t.Run("Defaults", func(t *testing.T) {

		call()

		assert.Assert(t, cmp.MarshalMatches(testpod, `
containers:
- command:
  - bash
  - -c
  - while true; do echo 'Hello!'; sleep 2; done
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin@pgo.com
  - name: PGADMIN_SETUP_PASSWORD
    valueFrom:
      secretKeyRef:
        key: password
        name: pgadmin-standalone-pgadmin
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin/conf.d
    name: standalone-pgadmin-config
    readOnly: true
  - mountPath: /var/lib/pgadmin
    name: standalone-pgadmin-data
volumes:
- name: standalone-pgadmin-data
  persistentVolumeClaim:
    claimName: ""
- name: standalone-pgadmin-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin.json
`))

		// No change when called again.
		before := testpod.DeepCopy()
		call()
		assert.DeepEqual(t, before, testpod)
	})

	t.Run("Customizations", func(t *testing.T) {
		pgadmin.Spec.ImagePullPolicy = corev1.PullAlways
		pgadmin.Spec.Image = initialize.String("new-image")
		pgadmin.Spec.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		}
		pgadmin.Spec.AdminUsername = "admin@pgo.com"

		call()

		assert.Assert(t, cmp.MarshalMatches(testpod, `
containers:
- command:
  - bash
  - -c
  - while true; do echo 'Hello!'; sleep 2; done
  env:
  - name: PGADMIN_SETUP_EMAIL
    value: admin@pgo.com
  - name: PGADMIN_SETUP_PASSWORD
    valueFrom:
      secretKeyRef:
        key: password
        name: pgadmin-standalone-pgadmin
  image: new-image
  imagePullPolicy: Always
  name: pgadmin
  ports:
  - containerPort: 5050
    name: pgadmin
    protocol: TCP
  resources:
    requests:
      cpu: 100m
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgadmin/conf.d
    name: standalone-pgadmin-config
    readOnly: true
  - mountPath: /var/lib/pgadmin
    name: standalone-pgadmin-data
volumes:
- name: standalone-pgadmin-data
  persistentVolumeClaim:
    claimName: ""
- name: standalone-pgadmin-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgadmin-settings.json
          path: ~postgres-operator/pgadmin.json
`))
	})
}

func TestPodConfigFiles(t *testing.T) {
	configmap := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "some-cm"}}

	pgadmin := v1beta1.PGAdmin{
		Spec: v1beta1.PGAdminSpec{
			Config: v1beta1.StandalonePGAdminConfiguration{Files: []corev1.VolumeProjection{{
				Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{
					Name: "test-secret",
				}},
			}, {
				ConfigMap: &corev1.ConfigMapProjection{LocalObjectReference: corev1.LocalObjectReference{
					Name: "test-cm",
				}},
			}}},
		},
	}

	projections := podConfigFiles(configmap, pgadmin)
	assert.Assert(t, cmp.MarshalMatches(projections, `
- secret:
    name: test-secret
- configMap:
    name: test-cm
- configMap:
    items:
    - key: pgadmin-settings.json
      path: ~postgres-operator/pgadmin.json
    name: some-cm
	`))
}
