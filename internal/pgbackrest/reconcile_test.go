// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp/cmpopts"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/pki"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestAddRepoVolumesToPod(t *testing.T) {

	postgresCluster := &v1beta1.PostgresCluster{ObjectMeta: metav1.ObjectMeta{Name: "hippo"}}

	testsCases := []struct {
		repos          []v1beta1.PGBackRestRepo
		containers     []corev1.Container
		initContainers []corev1.Container
		testMap        map[string]string
	}{{
		repos: []v1beta1.PGBackRestRepo{
			{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
			{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
		},
		initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
		containers:     []corev1.Container{{Name: "database"}, {Name: "pgbackrest"}},
		testMap:        map[string]string{},
	}, {
		repos: []v1beta1.PGBackRestRepo{
			{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
			{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
		},
		initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
		containers:     []corev1.Container{{Name: "database"}},
		testMap:        map[string]string{},
	}, {
		repos:          []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
		initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
		containers:     []corev1.Container{{Name: "database"}, {Name: "pgbackrest"}},
		testMap:        map[string]string{},
	}, {
		repos:          []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
		initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
		containers:     []corev1.Container{{Name: "database"}},
		testMap:        map[string]string{},
	}, {
		repos:          []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
		initContainers: []corev1.Container{},
		containers:     []corev1.Container{{Name: "database"}},
		testMap:        map[string]string{},
	},
		// rerun the same tests, but this time simulate an existing PVC
		{
			repos: []v1beta1.PGBackRestRepo{
				{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
				{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
			},
			initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
			containers:     []corev1.Container{{Name: "database"}, {Name: "pgbackrest"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos: []v1beta1.PGBackRestRepo{
				{Name: "repo1", Volume: &v1beta1.RepoPVC{}},
				{Name: "repo2", Volume: &v1beta1.RepoPVC{}},
			},
			initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
			containers:     []corev1.Container{{Name: "database"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos:          []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
			initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
			containers:     []corev1.Container{{Name: "database"}, {Name: "pgbackrest"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos:          []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
			initContainers: []corev1.Container{{Name: "pgbackrest-log-dir"}},
			containers:     []corev1.Container{{Name: "database"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}, {
			repos:          []v1beta1.PGBackRestRepo{{Name: "repo1", Volume: &v1beta1.RepoPVC{}}},
			initContainers: []corev1.Container{},
			containers:     []corev1.Container{{Name: "database"}},
			testMap: map[string]string{
				"repo1": "hippo-repo1",
			},
		}}

	for _, tc := range testsCases {
		t.Run(fmt.Sprintf("repos=%d, containers=%d", len(tc.repos), len(tc.containers)), func(t *testing.T) {
			postgresCluster.Spec.Backups.PGBackRest.Repos = tc.repos
			template := &corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					InitContainers: tc.initContainers,
					Containers:     tc.containers,
				},
			}
			err := AddRepoVolumesToPod(postgresCluster, template, tc.testMap, getContainerNames(tc.containers)...)
			if len(tc.initContainers) == 0 {
				assert.Error(t, err, "Unable to find init container \"pgbackrest-log-dir\" when adding pgBackRest repo volumes")
			} else {
				assert.NilError(t, err)

				// verify volumes and volume mounts
				for _, r := range tc.repos {
					var foundVolume bool
					for _, v := range template.Spec.Volumes {
						if v.Name == r.Name && v.VolumeSource.PersistentVolumeClaim.ClaimName ==
							naming.PGBackRestRepoVolume(postgresCluster, r.Name).Name {
							foundVolume = true
							break
						}
					}

					if !foundVolume {
						t.Errorf("volume %s is missing or invalid", r.Name)
					}

					for _, c := range template.Spec.Containers {
						var foundVolumeMount bool
						for _, vm := range c.VolumeMounts {
							if vm.Name == r.Name && vm.MountPath == "/pgbackrest/"+r.Name {
								foundVolumeMount = true
								break
							}
						}
						if !foundVolumeMount {
							t.Errorf("container volume mount %s is missing or invalid", r.Name)
						}
					}
					for _, c := range template.Spec.InitContainers {
						var foundVolumeMount bool
						for _, vm := range c.VolumeMounts {
							if vm.Name == r.Name && vm.MountPath == "/pgbackrest/"+r.Name {
								foundVolumeMount = true
								break
							}
						}
						if !foundVolumeMount {
							t.Errorf("init container volume mount %s is missing or invalid", r.Name)
						}
					}
				}
			}
		})
	}
}

func TestAddConfigToInstancePod(t *testing.T) {
	cluster := v1beta1.PostgresCluster{}
	cluster.Name = "hippo"
	cluster.Default()

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "database"},
			{Name: "other"},
			{Name: "pgbackrest"},
		},
	}

	alwaysExpect := func(t testing.TB, result *corev1.PodSpec) {
		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *result, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

		// Only database and pgBackRest containers have mounts.
		assert.Assert(t, cmp.MarshalMatches(result.Containers, `
- name: database
  resources: {}
  volumeMounts:
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
- name: other
  resources: {}
- name: pgbackrest
  resources: {}
  volumeMounts:
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
		`))
	}

	t.Run("CustomProjections", func(t *testing.T) {
		custom := corev1.ConfigMapProjection{}
		custom.Name = "custom-configmap"

		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Configuration = []corev1.VolumeProjection{
			{ConfigMap: &custom},
		}

		out := pod.DeepCopy()
		AddConfigToInstancePod(cluster, out)
		alwaysExpect(t, out)

		// Instance configuration files after custom projections.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        name: custom-configmap
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        - key: config-hash
          path: config-hash
        - key: pgbackrest-server.conf
          path: ~postgres-operator_server.conf
        name: hippo-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: hippo-pgbackrest
		`))
	})

	t.Run("NoVolumeRepo", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Repos = nil

		out := pod.DeepCopy()
		AddConfigToInstancePod(cluster, out)
		alwaysExpect(t, out)

		// Instance configuration and certificates.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        - key: config-hash
          path: config-hash
        - key: pgbackrest-server.conf
          path: ~postgres-operator_server.conf
        name: hippo-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: hippo-pgbackrest
		`))
	})

	t.Run("OneVolumeRepo", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{
			{
				Name:   "repo1",
				Volume: new(v1beta1.RepoPVC),
			},
		}

		out := pod.DeepCopy()
		AddConfigToInstancePod(cluster, out)
		alwaysExpect(t, out)

		// Instance configuration files, server config, and optional client certificates.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        - key: config-hash
          path: config-hash
        - key: pgbackrest-server.conf
          path: ~postgres-operator_server.conf
        name: hippo-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: hippo-pgbackrest
		`))
	})
}

func TestAddConfigToRepoPod(t *testing.T) {
	cluster := v1beta1.PostgresCluster{}
	cluster.Name = "hippo"
	cluster.Default()

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "other"},
			{Name: "pgbackrest"},
		},
	}

	alwaysExpect := func(t testing.TB, result *corev1.PodSpec) {
		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *result, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

		// Only pgBackRest containers have mounts.
		assert.Assert(t, cmp.MarshalMatches(result.Containers, `
- name: other
  resources: {}
- name: pgbackrest
  resources: {}
  volumeMounts:
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
		`))
	}

	t.Run("CustomProjections", func(t *testing.T) {
		custom := corev1.ConfigMapProjection{}
		custom.Name = "custom-configmap"

		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Configuration = []corev1.VolumeProjection{
			{ConfigMap: &custom},
		}

		out := pod.DeepCopy()
		AddConfigToRepoPod(cluster, out)
		alwaysExpect(t, out)

		// Repository configuration files, server config, and client certificates
		// after custom projections.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        name: custom-configmap
    - configMap:
        items:
        - key: pgbackrest_repo.conf
          path: pgbackrest_repo.conf
        - key: config-hash
          path: config-hash
        - key: pgbackrest-server.conf
          path: ~postgres-operator_server.conf
        name: hippo-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: hippo-pgbackrest
		`))
	})
}

func TestAddConfigToRestorePod(t *testing.T) {
	cluster := v1beta1.PostgresCluster{}
	cluster.Name = "source"
	cluster.Default()

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "other"},
			{Name: "pgbackrest"},
		},
	}

	alwaysExpect := func(t testing.TB, result *corev1.PodSpec) {
		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *result, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

		// Only pgBackRest containers have mounts.
		assert.Assert(t, cmp.MarshalMatches(result.Containers, `
- name: other
  resources: {}
- name: pgbackrest
  resources: {}
  volumeMounts:
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
		`))
	}

	t.Run("CustomProjections", func(t *testing.T) {
		custom := corev1.ConfigMapProjection{}
		custom.Name = "custom-configmap"

		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Configuration = []corev1.VolumeProjection{
			{ConfigMap: &custom},
		}

		custom2 := corev1.SecretProjection{}
		custom2.Name = "source-custom-secret"

		sourceCluster := cluster.DeepCopy()
		sourceCluster.Spec.Backups.PGBackRest.Configuration = []corev1.VolumeProjection{
			{Secret: &custom2},
		}

		out := pod.DeepCopy()
		AddConfigToRestorePod(cluster, sourceCluster, out)
		alwaysExpect(t, out)

		// Instance configuration files and optional client certificates
		// after custom projections.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        name: custom-configmap
    - secret:
        name: source-custom-secret
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        name: source-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: source-pgbackrest
        optional: true
		`))
	})

	t.Run("CloudBasedDataSourceProjections", func(t *testing.T) {
		custom := corev1.SecretProjection{}
		custom.Name = "custom-secret"

		cluster := cluster.DeepCopy()
		cluster.Spec.DataSource = &v1beta1.DataSource{
			PGBackRest: &v1beta1.PGBackRestDataSource{
				Configuration: []corev1.VolumeProjection{{Secret: &custom}},
			},
		}

		out := pod.DeepCopy()
		AddConfigToRestorePod(cluster, nil, out)
		alwaysExpect(t, out)

		// Instance configuration files and optional client certificates
		// after custom projections.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-config
  projected:
    sources:
    - secret:
        name: custom-secret
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        name: source-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: source-pgbackrest
        optional: true
		`))
	})

	t.Run("CustomFiles", func(t *testing.T) {
		custom := corev1.ConfigMapProjection{}
		custom.Name = "custom-configmap-files"

		cluster := cluster.DeepCopy()
		cluster.Spec.Config.Files = []corev1.VolumeProjection{
			{ConfigMap: &custom},
		}

		sourceCluster := cluster.DeepCopy()

		out := pod.DeepCopy()
		AddConfigToRestorePod(cluster, sourceCluster, out)
		alwaysExpect(t, out)

		// Instance configuration files and optional configuration files
		// after custom projections.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: postgres-config
  projected:
    sources:
    - configMap:
        name: custom-configmap-files
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        name: source-pgbackrest-config
    - secret:
        items:
        - key: pgbackrest.ca-roots
          path: ~postgres-operator/tls-ca.crt
        - key: pgbackrest-client.crt
          path: ~postgres-operator/client-tls.crt
        - key: pgbackrest-client.key
          mode: 384
          path: ~postgres-operator/client-tls.key
        name: source-pgbackrest
        optional: true
		`))
	})
}

func TestAddServerToInstancePod(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := v1beta1.PostgresCluster{}
	cluster.Name = "hippo"
	cluster.Default()

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "database"},
			{Name: "other"},
		},
		Volumes: []corev1.Volume{
			{Name: "other"},
			{Name: "postgres-data"},
			{Name: "postgres-wal"},
		},
	}

	t.Run("CustomResources", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Sidecars = &v1beta1.PGBackRestSidecars{
			PGBackRest: &v1beta1.Sidecar{
				Resources: &corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("5m"),
					},
				},
			},
			PGBackRestConfig: &v1beta1.Sidecar{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("17m"),
					},
				},
			},
		}

		out := pod.DeepCopy()
		AddServerToInstancePod(ctx, cluster, out, "instance-secret-name")

		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *out, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

		// The TLS server is added while other containers are untouched.
		// It has PostgreSQL volumes mounted while other volumes are ignored.
		assert.Assert(t, cmp.MarshalMatches(out.Containers, `
- name: database
  resources: {}
- name: other
  resources: {}
- command:
  - pgbackrest
  - server
  livenessProbe:
    exec:
      command:
      - pgbackrest
      - server-ping
  name: pgbackrest
  resources:
    requests:
      cpu: 5m
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
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
  - mountPath: /pgwal
    name: postgres-wal
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:||:)
    until read -r -t 5 -u "${fd}"; do
      if
        [[ "${filename}" -nt "/proc/self/fd/${fd}" ]] &&
        pkill -HUP --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --dereference --format='Loaded configuration dated %y' "${filename}"
      elif
        { [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] ||
          [[ "${authority}" -nt "/proc/self/fd/${fd}" ]]
        } &&
        pkill -HUP --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded certificates dated %y' "${directory}"
      fi
    done
    }; export directory="$1" authority="$2" filename="$3"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgbackrest-config
  - /etc/pgbackrest/server
  - /etc/pgbackrest/conf.d/~postgres-operator/tls-ca.crt
  - /etc/pgbackrest/conf.d/~postgres-operator_server.conf
  name: pgbackrest-config
  resources:
    limits:
      cpu: 17m
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
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
		`))

		// The server certificate comes from the instance Secret.
		// Other volumes are untouched.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: other
- name: postgres-data
- name: postgres-wal
- name: pgbackrest-server
  projected:
    sources:
    - secret:
        items:
        - key: pgbackrest-server.crt
          path: server-tls.crt
        - key: pgbackrest-server.key
          mode: 384
          path: server-tls.key
        name: instance-secret-name
		`))
	})

	t.Run("AddTablespaces", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.TablespaceVolumes: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		clusterWithTablespaces := cluster.DeepCopy()
		clusterWithTablespaces.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{
				TablespaceVolumes: []v1beta1.TablespaceVolume{
					{Name: "trial"},
					{Name: "castle"},
				},
			},
		}

		out := pod.DeepCopy()
		out.Volumes = append(out.Volumes, corev1.Volume{Name: "tablespace-trial"}, corev1.Volume{Name: "tablespace-castle"})
		AddServerToInstancePod(ctx, clusterWithTablespaces, out, "instance-secret-name")

		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *out, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))
		assert.Assert(t, cmp.MarshalMatches(out.Containers, `
- name: database
  resources: {}
- name: other
  resources: {}
- command:
  - pgbackrest
  - server
  livenessProbe:
    exec:
      command:
      - pgbackrest
      - server-ping
  name: pgbackrest
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
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
  - mountPath: /pgwal
    name: postgres-wal
  - mountPath: /tablespaces/trial
    name: tablespace-trial
  - mountPath: /tablespaces/castle
    name: tablespace-castle
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:||:)
    until read -r -t 5 -u "${fd}"; do
      if
        [[ "${filename}" -nt "/proc/self/fd/${fd}" ]] &&
        pkill -HUP --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --dereference --format='Loaded configuration dated %y' "${filename}"
      elif
        { [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] ||
          [[ "${authority}" -nt "/proc/self/fd/${fd}" ]]
        } &&
        pkill -HUP --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded certificates dated %y' "${directory}"
      fi
    done
    }; export directory="$1" authority="$2" filename="$3"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgbackrest-config
  - /etc/pgbackrest/server
  - /etc/pgbackrest/conf.d/~postgres-operator/tls-ca.crt
  - /etc/pgbackrest/conf.d/~postgres-operator_server.conf
  name: pgbackrest-config
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
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
		`))
	})
}

func TestAddServerToRepoPod(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := v1beta1.PostgresCluster{}
	cluster.Name = "hippo"
	cluster.Default()

	pod := corev1.PodSpec{
		Containers: []corev1.Container{
			{Name: "other"},
		},
	}

	t.Run("CustomResources", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.RepoHost = &v1beta1.PGBackRestRepoHost{
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceCPU: resource.MustParse("5m"),
				},
			},
		}
		cluster.Spec.Backups.PGBackRest.Sidecars = &v1beta1.PGBackRestSidecars{
			PGBackRestConfig: &v1beta1.Sidecar{
				Resources: &corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("19m"),
					},
				},
			},
		}

		out := pod.DeepCopy()
		AddServerToRepoPod(ctx, cluster, out)

		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *out, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

		// The TLS server is added while other containers are untouched.
		assert.Assert(t, cmp.MarshalMatches(out.Containers, `
- name: other
  resources: {}
- command:
  - pgbackrest
  - server
  livenessProbe:
    exec:
      command:
      - pgbackrest
      - server-ping
  name: pgbackrest
  resources:
    requests:
      cpu: 5m
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
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:||:)
    until read -r -t 5 -u "${fd}"; do
      if
        [[ "${filename}" -nt "/proc/self/fd/${fd}" ]] &&
        pkill -HUP --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --dereference --format='Loaded configuration dated %y' "${filename}"
      elif
        { [[ "${directory}" -nt "/proc/self/fd/${fd}" ]] ||
          [[ "${authority}" -nt "/proc/self/fd/${fd}" ]]
        } &&
        pkill -HUP --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:||:)
        stat --format='Loaded certificates dated %y' "${directory}"
      fi
    done
    }; export directory="$1" authority="$2" filename="$3"; export -f monitor; exec -a "$0" bash -ceu monitor
  - pgbackrest-config
  - /etc/pgbackrest/server
  - /etc/pgbackrest/conf.d/~postgres-operator/tls-ca.crt
  - /etc/pgbackrest/conf.d/~postgres-operator_server.conf
  name: pgbackrest-config
  resources:
    limits:
      cpu: 19m
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
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
		`))

		// The server certificate comes from the pgBackRest Secret.
		assert.Assert(t, cmp.MarshalMatches(out.Volumes, `
- name: pgbackrest-server
  projected:
    sources:
    - secret:
        items:
        - key: pgbackrest-repo-host.crt
          path: server-tls.crt
        - key: pgbackrest-repo-host.key
          mode: 384
          path: server-tls.key
        name: hippo-pgbackrest
		`))
	})
}

func getContainerNames(containers []corev1.Container) []string {
	names := make([]string, len(containers))
	for i, c := range containers {
		names[i] = c.Name
	}
	return names
}

func TestReplicaCreateCommand(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	instance := new(v1beta1.PostgresInstanceSetSpec)

	t.Run("NoRepositories", func(t *testing.T) {
		assert.Equal(t, 0, len(ReplicaCreateCommand(cluster, instance)))
	})

	t.Run("NoReadyRepositories", func(t *testing.T) {
		cluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
			Repos: []v1beta1.RepoStatus{{
				Name: "repo2", ReplicaCreateBackupComplete: false,
			}},
		}

		assert.Equal(t, 0, len(ReplicaCreateCommand(cluster, instance)))
	})

	t.Run("SomeReadyRepositories", func(t *testing.T) {
		cluster.Status.PGBackRest = &v1beta1.PGBackRestStatus{
			Repos: []v1beta1.RepoStatus{{
				Name: "repo2", ReplicaCreateBackupComplete: true,
			}, {
				Name: "repo3", ReplicaCreateBackupComplete: true,
			}},
		}

		assert.DeepEqual(t, ReplicaCreateCommand(cluster, instance), []string{
			"pgbackrest", "restore", "--delta", "--stanza=db", "--repo=2",
			"--link-map=pg_wal=/pgdata/pg0_wal", "--type=standby",
		})
	})

	t.Run("Standby", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Standby = &v1beta1.PostgresStandbySpec{
			Enabled:  true,
			RepoName: "repo7",
		}

		assert.DeepEqual(t, ReplicaCreateCommand(cluster, instance), []string{
			"pgbackrest", "restore", "--delta", "--stanza=db", "--repo=7",
			"--link-map=pg_wal=/pgdata/pg0_wal", "--type=standby",
		})
	})
}

func TestSecret(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	cluster := new(v1beta1.PostgresCluster)
	existing := new(corev1.Secret)
	intent := new(corev1.Secret)

	root, err := pki.NewRootCertificateAuthority()
	assert.NilError(t, err)

	t.Run("NoRepoHost", func(t *testing.T) {
		// Nothing happens when there is no repository host.
		constant := intent.DeepCopy()
		assert.NilError(t, Secret(ctx, cluster, nil, root, existing, intent))
		assert.DeepEqual(t, constant, intent)
	})

	host := new(appsv1.StatefulSet)
	host.Namespace = "ns1"
	host.Name = "some-repo"
	host.Spec.ServiceName = "some-domain"

	// The existing Secret does not change.
	constant := existing.DeepCopy()
	assert.NilError(t, Secret(ctx, cluster, host, root, existing, intent))
	assert.DeepEqual(t, constant, existing)

	// There is a leaf certificate and private key for the repository host.
	leaf := &pki.LeafCertificate{}
	assert.NilError(t, leaf.Certificate.UnmarshalText(intent.Data["pgbackrest-repo-host.crt"]))
	assert.NilError(t, leaf.PrivateKey.UnmarshalText(intent.Data["pgbackrest-repo-host.key"]))

	assert.DeepEqual(t, leaf.Certificate.DNSNames(), []string{
		leaf.Certificate.CommonName(),
		"some-repo-0.some-domain.ns1.svc",
		"some-repo-0.some-domain.ns1",
		"some-repo-0.some-domain",
	})

	// Assuming the intent is written, no change when called again.
	existing.Data = intent.Data
	before := intent.DeepCopy()
	assert.NilError(t, Secret(ctx, cluster, host, root, existing, intent))
	assert.DeepEqual(t, before, intent)

	t.Run("Rotation", func(t *testing.T) {
		// The leaf certificate is regenerated when the root authority changes.
		root2, err := pki.NewRootCertificateAuthority()
		assert.NilError(t, err)
		assert.NilError(t, Secret(ctx, cluster, host, root2, existing, intent))

		leaf2 := &pki.LeafCertificate{}
		assert.NilError(t, leaf2.Certificate.UnmarshalText(intent.Data["pgbackrest-repo-host.crt"]))
		assert.NilError(t, leaf2.PrivateKey.UnmarshalText(intent.Data["pgbackrest-repo-host.key"]))

		assert.Assert(t, !reflect.DeepEqual(leaf.Certificate, leaf2.Certificate))
		assert.Assert(t, !reflect.DeepEqual(leaf.PrivateKey, leaf2.PrivateKey))
	})
}
