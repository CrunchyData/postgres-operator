// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package patroni

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestClusterConfigMap(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	cluster := new(v1beta1.PostgresCluster)
	pgHBAs := postgres.HBAs{}
	pgParameters := postgres.Parameters{}

	cluster.Default()
	config := new(corev1.ConfigMap)
	assert.NilError(t, ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters, config))

	// The output of clusterYAML should go into config.
	data, _ := clusterYAML(cluster, pgHBAs, pgParameters)
	assert.DeepEqual(t, config.Data["patroni.yaml"], data)

	// No change when called again.
	before := config.DeepCopy()
	assert.NilError(t, ClusterConfigMap(ctx, cluster, pgHBAs, pgParameters, config))
	assert.DeepEqual(t, config, before)
}

func TestReconcileInstanceCertificates(t *testing.T) {
	t.Parallel()

	root, err := pki.NewRootCertificateAuthority()
	assert.NilError(t, err, "bug in test")

	leaf, err := root.GenerateLeafCertificate("any", nil)
	assert.NilError(t, err, "bug in test")

	dataCA, _ := certFile(root.Certificate)
	assert.Assert(t,
		cmp.Regexp(`^`+
			`-----BEGIN CERTIFICATE-----\n`+
			`([^-]+\n)+`+
			`-----END CERTIFICATE-----\n`+
			`$`, string(dataCA),
		),
		"expected a PEM-encoded certificate bundle")

	dataCert, _ := certFile(leaf.PrivateKey, leaf.Certificate)
	assert.Assert(t,
		cmp.Regexp(`^`+
			`-----BEGIN [^ ]+ PRIVATE KEY-----\n`+
			`([^-]+\n)+`+
			`-----END [^ ]+ PRIVATE KEY-----\n`+
			`-----BEGIN CERTIFICATE-----\n`+
			`([^-]+\n)+`+
			`-----END CERTIFICATE-----\n`+
			`$`, string(dataCert),
		),
		// - https://docs.python.org/3/library/ssl.html#combined-key-and-certificate
		// - https://docs.python.org/3/library/ssl.html#certificate-chains
		"expected a PEM-encoded key followed by the certificate")

	ctx := context.Background()
	secret := new(corev1.Secret)

	assert.NilError(t, InstanceCertificates(ctx,
		root.Certificate, leaf.Certificate, leaf.PrivateKey, secret))

	assert.DeepEqual(t, secret.Data["patroni.ca-roots"], dataCA)
	assert.DeepEqual(t, secret.Data["patroni.crt-combined"], dataCert)

	// No change when called again.
	before := secret.DeepCopy()
	assert.NilError(t, InstanceCertificates(ctx,
		root.Certificate, leaf.Certificate, leaf.PrivateKey, secret))
	assert.DeepEqual(t, secret, before)
}

func TestInstanceConfigMap(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := new(v1beta1.PostgresCluster)
	instance := new(v1beta1.PostgresInstanceSetSpec)
	config := new(corev1.ConfigMap)
	data, _ := instanceYAML(cluster, instance, nil)

	assert.NilError(t, InstanceConfigMap(ctx, cluster, instance, config))

	assert.DeepEqual(t, config.Data["patroni.yaml"], data)

	// No change when called again.
	before := config.DeepCopy()
	assert.NilError(t, InstanceConfigMap(ctx, cluster, instance, config))
	assert.DeepEqual(t, config, before)
}

func TestInstancePod(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	cluster.Default()
	cluster.Name = "some-such"
	cluster.Spec.PostgresVersion = 11
	cluster.Spec.Image = "image"
	cluster.Spec.ImagePullPolicy = corev1.PullAlways
	clusterConfigMap := new(corev1.ConfigMap)
	clusterPodService := new(corev1.Service)
	instanceCertificates := new(corev1.Secret)
	instanceConfigMap := new(corev1.ConfigMap)
	instanceSpec := new(v1beta1.PostgresInstanceSetSpec)
	patroniLeaderService := new(corev1.Service)
	template := new(corev1.PodTemplateSpec)
	template.Spec.Containers = []corev1.Container{{Name: "database"}}

	call := func() error {
		return InstancePod(context.Background(),
			cluster, clusterConfigMap, clusterPodService, patroniLeaderService,
			instanceSpec, instanceCertificates, instanceConfigMap, template)
	}

	assert.NilError(t, call())

	assert.DeepEqual(t, template.ObjectMeta, metav1.ObjectMeta{
		Labels: map[string]string{naming.LabelPatroni: "some-such-ha"},
	})

	assert.Assert(t, cmp.MarshalMatches(template.Spec, `
containers:
- command:
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
  - name: PATRONI_POSTGRESQL_CONFIG_DIR
    value: /pgdata/pg11
  - name: PATRONI_POSTGRESQL_DATA_DIR
    value: /pgdata/pg11
  - name: PATRONI_RESTAPI_CONNECT_ADDRESS
    value: $(PATRONI_NAME).:8008
  - name: PATRONI_RESTAPI_LISTEN
    value: '*:8008'
  - name: PATRONICTL_CONFIG_FILE
    value: /etc/patroni
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
	`))
}

func TestPodIsPrimary(t *testing.T) {
	// No object
	assert.Assert(t, !PodIsPrimary(nil))

	// No annotations
	pod := &corev1.Pod{}
	assert.Assert(t, !PodIsPrimary(pod))

	// No role
	pod.Annotations = map[string]string{"status": `{}`}
	assert.Assert(t, !PodIsPrimary(pod))

	// Replica
	pod.Annotations["status"] = `{"role":"replica"}`
	assert.Assert(t, !PodIsPrimary(pod))

	// Standby leader
	pod.Annotations["status"] = `{"role":"standby_leader"}`
	assert.Assert(t, !PodIsPrimary(pod))

	// Primary
	pod.Annotations["status"] = `{"role":"master"}`
	assert.Assert(t, PodIsPrimary(pod))
}

func TestPodIsStandbyLeader(t *testing.T) {
	// No object
	assert.Assert(t, !PodIsStandbyLeader(nil))

	// No annotations
	pod := &corev1.Pod{}
	assert.Assert(t, !PodIsStandbyLeader(pod))

	// No role
	pod.Annotations = map[string]string{"status": `{}`}
	assert.Assert(t, !PodIsStandbyLeader(pod))

	// Leader
	pod.Annotations["status"] = `{"role":"master"}`
	assert.Assert(t, !PodIsStandbyLeader(pod))

	// Replica
	pod.Annotations["status"] = `{"role":"replica"}`
	assert.Assert(t, !PodIsStandbyLeader(pod))

	// Standby leader
	pod.Annotations["status"] = `{"role":"standby_leader"}`
	assert.Assert(t, PodIsStandbyLeader(pod))
}

func TestPodRequiresRestart(t *testing.T) {
	// No object
	assert.Assert(t, !PodRequiresRestart(nil))

	// No annotations
	pod := &corev1.Pod{}
	assert.Assert(t, !PodRequiresRestart(pod))

	// Normal; no flag
	pod.Annotations = map[string]string{"status": `{}`}
	assert.Assert(t, !PodRequiresRestart(pod))

	// Unexpected value
	pod.Annotations["status"] = `{"pending_restart":"mystery"}`
	assert.Assert(t, !PodRequiresRestart(pod))

	// Expected value
	pod.Annotations["status"] = `{"pending_restart":true}`
	assert.Assert(t, PodRequiresRestart(pod))
}
