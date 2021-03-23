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

package patroni

import (
	"context"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1alpha1"
)

func TestClusterConfigMap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := new(v1alpha1.PostgresCluster)
	cluster.Default()
	config := new(v1.ConfigMap)
	pgHBAs := postgres.HBAs{}
	pgParameters := postgres.Parameters{}
	pgUser := new(v1.Secret)

	assert.NilError(t, ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters, pgUser, config))

	// The output of clusterYAML should go into config.
	data, _ := clusterYAML(cluster, pgUser, pgHBAs, pgParameters)
	assert.DeepEqual(t, config.Data["patroni.yaml"], data)

	// No change when called again.
	before := config.DeepCopy()
	assert.NilError(t, ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters, pgUser, config))
	assert.DeepEqual(t, config, before)
}

func TestReconcileInstanceCertificates(t *testing.T) {
	t.Parallel()

	root := pki.NewRootCertificateAuthority()
	assert.NilError(t, root.Generate(), "bug in test")

	leaf := pki.NewLeafCertificate("any", nil, nil)
	assert.NilError(t, leaf.Generate(root), "bug in test")

	ctx := context.Background()
	secret := new(v1.Secret)
	cert := leaf.Certificate
	key := leaf.PrivateKey

	dataCA, _ := certAuthorities(root.Certificate)
	dataCert, _ := certFile(key, cert)

	assert.NilError(t, InstanceCertificates(ctx, root.Certificate, cert, key, secret))

	assert.DeepEqual(t, secret.Data["patroni.ca-roots"], dataCA)
	assert.DeepEqual(t, secret.Data["patroni.crt-combined"], dataCert)

	// No change when called again.
	before := secret.DeepCopy()
	assert.NilError(t, InstanceCertificates(ctx, root.Certificate, cert, key, secret))
	assert.DeepEqual(t, secret, before)
}

func TestInstanceConfigMap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := new(v1alpha1.PostgresCluster)
	instance := new(appsv1.StatefulSet)
	config := new(v1.ConfigMap)
	data, _ := instanceYAML(cluster, instance)

	assert.NilError(t, InstanceConfigMap(ctx, cluster, instance, config))

	assert.DeepEqual(t, config.Data["patroni.yaml"], data)

	// No change when called again.
	before := config.DeepCopy()
	assert.NilError(t, InstanceConfigMap(ctx, cluster, instance, config))
	assert.DeepEqual(t, config, before)
}

func TestInstancePod(t *testing.T) {
	t.Parallel()

	cluster := new(v1alpha1.PostgresCluster)
	cluster.Default()
	cluster.Name = "some-such"
	clusterConfigMap := new(v1.ConfigMap)
	clusterPodService := new(v1.Service)
	instanceCertficates := new(v1.Secret)
	instanceConfigMap := new(v1.ConfigMap)
	patroniLeaderService := new(v1.Service)
	template := new(v1.PodTemplateSpec)

	call := func() error {
		return InstancePod(context.Background(),
			cluster, clusterConfigMap, clusterPodService, patroniLeaderService,
			instanceCertficates, instanceConfigMap, template)
	}

	assert.NilError(t, call())

	assert.DeepEqual(t, template.ObjectMeta, metav1.ObjectMeta{
		Labels: map[string]string{naming.LabelPatroni: "some-such-ha"},
	})

	assert.Assert(t, marshalEquals(template.Spec, strings.TrimSpace(`
containers:
- args:
  - patroni
  - /etc/patroni
  env:
  - name: PATRONI_NAME
    valueFrom:
      fieldRef:
        apiVersion: v1
        fieldPath: metadata.name
  - name: PATRONI_KUBERNETES_POD_IP
    valueFrom:
      fieldRef:
        apiVersion: v1
        fieldPath: status.podIP
  - name: PATRONI_KUBERNETES_PORTS
    value: |
      []
  - name: PATRONI_POSTGRESQL_CONNECT_ADDRESS
    value: $(PATRONI_NAME).:5432
  - name: PATRONI_POSTGRESQL_LISTEN
    value: '*:5432'
  - name: PATRONI_RESTAPI_CONNECT_ADDRESS
    value: $(PATRONI_NAME).:8008
  - name: PATRONI_RESTAPI_LISTEN
    value: '*:8008'
  - name: PATRONICTL_CONFIG_FILE
    value: /etc/patroni
  - name: PGHOST
    value: /tmp
  livenessProbe:
    failureThreshold: 3
    httpGet:
      path: /liveness
      port: 8008
      scheme: HTTPS
    initialDelaySeconds: 3
    periodSeconds: 10
    successThreshold: 1
    timeoutSeconds: 5
  name: database
  readinessProbe:
    failureThreshold: 3
    httpGet:
      path: /readiness
      port: 8008
      scheme: HTTPS
    initialDelaySeconds: 3
    periodSeconds: 10
    successThreshold: 1
    timeoutSeconds: 5
  resources: {}
  volumeMounts:
  - mountPath: /etc/patroni
    name: patroni-config
    readOnly: true
volumes:
- name: patroni-config
  projected:
    sources:
    - configMap:
        items:
        - key: patroni.yaml
          path: ~postgres-operator_cluster.yaml
    - configMap:
        items:
        - key: patroni.yaml
          path: ~postgres-operator_instance.yaml
    - secret:
        items:
        - key: patroni.ca-roots
          path: ~postgres-operator/patroni.ca-roots
        - key: patroni.crt-combined
          path: ~postgres-operator/patroni.crt+key
	`)+"\n"))

	// No change when called again.
	before := template.DeepCopy()
	assert.NilError(t, call())
	assert.DeepEqual(t, template, before)

	t.Run("ExistingEnvironment", func(t *testing.T) {
		template.Spec.Containers[0].Env = []v1.EnvVar{
			{Name: "existed"},
			{Name: "PATRONI_KUBERNETES_POD_IP"},
			{Name: "also", Value: "kept"},
		}

		assert.NilError(t, call())

		// Correct values are there and in order.
		assert.Assert(t, marshalContains(template.Spec.Containers[0].Env,
			strings.TrimSpace(`
- name: PATRONI_NAME
  valueFrom:
    fieldRef:
      apiVersion: v1
      fieldPath: metadata.name
- name: PATRONI_KUBERNETES_POD_IP
  valueFrom:
    fieldRef:
      apiVersion: v1
      fieldPath: status.podIP
			`)+"\n"))

		// Existing values are there and in the original order.
		assert.Assert(t, marshalContains(template.Spec.Containers[0].Env,
			strings.TrimSpace(`
- name: existed
- name: also
  value: kept
			`)+"\n"))

		// Correct values can be in the middle somewhere.
		template.Spec.Containers[0].Env = append(template.Spec.Containers[0].Env,
			v1.EnvVar{Name: "at", Value: "end"})

		// No change when already correct.
		before := template.DeepCopy()
		assert.NilError(t, call())
		assert.DeepEqual(t, template, before)
	})

	t.Run("ExistingVolumes", func(t *testing.T) {
		template.Spec.Volumes = []v1.Volume{
			{Name: "existing"},
			{Name: "patroni-config", VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{Medium: "Memory"},
			}},
		}

		assert.NilError(t, call())

		// Correct values are there.
		assert.Assert(t, marshalContains(template.Spec.Volumes,
			strings.TrimSpace(`
- name: patroni-config
  projected:
    sources:
    - configMap:
        items:
        - key: patroni.yaml
			`)+"\n"))

		// Existing values are there.
		assert.Assert(t, marshalContains(template.Spec.Volumes,
			strings.TrimSpace(`
- name: existing
			`)+"\n"))

		// Correct values can be in the middle somewhere.
		template.Spec.Volumes = append(template.Spec.Volumes,
			v1.Volume{Name: "later"})

		// No change when already correct.
		before := template.DeepCopy()
		assert.NilError(t, call())
		assert.DeepEqual(t, template, before)
	})

	t.Run("ExistingVolumeMounts", func(t *testing.T) {
		template.Spec.Containers[0].VolumeMounts = []v1.VolumeMount{
			{Name: "existing", MountPath: "mount"},
			{Name: "patroni-config", MountPath: "wrong"},
		}

		assert.NilError(t, call())

		// Correct values are there.
		assert.Assert(t, marshalContains(template.Spec.Containers[0].VolumeMounts,
			strings.TrimSpace(`
- mountPath: /etc/patroni
  name: patroni-config
  readOnly: true
			`)+"\n"))

		// Existing values are there.
		assert.Assert(t, marshalContains(template.Spec.Containers[0].VolumeMounts,
			strings.TrimSpace(`
- mountPath: mount
  name: existing
			`)+"\n"))

		// Correct values can be in the middle somewhere.
		template.Spec.Containers[0].VolumeMounts = append(
			template.Spec.Containers[0].VolumeMounts, v1.VolumeMount{Name: "later"})

		// No change when already correct.
		before := template.DeepCopy()
		assert.NilError(t, call())
		assert.DeepEqual(t, template, before)
	})
}
