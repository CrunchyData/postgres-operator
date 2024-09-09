// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbouncer

import (
	"context"
	"testing"

	gocmp "github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestConfigMap(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	config := new(corev1.ConfigMap)

	t.Run("Disabled", func(t *testing.T) {
		// Nothing happens when PgBouncer is disabled.
		constant := config.DeepCopy()
		ConfigMap(cluster, config)
		assert.DeepEqual(t, constant, config)
	})

	cluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
	cluster.Spec.Proxy.PGBouncer = new(v1beta1.PGBouncerPodSpec)
	cluster.Default()

	ConfigMap(cluster, config)

	// The output of clusterINI should go into config.
	data := clusterINI(cluster)
	assert.DeepEqual(t, config.Data["pgbouncer.ini"], data)

	// No change when called again.
	before := config.DeepCopy()
	ConfigMap(cluster, config)
	assert.DeepEqual(t, before, config)
}

func TestSecret(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := new(v1beta1.PostgresCluster)
	service := new(corev1.Service)
	existing := new(corev1.Secret)
	intent := new(corev1.Secret)

	root, err := pki.NewRootCertificateAuthority()
	assert.NilError(t, err)

	t.Run("Disabled", func(t *testing.T) {
		// Nothing happens when PgBouncer is disabled.
		constant := intent.DeepCopy()
		assert.NilError(t, Secret(ctx, cluster, root, existing, service, intent))
		assert.DeepEqual(t, constant, intent)
	})

	cluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
	cluster.Spec.Proxy.PGBouncer = new(v1beta1.PGBouncerPodSpec)
	cluster.Default()

	constant := existing.DeepCopy()
	assert.NilError(t, Secret(ctx, cluster, root, existing, service, intent))
	assert.DeepEqual(t, constant, existing)

	// A password should be generated.
	assert.Assert(t, len(intent.Data["pgbouncer-password"]) != 0)
	assert.Assert(t, len(intent.Data["pgbouncer-verifier"]) != 0)

	// The output of authFileContents should go into intent.
	assert.Assert(t, len(intent.Data["pgbouncer-users.txt"]) != 0)

	// Assuming the intent is written, no change when called again.
	existing.Data = intent.Data
	before := intent.DeepCopy()
	assert.NilError(t, Secret(ctx, cluster, root, existing, service, intent))
	assert.DeepEqual(t, before, intent)
}

func TestPod(t *testing.T) {
	t.Parallel()

	features := feature.NewGate()
	ctx := feature.NewContext(context.Background(), features)

	cluster := new(v1beta1.PostgresCluster)
	configMap := new(corev1.ConfigMap)
	primaryCertificate := new(corev1.SecretProjection)
	secret := new(corev1.Secret)
	pod := new(corev1.PodSpec)

	call := func() { Pod(ctx, cluster, configMap, primaryCertificate, secret, pod) }

	t.Run("Disabled", func(t *testing.T) {
		before := pod.DeepCopy()
		call()

		// No change when PgBouncer is not requested in the spec.
		assert.DeepEqual(t, before, pod)
	})

	t.Run("Defaults", func(t *testing.T) {
		cluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
		cluster.Spec.Proxy.PGBouncer = new(v1beta1.PGBouncerPodSpec)
		cluster.Default()

		call()

		assert.Assert(t, cmp.MarshalMatches(pod, `
containers:
- command:
  - pgbouncer
  - /etc/pgbouncer/~postgres-operator.ini
  name: pgbouncer
  ports:
  - containerPort: 5432
    name: pgbouncer
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
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgbouncer
    name: pgbouncer-config
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:||:)
    while read -r -t 5 -u "${fd}" ||:; do
      if [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] && pkill -HUP --exact pgbouncer
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded configuration dated %y' "${directory}"
      fi
    done
    }; export directory="$1"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgbouncer-config
  - /etc/pgbouncer
  name: pgbouncer-config
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgbouncer
    name: pgbouncer-config
    readOnly: true
volumes:
- name: pgbouncer-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbouncer-empty
          path: pgbouncer.ini
    - configMap:
        items:
        - key: pgbouncer.ini
          path: ~postgres-operator.ini
    - secret:
        items:
        - key: pgbouncer-users.txt
          path: ~postgres-operator/users.txt
    - secret:
        items:
        - key: pgbouncer-frontend.ca-roots
          path: ~postgres-operator/frontend-ca.crt
        - key: pgbouncer-frontend.key
          path: ~postgres-operator/frontend-tls.key
        - key: pgbouncer-frontend.crt
          path: ~postgres-operator/frontend-tls.crt
    - secret:
        items:
        - key: ca.crt
          path: ~postgres-operator/backend-ca.crt
		`))

		// No change when called again.
		before := pod.DeepCopy()
		call()
		assert.DeepEqual(t, before, pod)
	})

	t.Run("Customizations", func(t *testing.T) {
		cluster.Spec.ImagePullPolicy = corev1.PullAlways
		cluster.Spec.Proxy.PGBouncer.Image = "image-town"
		cluster.Spec.Proxy.PGBouncer.Resources.Requests = corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		}
		cluster.Spec.Proxy.PGBouncer.CustomTLSSecret = &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{Name: "tls-name"},
			Items: []corev1.KeyToPath{
				{Key: "k1", Path: "tls.crt"},
				{Key: "k2", Path: "tls.key"},
			},
		}

		call()

		assert.Assert(t, cmp.MarshalMatches(pod, `
containers:
- command:
  - pgbouncer
  - /etc/pgbouncer/~postgres-operator.ini
  image: image-town
  imagePullPolicy: Always
  name: pgbouncer
  ports:
  - containerPort: 5432
    name: pgbouncer
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
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgbouncer
    name: pgbouncer-config
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:||:)
    while read -r -t 5 -u "${fd}" ||:; do
      if [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] && pkill -HUP --exact pgbouncer
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded configuration dated %y' "${directory}"
      fi
    done
    }; export directory="$1"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgbouncer-config
  - /etc/pgbouncer
  image: image-town
  imagePullPolicy: Always
  name: pgbouncer-config
  resources:
    limits:
      cpu: 5m
      memory: 16Mi
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgbouncer
    name: pgbouncer-config
    readOnly: true
volumes:
- name: pgbouncer-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbouncer-empty
          path: pgbouncer.ini
    - configMap:
        items:
        - key: pgbouncer.ini
          path: ~postgres-operator.ini
    - secret:
        items:
        - key: pgbouncer-users.txt
          path: ~postgres-operator/users.txt
    - secret:
        items:
        - key: k1
          path: ~postgres-operator/frontend-tls.crt
        - key: k2
          path: ~postgres-operator/frontend-tls.key
        name: tls-name
    - secret:
        items:
        - key: ca.crt
          path: ~postgres-operator/backend-ca.crt
			`))
	})

	t.Run("Sidecar customization", func(t *testing.T) {
		cluster.Spec.Proxy.PGBouncer.Sidecars = &v1beta1.PGBouncerSidecars{
			PGBouncerConfig: &v1beta1.Sidecar{
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("200m"),
					},
				},
			},
		}

		call()

		assert.Assert(t, cmp.MarshalMatches(pod, `
containers:
- command:
  - pgbouncer
  - /etc/pgbouncer/~postgres-operator.ini
  image: image-town
  imagePullPolicy: Always
  name: pgbouncer
  ports:
  - containerPort: 5432
    name: pgbouncer
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
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgbouncer
    name: pgbouncer-config
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:||:)
    while read -r -t 5 -u "${fd}" ||:; do
      if [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] && pkill -HUP --exact pgbouncer
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded configuration dated %y' "${directory}"
      fi
    done
    }; export directory="$1"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgbouncer-config
  - /etc/pgbouncer
  image: image-town
  imagePullPolicy: Always
  name: pgbouncer-config
  resources:
    requests:
      cpu: 200m
  securityContext:
    allowPrivilegeEscalation: false
    capabilities:
      drop:
      - ALL
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
    seccompProfile:
      type: RuntimeDefault
  volumeMounts:
  - mountPath: /etc/pgbouncer
    name: pgbouncer-config
    readOnly: true
volumes:
- name: pgbouncer-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbouncer-empty
          path: pgbouncer.ini
    - configMap:
        items:
        - key: pgbouncer.ini
          path: ~postgres-operator.ini
    - secret:
        items:
        - key: pgbouncer-users.txt
          path: ~postgres-operator/users.txt
    - secret:
        items:
        - key: k1
          path: ~postgres-operator/frontend-tls.crt
        - key: k2
          path: ~postgres-operator/frontend-tls.key
        name: tls-name
    - secret:
        items:
        - key: ca.crt
          path: ~postgres-operator/backend-ca.crt
		`))
	})

	t.Run("WithCustomSidecarContainer", func(t *testing.T) {
		cluster.Spec.Proxy.PGBouncer.Containers = []corev1.Container{
			{Name: "customsidecar1"},
		}

		t.Run("SidecarNotEnabled", func(t *testing.T) {

			call()
			assert.Equal(t, len(pod.Containers), 2, "expected 2 containers in Pod, got %d", len(pod.Containers))
		})

		t.Run("SidecarEnabled", func(t *testing.T) {
			assert.NilError(t, features.SetFromMap(map[string]bool{
				feature.PGBouncerSidecars: true,
			}))
			call()

			assert.Equal(t, len(pod.Containers), 3, "expected 3 containers in Pod, got %d", len(pod.Containers))

			var found bool
			for i := range pod.Containers {
				if pod.Containers[i].Name == "customsidecar1" {
					found = true
					break
				}
			}
			assert.Assert(t, found, "expected custom sidecar 'customsidecar1', but container not found")
		})
	})
}

func TestPostgreSQL(t *testing.T) {
	t.Parallel()

	cluster := new(v1beta1.PostgresCluster)
	hbas := new(postgres.HBAs)

	t.Run("Disabled", func(t *testing.T) {
		PostgreSQL(cluster, hbas)

		// No change when PgBouncer is not requested in the spec.
		assert.DeepEqual(t, hbas, new(postgres.HBAs))
	})

	t.Run("Enabled", func(t *testing.T) {
		cluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
		cluster.Spec.Proxy.PGBouncer = new(v1beta1.PGBouncerPodSpec)
		cluster.Default()

		PostgreSQL(cluster, hbas)

		assert.DeepEqual(t, hbas,
			&postgres.HBAs{
				Mandatory: postgresqlHBAs(),
			},
			// postgres.HostBasedAuthentication has unexported fields. Call String() to compare.
			gocmp.Transformer("", postgres.HostBasedAuthentication.String))
	})
}
