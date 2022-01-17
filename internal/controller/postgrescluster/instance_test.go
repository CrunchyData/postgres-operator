//go:build envtest
// +build envtest

/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/wait"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestInstanceIsRunning(t *testing.T) {
	var instance Instance
	var known, running bool

	// No pods
	running, known = instance.IsRunning("any")
	assert.Assert(t, !known)
	assert.Assert(t, !running)

	// No statuses
	instance.Pods = []*corev1.Pod{{}}
	running, known = instance.IsRunning("any")
	assert.Assert(t, !known)
	assert.Assert(t, !running)

	// No states
	instance.Pods[0].Status.ContainerStatuses = []corev1.ContainerStatus{{
		Name: "c1",
	}}
	running, known = instance.IsRunning("c1")
	assert.Assert(t, known)
	assert.Assert(t, !running)

	running, known = instance.IsRunning("missing")
	assert.Assert(t, !known)
	assert.Assert(t, !running)

	// Running state
	// - https://releases.k8s.io/v1.21.0/staging/src/k8s.io/kubectl/pkg/cmd/debug/debug.go#L668
	instance.Pods[0].Status.ContainerStatuses[0].State.Running =
		new(corev1.ContainerStateRunning)

	running, known = instance.IsRunning("c1")
	assert.Assert(t, known)
	assert.Assert(t, running)

	running, known = instance.IsRunning("missing")
	assert.Assert(t, !known)
	assert.Assert(t, !running)

	// Init containers
	instance.Pods[0].Status.InitContainerStatuses = []corev1.ContainerStatus{{
		Name: "i1",
		State: corev1.ContainerState{
			Running: new(corev1.ContainerStateRunning),
		},
	}}

	running, known = instance.IsRunning("i1")
	assert.Assert(t, known)
	assert.Assert(t, running)
}

func TestInstanceIsWritable(t *testing.T) {
	var instance Instance
	var known, writable bool

	// No pods
	writable, known = instance.IsWritable()
	assert.Assert(t, !known)
	assert.Assert(t, !writable)

	// No annotations
	instance.Pods = []*corev1.Pod{{}}
	writable, known = instance.IsWritable()
	assert.Assert(t, !known)
	assert.Assert(t, !writable)

	// No role
	instance.Pods[0].Annotations = map[string]string{"status": `{}`}
	writable, known = instance.IsWritable()
	assert.Assert(t, !known)
	assert.Assert(t, !writable)

	// Patroni leader
	instance.Pods[0].Annotations["status"] = `{"role":"master"}`
	writable, known = instance.IsWritable()
	assert.Assert(t, known)
	assert.Assert(t, writable)

	// Patroni replica
	instance.Pods[0].Annotations["status"] = `{"role":"replica"}`
	writable, known = instance.IsWritable()
	assert.Assert(t, known)
	assert.Assert(t, !writable)

	// Patroni standby leader
	instance.Pods[0].Annotations["status"] = `{"role":"standby_leader"}`
	writable, known = instance.IsWritable()
	assert.Assert(t, known)
	assert.Assert(t, !writable)
}

func TestNewObservedInstances(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		observed := newObservedInstances(cluster, nil, nil)

		assert.Equal(t, len(observed.forCluster), 0)
		assert.Equal(t, len(observed.byName), 0)
		assert.Equal(t, len(observed.bySet), 0)
	})

	t.Run("PodMissingOthers", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		observed := newObservedInstances(
			cluster,
			nil,
			[]corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-pod-name",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "missing",
							"postgres-operator.crunchydata.com/instance":     "the-name",
						},
					},
				},
			})

		// Registers as an instance.
		assert.Equal(t, len(observed.forCluster), 1)
		assert.Equal(t, len(observed.byName), 1)
		assert.Equal(t, len(observed.bySet), 1)

		instance := observed.forCluster[0]
		assert.Equal(t, instance.Name, "the-name")
		assert.Equal(t, len(instance.Pods), 1)   // The Pod
		assert.Assert(t, instance.Runner == nil) // No matching StatefulSet
		assert.Assert(t, instance.Spec == nil)   // No matching PostgresInstanceSetSpec

		// Lookup based on its labels.
		assert.Equal(t, observed.byName["the-name"], instance)
		assert.DeepEqual(t, observed.bySet["missing"], []*Instance{instance})
		assert.DeepEqual(t, observed.setNames.List(), []string{"missing"})
	})

	t.Run("RunnerMissingOthers", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		observed := newObservedInstances(
			cluster,
			[]appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "the-name",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "missing",
						},
					},
				},
			},
			nil)

		// Registers as an instance.
		assert.Equal(t, len(observed.forCluster), 1)
		assert.Equal(t, len(observed.byName), 1)
		assert.Equal(t, len(observed.bySet), 1)

		instance := observed.forCluster[0]
		assert.Equal(t, instance.Name, "the-name")
		assert.Equal(t, len(instance.Pods), 0)   // No matching Pods
		assert.Assert(t, instance.Runner != nil) // The StatefulSet
		assert.Assert(t, instance.Spec == nil)   // No matching PostgresInstanceSetSpec

		// Lookup based on its name and labels.
		assert.Equal(t, observed.byName["the-name"], instance)
		assert.DeepEqual(t, observed.bySet["missing"], []*Instance{instance})
		assert.DeepEqual(t, observed.setNames.List(), []string{"missing"})
	})

	t.Run("Matching", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{Name: "00"}}

		observed := newObservedInstances(
			cluster,
			[]appsv1.StatefulSet{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "the-name",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "00",
						},
					},
				},
			},
			[]corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "some-pod-name",
						Labels: map[string]string{
							"postgres-operator.crunchydata.com/instance-set": "00",
							"postgres-operator.crunchydata.com/instance":     "the-name",
						},
					},
				},
			})

		// Registers as one instance.
		assert.Equal(t, len(observed.forCluster), 1)
		assert.Equal(t, len(observed.byName), 1)
		assert.Equal(t, len(observed.bySet), 1)

		instance := observed.forCluster[0]
		assert.Equal(t, instance.Name, "the-name")
		assert.Equal(t, len(instance.Pods), 1)   // The Pod
		assert.Assert(t, instance.Runner != nil) // The StatefulSet
		assert.Assert(t, instance.Spec != nil)   // The PostgresInstanceSetSpec

		// Lookup based on its name and labels.
		assert.Equal(t, observed.byName["the-name"], instance)
		assert.DeepEqual(t, observed.bySet["00"], []*Instance{instance})
		assert.DeepEqual(t, observed.setNames.List(), []string{"00"})
	})
}

func TestWritablePod(t *testing.T) {
	container := "container"

	t.Run("empty observed", func(t *testing.T) {
		observed := &observedInstances{}

		pod, instance := observed.writablePod("container")
		assert.Assert(t, pod == nil)
		assert.Assert(t, instance == nil)
	})
	t.Run("terminating", func(t *testing.T) {
		instances := []*Instance{
			{
				Name: "instance",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "namespace",
						Name:      "pod",
						Annotations: map[string]string{
							"status": `{"role":"master"}`,
						},
						DeletionTimestamp: &metav1.Time{},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name: container,
							State: corev1.ContainerState{
								Running: new(corev1.ContainerStateRunning),
							},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		}
		observed := &observedInstances{forCluster: instances}

		terminating, known := observed.forCluster[0].IsTerminating()
		assert.Assert(t, terminating && known)

		pod, instance := observed.writablePod("container")
		assert.Assert(t, pod == nil)
		assert.Assert(t, instance == nil)
	})
	t.Run("not running", func(t *testing.T) {
		instances := []*Instance{
			{
				Name: "instance",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "namespace",
						Name:      "pod",
						Annotations: map[string]string{
							"status": `{"role":"master"}`,
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name: container,
							State: corev1.ContainerState{
								Waiting: new(corev1.ContainerStateWaiting)},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		}
		observed := &observedInstances{forCluster: instances}

		running, known := observed.forCluster[0].IsRunning(container)
		assert.Check(t, !running && known)

		pod, instance := observed.writablePod("container")
		assert.Assert(t, pod == nil)
		assert.Assert(t, instance == nil)
	})
	t.Run("not writable", func(t *testing.T) {
		instances := []*Instance{
			{
				Name: "instance",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "namespace",
						Name:      "pod",
						Annotations: map[string]string{
							"status": `{"role":"replica"}`,
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name: container,
							State: corev1.ContainerState{
								Running: new(corev1.ContainerStateRunning),
							},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		}
		observed := &observedInstances{forCluster: instances}

		writable, known := observed.forCluster[0].IsWritable()
		assert.Check(t, !writable && known)

		pod, instance := observed.writablePod("container")
		assert.Assert(t, pod == nil)
		assert.Assert(t, instance == nil)
	})
	t.Run("writable instance exists", func(t *testing.T) {
		instances := []*Instance{
			{
				Name: "instance",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "namespace",
						Name:      "pod",
						Annotations: map[string]string{
							"status": `{"role":"master"}`,
						},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name: container,
							State: corev1.ContainerState{
								Running: new(corev1.ContainerStateRunning),
							},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		}
		observed := &observedInstances{forCluster: instances}

		terminating, known := observed.forCluster[0].IsTerminating()
		assert.Check(t, !terminating && known)
		writable, known := observed.forCluster[0].IsWritable()
		assert.Check(t, writable && known)
		running, known := observed.forCluster[0].IsRunning(container)
		assert.Check(t, running && known)

		pod, instance := observed.writablePod("container")
		assert.Assert(t, pod != nil)
		assert.Assert(t, instance != nil)
	})
}

func TestAddPGBackRestToInstancePodSpec(t *testing.T) {
	cluster := v1beta1.PostgresCluster{}
	cluster.Name = "hippo"
	cluster.Default()

	certificates := corev1.Secret{}
	certificates.Name = "some-secret"

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

	t.Run("NoVolumeRepo", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Repos = nil

		out := pod.DeepCopy()
		addPGBackRestToInstancePodSpec(cluster, &certificates, out)

		// Only Containers and Volumes fields have changed.
		assert.DeepEqual(t, pod, *out, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

		// Only database container has mounts.
		// Other containers are ignored.
		assert.Assert(t, marshalMatches(out.Containers, `
- name: database
  resources: {}
  volumeMounts:
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
- name: other
  resources: {}
		`))

		// Instance configuration files but no certificates.
		// Other volumes are ignored.
		assert.Assert(t, marshalMatches(out.Volumes, `
- name: other
- name: postgres-data
- name: postgres-wal
- name: pgbackrest-config
  projected:
    sources:
    - configMap:
        items:
        - key: pgbackrest_instance.conf
          path: pgbackrest_instance.conf
        - key: config-hash
          path: config-hash
        name: hippo-pgbackrest-config
		`))
	})

	t.Run("OneVolumeRepo", func(t *testing.T) {
		alwaysExpect := func(t testing.TB, result *corev1.PodSpec) {
			// Only Containers and Volumes fields have changed.
			assert.DeepEqual(t, pod, *result, cmpopts.IgnoreFields(pod, "Containers", "Volumes"))

			// Instance configuration files plus client and server certificates.
			// The server certificate comes from the instance Secret.
			// Other volumes are untouched.
			assert.Assert(t, marshalMatches(result.Volumes, `
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
        name: some-secret
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
        optional: true
			`))
		}

		cluster := cluster.DeepCopy()
		cluster.Spec.Backups.PGBackRest.Repos = []v1beta1.PGBackRestRepo{
			{
				Name:   "repo1",
				Volume: new(v1beta1.RepoPVC),
			},
		}

		out := pod.DeepCopy()
		addPGBackRestToInstancePodSpec(cluster, &certificates, out)
		alwaysExpect(t, out)

		// The TLS server is added and configuration mounted.
		// It has PostgreSQL volumes mounted while other volumes are ignored.
		assert.Assert(t, marshalMatches(out.Containers, `
- name: database
  resources: {}
  volumeMounts:
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
- name: other
  resources: {}
- command:
  - pgbackrest
  - server-start
  livenessProbe:
    exec:
      command:
      - pgbackrest
      - server-ping
  name: pgbackrest
  resources: {}
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
  - mountPath: /pgwal
    name: postgres-wal
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:)
    until read -r -t 5 -u "${fd}"; do
      if
        [ "${filename}" -nt "/proc/self/fd/${fd}" ] &&
        pkill --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:)
        stat --dereference --format='Loaded configuration dated %y' "${filename}"
      elif
        { [ "${directory}" -nt "/proc/self/fd/${fd}" ] ||
          [ "${authority}" -nt "/proc/self/fd/${fd}" ]
        } &&
        pkill --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:)
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
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
		`))

		t.Run("CustomResources", func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Backups.PGBackRest.Sidecars = &v1beta1.PGBackRestSidecars{
				PGBackRest: &v1beta1.Sidecar{
					Resources: &corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU: resource.MustParse("5m"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceMemory: resource.MustParse("9Mi"),
						},
					},
				},
			}

			before := out.DeepCopy()
			out := pod.DeepCopy()
			addPGBackRestToInstancePodSpec(cluster, &certificates, out)
			alwaysExpect(t, out)

			// Only the TLS server container changed.
			assert.Equal(t, len(before.Containers), len(out.Containers))
			assert.Assert(t, len(before.Containers) > 2)
			assert.DeepEqual(t, before.Containers[:2], out.Containers[:2])

			// It has the custom resources.
			assert.Assert(t, marshalMatches(out.Containers[2:], `
- command:
  - pgbackrest
  - server-start
  livenessProbe:
    exec:
      command:
      - pgbackrest
      - server-ping
  name: pgbackrest
  resources:
    limits:
      memory: 9Mi
    requests:
      cpu: 5m
  securityContext:
    allowPrivilegeEscalation: false
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
  - mountPath: /pgdata
    name: postgres-data
  - mountPath: /pgwal
    name: postgres-wal
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
- command:
  - bash
  - -ceu
  - --
  - |-
    monitor() {
    exec {fd}<> <(:)
    until read -r -t 5 -u "${fd}"; do
      if
        [ "${filename}" -nt "/proc/self/fd/${fd}" ] &&
        pkill --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:)
        stat --dereference --format='Loaded configuration dated %y' "${filename}"
      elif
        { [ "${directory}" -nt "/proc/self/fd/${fd}" ] ||
          [ "${authority}" -nt "/proc/self/fd/${fd}" ]
        } &&
        pkill --exact --parent=0 pgbackrest
      then
        exec {fd}>&- && exec {fd}<> <(:)
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
    privileged: false
    readOnlyRootFilesystem: true
    runAsNonRoot: true
  volumeMounts:
  - mountPath: /etc/pgbackrest/server
    name: pgbackrest-server
    readOnly: true
  - mountPath: /etc/pgbackrest/conf.d
    name: pgbackrest-config
    readOnly: true
			`))
		})
	})

}

func TestPodsToKeep(t *testing.T) {
	for _, test := range []struct {
		name      string
		instances []corev1.Pod
		want      map[string]int
		checks    func(*testing.T, []corev1.Pod)
	}{
		{
			name: "RemoveSetWithMasterOnly",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 0)
			},
		}, {
			name: "RemoveSetWithReplicaOnly",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 0)
			},
		}, {
			name: "KeepMasterOnly",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"daisy": 1,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 1)
			},
		}, {
			name: "KeepNoRoleLabels",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"daisy": 1,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 1)
			},
		}, {
			name: "RemoveSetWithNoRoleLabels",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 0)
			},
		}, {
			name: "KeepUnknownRoleLabel",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "unknownLabelRole",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"daisy": 1,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 1)
			},
		}, {
			name: "RemoveSetWithUnknownRoleLabel",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "unknownLabelRole",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 0)
			},
		}, {
			name: "MasterLastInSet",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-poih",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"daisy": 1,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 1)
				assert.Equal(t, p[0].Labels[naming.LabelRole], "master")
			},
		}, {
			name: "ScaleDownSetWithMaster",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "max",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-poih",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"max":   1,
				"daisy": 1,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 2)
				assert.Equal(t, p[0].Labels[naming.LabelRole], "master")
				assert.Equal(t, p[0].Labels[naming.LabelInstanceSet], "daisy")
				assert.Equal(t, p[1].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[1].Labels[naming.LabelInstanceSet], "max")
			},
		}, {
			name: "ScaleDownSetWithoutMaster",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "max",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-poih",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"max":   1,
				"daisy": 2,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 3)
				assert.Equal(t, p[0].Labels[naming.LabelRole], "master")
				assert.Equal(t, p[0].Labels[naming.LabelInstanceSet], "max")
				assert.Equal(t, p[1].Labels[naming.LabelInstanceSet], "daisy")
				assert.Equal(t, p[1].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[2].Labels[naming.LabelInstanceSet], "daisy")
				assert.Equal(t, p[2].Labels[naming.LabelRole], "replica")
			},
		}, {
			name: "ScaleMasterSetToZero",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "max",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-poih",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"max":   0,
				"daisy": 2,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 2)
				assert.Equal(t, p[0].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[0].Labels[naming.LabelInstanceSet], "daisy")
				assert.Equal(t, p[1].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[1].Labels[naming.LabelInstanceSet], "daisy")
			},
		}, {
			name: "RemoveMasterInstanceSet",
			instances: []corev1.Pod{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-asdf",
						Labels: map[string]string{
							naming.LabelRole:        "master",
							naming.LabelInstanceSet: "max",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-poih",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "max-dogs",
						Labels: map[string]string{
							naming.LabelRole:        "replica",
							naming.LabelInstanceSet: "daisy",
						},
					},
				},
			},
			want: map[string]int{
				"daisy": 3,
			},
			checks: func(t *testing.T, p []corev1.Pod) {
				assert.Equal(t, len(p), 3)
				assert.Equal(t, p[0].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[0].Labels[naming.LabelInstanceSet], "daisy")
				assert.Equal(t, p[1].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[1].Labels[naming.LabelInstanceSet], "daisy")
				assert.Equal(t, p[2].Labels[naming.LabelRole], "replica")
				assert.Equal(t, p[2].Labels[naming.LabelInstanceSet], "daisy")
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			keep := podsToKeep(test.instances, test.want)
			sort.Slice(keep, func(i, j int) bool {
				return keep[i].Labels[naming.LabelRole] == "master"
			})
			test.checks(t, keep)
		})
	}
}

func TestDeleteInstance(t *testing.T) {
	env, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &Reconciler{}
	ctx, cancel := setupManager(t, env.Config, func(mgr manager.Manager) {
		reconciler = &Reconciler{
			Client:   cc,
			Owner:    client.FieldOwner(t.Name()),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(t.Name()),
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	// Define, Create, and Reconcile a cluster to get an instance running in kube
	cluster := testCluster()
	cluster.Namespace = setupNamespace(t, cc).Name

	assert.NilError(t, errors.WithStack(reconciler.Client.Create(ctx, cluster)))
	t.Cleanup(func() {
		// Remove finalizers, if any, so the namespace can terminate.
		assert.Check(t, client.IgnoreNotFound(
			reconciler.Client.Patch(ctx, cluster, client.RawPatch(
				client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
	})

	// Reconcile the entire cluster so that we don't have to create all the
	// resources needed to reconcile a single instance (cm,secrets,svc, etc.)
	result, err := reconciler.Reconcile(ctx, reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(cluster),
	})
	assert.NilError(t, err)
	assert.Assert(t, result.Requeue == false)

	stsList := &appsv1.StatefulSetList{}
	assert.NilError(t, reconciler.Client.List(ctx, stsList,
		client.InNamespace(cluster.Namespace),
		client.MatchingLabels{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: cluster.Spec.InstanceSets[0].Name,
		}))

	// Grab the instance name off of the instance set at index0
	instanceName := stsList.Items[0].Labels[naming.LabelInstance]

	// Use the instance name to delete the single instance
	assert.NilError(t, reconciler.deleteInstance(ctx, cluster, instanceName))

	gvks := []schema.GroupVersionKind{
		corev1.SchemeGroupVersion.WithKind("PersistentVolumeClaim"),
		corev1.SchemeGroupVersion.WithKind("ConfigMap"),
		corev1.SchemeGroupVersion.WithKind("Secret"),
		appsv1.SchemeGroupVersion.WithKind("StatefulSet"),
	}

	selector, err := naming.AsSelector(metav1.LabelSelector{
		MatchLabels: map[string]string{
			naming.LabelCluster:  cluster.Name,
			naming.LabelInstance: instanceName,
		}})
	assert.NilError(t, err)

	for _, gvk := range gvks {
		t.Run(gvk.Kind, func(t *testing.T) {
			uList := &unstructured.UnstructuredList{}
			err := wait.Poll(time.Second*3, Scale(time.Second*30), func() (bool, error) {
				uList.SetGroupVersionKind(gvk)
				assert.NilError(t, errors.WithStack(reconciler.Client.List(ctx, uList,
					client.InNamespace(cluster.Namespace),
					client.MatchingLabelsSelector{Selector: selector})))

				if len(uList.Items) == 0 {
					return true, nil
				}

				// Check existing objects for deletionTimestamp ensuring they
				// are staged for delete
				deleted := true
				for i := range uList.Items {
					u := uList.Items[i]
					if u.GetDeletionTimestamp() == nil {
						deleted = false
					}
				}

				// We have found objects that are not staged for delete
				// so deleteInstance has failed
				return deleted, nil
			})
			assert.NilError(t, err)
		})
	}
}

func TestGenerateInstanceStatefulSetIntent(t *testing.T) {
	type intentParams struct {
		cluster                    *v1beta1.PostgresCluster
		spec                       *v1beta1.PostgresInstanceSetSpec
		clusterPodServiceName      string
		instanceServiceAccountName string
		sts                        *appsv1.StatefulSet
		shutdown                   bool
		startupInstance            string
		numInstancePods            int
	}

	for _, test := range []struct {
		name string
		ip   intentParams
		run  func(*testing.T, *appsv1.StatefulSet)
	}{{
		name: "cluster pod service name",
		ip: intentParams{
			clusterPodServiceName: "daisy-svc",
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, ss.Spec.ServiceName, "daisy-svc")
		},
	}, {
		name: "instance service account name",
		ip: intentParams{
			instanceServiceAccountName: "daisy-sa",
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, ss.Spec.Template.Spec.ServiceAccountName, "daisy-sa")
		},
	}, {
		name: "custom affinity",
		ip: intentParams{
			spec: &v1beta1.PostgresInstanceSetSpec{
				Affinity: &corev1.Affinity{},
			},
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Assert(t, ss.Spec.Template.Spec.Affinity != nil)
		},
	}, {
		name: "custom tolerations",
		ip: intentParams{
			spec: &v1beta1.PostgresInstanceSetSpec{
				Tolerations: []corev1.Toleration{},
			},
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Assert(t, ss.Spec.Template.Spec.Tolerations != nil)
		},
	}, {
		name: "custom topology spread constraints",
		ip: intentParams{
			spec: &v1beta1.PostgresInstanceSetSpec{
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{},
			},
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Assert(t, ss.Spec.Template.Spec.TopologySpreadConstraints != nil)
		},
	}, {
		name: "shutdown replica",
		ip: intentParams{
			shutdown:        true,
			numInstancePods: 2,
			startupInstance: "testInstance1",
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, *ss.Spec.Replicas, int32(0))
		},
	}, {
		name: "shutdown primary",
		ip: intentParams{
			shutdown:        true,
			numInstancePods: 1,
			startupInstance: "testInstance1",
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, *ss.Spec.Replicas, int32(0))
		},
	}, {
		name: "startup primary",
		ip: intentParams{
			shutdown:        false,
			numInstancePods: 0,
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, *ss.Spec.Replicas, int32(1))
		},
	}, {
		name: "startup replica",
		ip: intentParams{
			shutdown:        false,
			numInstancePods: 1,
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, *ss.Spec.Replicas, int32(1))
		},
	}, {
		name: "do not startup replica",
		ip: intentParams{
			shutdown:        false,
			numInstancePods: 0,
			startupInstance: "testInstance1",
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, *ss.Spec.Replicas, int32(0))
		},
	}, {
		name: "do not shutdown primary",
		ip: intentParams{
			shutdown:        true,
			numInstancePods: 2,
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, *ss.Spec.Replicas, int32(1))
		},
	}, {
		name: "check imagepullsecret",
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Assert(t, ss.Spec.Template.Spec.ImagePullSecrets != nil)
			assert.Equal(t, ss.Spec.Template.Spec.ImagePullSecrets[0].Name,
				"myImagePullSecret")
		},
	}, {
		name: "check pod priority",
		ip: intentParams{
			spec: &v1beta1.PostgresInstanceSetSpec{
				PriorityClassName: initialize.String("some-priority-class"),
			},
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, ss.Spec.Template.Spec.PriorityClassName,
				"some-priority-class")
		},
	}, {
		name: "check default scheduling constraints are added",
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, len(ss.Spec.Template.Spec.TopologySpreadConstraints), 2)
			assert.Assert(t, marshalMatches(ss.Spec.Template.Spec.TopologySpreadConstraints, `
- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/data
      operator: In
      values:
      - postgres
      - pgbackrest
    matchLabels:
      postgres-operator.crunchydata.com/cluster: hippo
  maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/data
      operator: In
      values:
      - postgres
      - pgbackrest
    matchLabels:
      postgres-operator.crunchydata.com/cluster: hippo
  maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
			`))
		},
	}, {
		name: "check default scheduling constraints are appended to existing",
		ip: intentParams{
			spec: &v1beta1.PostgresInstanceSetSpec{
				Name: "instance1",
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           int32(1),
					TopologyKey:       "kubernetes.io/hostname",
					WhenUnsatisfiable: corev1.ScheduleAnyway,
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: naming.LabelCluster, Operator: "In", Values: []string{"somename"}},
							{Key: naming.LabelData, Operator: "Exists"},
						},
					},
				}},
			},
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, len(ss.Spec.Template.Spec.TopologySpreadConstraints), 3)
			assert.Assert(t, marshalMatches(ss.Spec.Template.Spec.TopologySpreadConstraints, `
- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/cluster
      operator: In
      values:
      - somename
    - key: postgres-operator.crunchydata.com/data
      operator: Exists
  maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/data
      operator: In
      values:
      - postgres
      - pgbackrest
    matchLabels:
      postgres-operator.crunchydata.com/cluster: hippo
  maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/data
      operator: In
      values:
      - postgres
      - pgbackrest
    matchLabels:
      postgres-operator.crunchydata.com/cluster: hippo
  maxSkew: 1
  topologyKey: topology.kubernetes.io/zone
  whenUnsatisfiable: ScheduleAnyway
			`))
		},
	}, {
		name: "check defined constraint when defaults disabled",
		ip: intentParams{
			cluster: &v1beta1.PostgresCluster{
				ObjectMeta: metav1.ObjectMeta{
					Name: "hippo",
				},
				Spec: v1beta1.PostgresClusterSpec{
					PostgresVersion:             13,
					DisableDefaultPodScheduling: initialize.Bool(true),
					InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
						Name:                "instance1",
						Replicas:            Int32(1),
						DataVolumeClaimSpec: testVolumeClaimSpec(),
						TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
							MaxSkew:           int32(1),
							TopologyKey:       "kubernetes.io/hostname",
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchExpressions: []metav1.LabelSelectorRequirement{
									{Key: naming.LabelCluster, Operator: "In", Values: []string{"somename"}},
									{Key: naming.LabelData, Operator: "Exists"},
								},
							},
						}},
					}},
				},
			},
			spec: &v1beta1.PostgresInstanceSetSpec{
				Name: "instance1",
				TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
					MaxSkew:           int32(1),
					TopologyKey:       "kubernetes.io/hostname",
					WhenUnsatisfiable: corev1.ScheduleAnyway,
					LabelSelector: &metav1.LabelSelector{
						MatchExpressions: []metav1.LabelSelectorRequirement{
							{Key: naming.LabelCluster, Operator: "In", Values: []string{"somename"}},
							{Key: naming.LabelData, Operator: "Exists"},
						},
					},
				}},
			},
		},
		run: func(t *testing.T, ss *appsv1.StatefulSet) {
			assert.Equal(t, len(ss.Spec.Template.Spec.TopologySpreadConstraints), 1)
			assert.Assert(t, marshalMatches(ss.Spec.Template.Spec.TopologySpreadConstraints,
				`- labelSelector:
    matchExpressions:
    - key: postgres-operator.crunchydata.com/cluster
      operator: In
      values:
      - somename
    - key: postgres-operator.crunchydata.com/data
      operator: Exists
  maxSkew: 1
  topologyKey: kubernetes.io/hostname
  whenUnsatisfiable: ScheduleAnyway
`))
		},
	}} {
		t.Run(test.name, func(t *testing.T) {

			cluster := test.ip.cluster
			if cluster == nil {
				cluster = testCluster()
			}

			cluster.Default()
			cluster.UID = types.UID("hippouid")
			cluster.Namespace = test.name + "-ns"
			cluster.Spec.Shutdown = &test.ip.shutdown
			cluster.Status.StartupInstance = test.ip.startupInstance

			spec := test.ip.spec
			if spec == nil {
				spec = new(v1beta1.PostgresInstanceSetSpec)
				spec.Default(0)
			}

			clusterPodServiceName := test.ip.clusterPodServiceName
			instanceServiceAccountName := test.ip.instanceServiceAccountName
			sts := test.ip.sts
			if sts == nil {
				sts = &appsv1.StatefulSet{}
			}

			generateInstanceStatefulSetIntent(context.Background(),
				cluster, spec,
				clusterPodServiceName,
				instanceServiceAccountName,
				sts,
				test.ip.numInstancePods,
			)

			test.run(t, sts)

			if assert.Check(t, sts.Spec.Template.Spec.EnableServiceLinks != nil) {
				assert.Equal(t, *sts.Spec.Template.Spec.EnableServiceLinks, false)
			}
		})
	}
}

func TestFindAvailableInstanceNames(t *testing.T) {

	testCases := []struct {
		set                   v1beta1.PostgresInstanceSetSpec
		fakeObservedInstances *observedInstances
		fakeClusterVolumes    []corev1.PersistentVolumeClaim
		expectedInstanceNames []string
	}{{
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1"},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{}},
			}},
			[]appsv1.StatefulSet{{}},
			[]corev1.Pod{},
		),
		fakeClusterVolumes:    []corev1.PersistentVolumeClaim{{}},
		expectedInstanceNames: []string{},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1"},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc",
				Labels: map[string]string{
					naming.LabelInstanceSet: "instance1"}}}},
			[]corev1.Pod{},
		),
		fakeClusterVolumes: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{
			Name: "instance1-abc-def",
			Labels: map[string]string{
				naming.LabelRole:        naming.RolePostgresData,
				naming.LabelInstanceSet: "instance1",
				naming.LabelInstance:    "instance1-abc"}}}},
		expectedInstanceNames: []string{},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1"},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc",
				Labels: map[string]string{
					naming.LabelInstanceSet: "instance1"}}}},
			[]corev1.Pod{},
		),
		fakeClusterVolumes:    []corev1.PersistentVolumeClaim{},
		expectedInstanceNames: []string{},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1"},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc",
				Labels: map[string]string{
					naming.LabelInstanceSet: "instance1"}}}},
			[]corev1.Pod{},
		),
		fakeClusterVolumes: []corev1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc-def",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresData,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-abc"}}},
			{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc-efg",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresData,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-def"}}},
		},
		expectedInstanceNames: []string{"instance1-def"},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1"},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc",
				Labels: map[string]string{
					naming.LabelInstanceSet: "instance1"}}}},
			[]corev1.Pod{},
		),
		fakeClusterVolumes: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{
			Name: "instance1-abc-def",
			Labels: map[string]string{
				naming.LabelRole:        naming.RolePostgresData,
				naming.LabelInstanceSet: "instance1",
				naming.LabelInstance:    "instance1-def"}}}},
		expectedInstanceNames: []string{"instance1-def"},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1",
			WALVolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{}},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc",
				Labels: map[string]string{
					naming.LabelInstanceSet: "instance1"}}}},
			[]corev1.Pod{},
		),
		fakeClusterVolumes: []corev1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc-def",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresData,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-abc"}}},
			{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-abc-def",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresWAL,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-abc"}}}},
		expectedInstanceNames: []string{},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1",
			WALVolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{}},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{},
			[]corev1.Pod{},
		),
		fakeClusterVolumes: []corev1.PersistentVolumeClaim{
			{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-def-ghi",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresData,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-def"}}},
			{ObjectMeta: metav1.ObjectMeta{
				Name: "instance1-def-jkl",
				Labels: map[string]string{
					naming.LabelRole:        naming.RolePostgresWAL,
					naming.LabelInstanceSet: "instance1",
					naming.LabelInstance:    "instance1-def"}}}},
		expectedInstanceNames: []string{"instance1-def"},
	}, {
		set: v1beta1.PostgresInstanceSetSpec{Name: "instance1",
			WALVolumeClaimSpec: &corev1.PersistentVolumeClaimSpec{}},
		fakeObservedInstances: newObservedInstances(
			&v1beta1.PostgresCluster{Spec: v1beta1.PostgresClusterSpec{
				InstanceSets: []v1beta1.PostgresInstanceSetSpec{{Name: "instance1"}},
			}},
			[]appsv1.StatefulSet{},
			[]corev1.Pod{},
		),
		fakeClusterVolumes: []corev1.PersistentVolumeClaim{{ObjectMeta: metav1.ObjectMeta{
			Name: "instance1-def-ghi",
			Labels: map[string]string{
				naming.LabelRole:        naming.RolePostgresData,
				naming.LabelInstanceSet: "instance1",
				naming.LabelInstance:    "instance1-def"}}}},
		expectedInstanceNames: []string{},
	}}

	for _, tc := range testCases {
		var walEnabled string
		if tc.set.WALVolumeClaimSpec != nil {
			walEnabled = ", WAL volume enabled"
		}
		name := fmt.Sprintf("%d set(s), %d volume(s)%s: expect %d instance names(s)",
			len(tc.fakeObservedInstances.setNames), len(tc.fakeClusterVolumes), walEnabled,
			len(tc.expectedInstanceNames))
		t.Run(name, func(t *testing.T) {
			assert.DeepEqual(t, findAvailableInstanceNames(tc.set, tc.fakeObservedInstances,
				tc.fakeClusterVolumes), tc.expectedInstanceNames)
		})
	}
}

func TestReconcileUpgrade(t *testing.T) {
	tEnv, tClient := setupKubernetes(t)

	// TODO(cbandy): Assume this should run alone for now.
	require.ParallelCapacity(t, 99)

	r := &Reconciler{}
	ctx, cancel := setupManager(t, tEnv.Config, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	ns := setupNamespace(t, tClient)
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "hippo-sa"},
	}

	generateJob := func(clusterName, pgVersion string, completed, failed *bool) {

		cluster := &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: ns.GetName(),
			},
		}
		meta := naming.PGUpgradeJob(cluster)
		labels := naming.PGUpgradeJobLabels(cluster.Name)
		meta.Labels = labels
		meta.Annotations = map[string]string{
			naming.PGBackRestConfigHash: "testhash",
			naming.PGUpgradeVersion:     pgVersion,
		}

		upgradeJob := &batchv1.Job{
			ObjectMeta: meta,
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: meta,
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "test",
							Name:  naming.PGBackRestRestoreContainerName,
						}},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}

		var updateStatus bool
		if completed != nil {
			if *completed {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionTrue,
					Reason:  "test",
					Message: "test",
				})
			} else {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionFalse,
					Reason:  "test",
					Message: "test",
				})
			}
			updateStatus = true
		} else if failed != nil {
			if *failed {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "test",
					Message: "test",
				})
			} else {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionFalse,
					Reason:  "test",
					Message: "test",
				})
			}
			updateStatus = true
		}

		assert.NilError(t, r.Client.Create(ctx, upgradeJob.DeepCopy()))
		if updateStatus {
			assert.NilError(t, r.Client.Status().Update(ctx, upgradeJob))
		}
	}

	obs := &observedInstances{}

	testCases := []struct {
		// a description of the test
		testDesc string
		// function that creates resources for the test
		createResources func(t *testing.T, clusterName string)
		// conditions to apply to the mock postgrescluster
		clusterConditions []*metav1.Condition
		// the status to apply to the mock postgrescluster
		status *v1beta1.PostgresClusterStatus
		// the upgrade field to define in the postgrescluster spec for the test
		upgrade *v1beta1.PGMajorUpgrade
		// whether or not the test should expect a Job to be reconciled
		expectReconcile bool
		// expected return value
		expectedReturnEarly bool
	}{{
		testDesc: "upgrade not enabled",
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(false),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		status:              &v1beta1.PostgresClusterStatus{},
		expectReconcile:     false,
		expectedReturnEarly: false,
	}, {
		testDesc: "upgrade enabled, no upgrade job, completed condition not set",
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		status: &v1beta1.PostgresClusterStatus{
			StartupInstance:    "instance1-abcd",
			StartupInstanceSet: "instance1",
		},
		expectReconcile:     false,
		expectedReturnEarly: true,
	}, {
		testDesc: "upgrade job completed",
		createResources: func(t *testing.T, clusterName string) {
			if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires mocking of Job conditions")
			}
			generateJob(clusterName, "13", initialize.Bool(true), nil)
		},
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		status: &v1beta1.PostgresClusterStatus{
			StartupInstance:    "instance1-abcd",
			StartupInstanceSet: "instance1",
		},
		expectReconcile:     false,
		expectedReturnEarly: false,
	}, {
		testDesc: "upgrade job failed",
		createResources: func(t *testing.T, clusterName string) {
			if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires mocking of Job conditions")
			}
			generateJob(clusterName, "13", nil, initialize.Bool(true))
		},
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		status: &v1beta1.PostgresClusterStatus{
			StartupInstance:    "instance1-abcd",
			StartupInstanceSet: "instance1",
		},
		expectReconcile:     false,
		expectedReturnEarly: true,
	}, {
		testDesc: "invalid from PG version",
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 14,
			Image:               initialize.String("upgrade-image"),
		},
		clusterConditions: []*metav1.Condition{{
			Type:    ConditionPGUpgradeCompleted,
			Status:  metav1.ConditionTrue,
			Reason:  "test",
			Message: "",
		}},
		expectReconcile:     false,
		expectedReturnEarly: true,
	}, {
		testDesc: "upgrade progressing, not ready for pg_upgrade job",
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		status: &v1beta1.PostgresClusterStatus{
			StartupInstance:    "instance1-abcd",
			StartupInstanceSet: "instance1",
		},
		clusterConditions: []*metav1.Condition{{
			Type:    ConditionPGUpgradeProgressing,
			Reason:  "test",
			Status:  metav1.ConditionTrue,
			Message: "",
		}},
		expectReconcile:     false,
		expectedReturnEarly: true,
	}, {
		testDesc: "upgrade progressing, ready for pg_upgrade job",
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		status: &v1beta1.PostgresClusterStatus{
			StartupInstance:    "instance1-abcd",
			StartupInstanceSet: "instance1",
		},
		clusterConditions: []*metav1.Condition{{
			Type:    ConditionPGUpgradeProgressing,
			Reason:  ReasonReadyForUpgrade,
			Status:  metav1.ConditionTrue,
			Message: "test",
		}},
		expectReconcile:     true,
		expectedReturnEarly: true,
	}, {
		testDesc: "upgrade progressing, shutdown",
		upgrade: &v1beta1.PGMajorUpgrade{
			Enabled:             initialize.Bool(true),
			FromPostgresVersion: 12,
			Image:               initialize.String("upgrade-image"),
		},
		clusterConditions: []*metav1.Condition{{
			Type:    ConditionRepoHostReady,
			Reason:  "test",
			Status:  metav1.ConditionTrue,
			Message: "test",
		}},
		expectReconcile:     false,
		expectedReturnEarly: false,
	}}

	for i, tc := range testCases {
		clusterName := "pg-upgrade-" + strconv.Itoa(i)

		t.Run(tc.testDesc, func(t *testing.T) {

			if tc.createResources != nil {
				tc.createResources(t, clusterName)
			}

			ctx := context.Background()
			cluster := fakeUpgradeCluster(clusterName, ns.GetName(), "")
			cluster.Spec.Upgrade = tc.upgrade
			if tc.status != nil {
				cluster.Status = *tc.status
			}
			for i := range tc.clusterConditions {
				meta.SetStatusCondition(&cluster.Status.Conditions, *tc.clusterConditions[i])
			}
			assert.NilError(t, tClient.Create(ctx, cluster))
			t.Cleanup(func() {
				// Remove finalizers, if any, so the namespace can terminate.
				assert.Check(t, client.IgnoreNotFound(
					tClient.Patch(ctx, cluster, client.RawPatch(
						client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`)))))
			})
			assert.NilError(t, tClient.Status().Update(ctx, cluster))

			// resources needed for reconcileUpgradeJob
			spec := []v1beta1.PostgresInstanceSetSpec{{
				Name:                "instance1",
				Replicas:            Int32(1),
				DataVolumeClaimSpec: testVolumeClaimSpec(),
			}}
			clusterCerts := &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "cluster-certs",
				},
			}
			clientCerts := &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "client-certs",
				},
			}
			volumes := []corev1.PersistentVolumeClaim{{}}

			returnEarly, err := r.reconcileUpgradeJob(ctx, cluster, obs, spec, sa.Name, clusterCerts, clientCerts, volumes)
			assert.NilError(t, err)
			assert.Equal(t, returnEarly, tc.expectedReturnEarly)

			if tc.expectReconcile {
				// verify expected behavior when a reconcile is expected
				jobs := &batchv1.JobList{}
				err := tClient.List(ctx, jobs, &client.ListOptions{
					LabelSelector: naming.PGUpgradeJobSelector(clusterName),
				})
				assert.NilError(t, err)
				runningJobs := []*batchv1.Job{}
				for i := range jobs.Items {
					if !jobFailed(&jobs.Items[i]) && !jobCompleted(&jobs.Items[i]) {
						runningJobs = append(runningJobs, &jobs.Items[i])
					}
				}
				assert.Assert(t, len(runningJobs) == 1)

				var foundOwnershipRef bool
				for _, r := range jobs.Items[0].GetOwnerReferences() {
					if r.Kind == "PostgresCluster" && r.Name == clusterName &&
						r.UID == cluster.GetUID() {
						foundOwnershipRef = true
						break
					}
				}
				assert.Assert(t, foundOwnershipRef)
			} else {
				// verify expected results when a reconcile is not expected
				jobs := &batchv1.JobList{}
				// use a pgupgrade selector to check for the existence of any jobs
				err := tClient.List(ctx, jobs, &client.ListOptions{
					LabelSelector: naming.PGUpgradeJobSelector(clusterName),
				})
				assert.NilError(t, err)
				runningJobs := []*batchv1.Job{}
				for i := range jobs.Items {
					if !jobFailed(&jobs.Items[i]) && !jobCompleted(&jobs.Items[i]) {
						runningJobs = append(runningJobs, &jobs.Items[i])
					}
				}
				assert.Assert(t, len(runningJobs) == 0)
			}

			progressing := meta.FindStatusCondition(cluster.Status.Conditions,
				ConditionPGUpgradeProgressing)
			if progressing != nil && progressing.Reason == ReasonClusterShutdown {
				assert.Equal(t, *cluster.Spec.Shutdown, true)
				assert.Equal(t, cluster.Spec.PostgresVersion, cluster.Spec.Upgrade.FromPostgresVersion)
			}
		})
	}
}

func TestObserveUpgradeEnv(t *testing.T) {
	tEnv, tClient := setupKubernetes(t)

	// TODO(cbandy): Assume this should run alone for now.
	require.ParallelCapacity(t, 99)

	r := &Reconciler{}
	ctx, cancel := setupManager(t, tEnv.Config, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	namespace := setupNamespace(t, tClient).Name

	generateJob := func(clusterName string, completed, failed *bool) *batchv1.Job {

		cluster := &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: namespace,
			},
		}
		meta := naming.PGUpgradeJob(cluster)
		labels := naming.PGUpgradeJobLabels(cluster.Name)
		meta.Labels = labels
		meta.Annotations = map[string]string{
			naming.PGBackRestConfigHash: "testhash",
			naming.PGUpgradeVersion:     "13",
		}

		upgradeJob := &batchv1.Job{
			ObjectMeta: meta,
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: meta,
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "test",
							Name:  naming.ContainerPGUpgrade,
						}},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}

		if completed != nil {
			if *completed {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionTrue,
					Reason:  "test",
					Message: "test",
				})
			} else {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobComplete,
					Status:  corev1.ConditionFalse,
					Reason:  "test",
					Message: "test",
				})
			}
		} else if failed != nil {
			if *failed {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Reason:  "test",
					Message: "test",
				})
			} else {
				upgradeJob.Status.Conditions = append(upgradeJob.Status.Conditions, batchv1.JobCondition{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionFalse,
					Reason:  "test",
					Message: "test",
				})
			}
		}

		return upgradeJob
	}

	type testResult struct {
		foundUpgradeJob          bool
		endpointCount            int
		expectedClusterCondition *metav1.Condition
	}

	testCases := []struct {
		desc            string
		createResources func(t *testing.T, cluster *v1beta1.PostgresCluster)
		result          testResult
	}{{
		desc: "upgrade job and all patroni endpoints exist",
		createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
			fakeLeaderEP := &corev1.Endpoints{}
			fakeLeaderEP.ObjectMeta = naming.PatroniLeaderEndpoints(cluster)
			fakeLeaderEP.ObjectMeta.Namespace = namespace
			assert.NilError(t, r.Client.Create(ctx, fakeLeaderEP))
			fakeDCSEP := &corev1.Endpoints{}
			fakeDCSEP.ObjectMeta = naming.PatroniDistributedConfiguration(cluster)
			fakeDCSEP.ObjectMeta.Namespace = namespace
			assert.NilError(t, r.Client.Create(ctx, fakeDCSEP))
			fakeFailoverEP := &corev1.Endpoints{}
			fakeFailoverEP.ObjectMeta = naming.PatroniTrigger(cluster)
			fakeFailoverEP.ObjectMeta.Namespace = namespace
			assert.NilError(t, r.Client.Create(ctx, fakeFailoverEP))

			job := generateJob(cluster.Name, initialize.Bool(false), initialize.Bool(false))
			assert.NilError(t, r.Client.Create(ctx, job))
		},
		result: testResult{
			foundUpgradeJob:          true,
			endpointCount:            3,
			expectedClusterCondition: nil,
		},
	}, {
		desc: "patroni endpoints only exist",
		createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
			fakeLeaderEP := &corev1.Endpoints{}
			fakeLeaderEP.ObjectMeta = naming.PatroniLeaderEndpoints(cluster)
			fakeLeaderEP.ObjectMeta.Namespace = namespace
			assert.NilError(t, r.Client.Create(ctx, fakeLeaderEP))
			fakeDCSEP := &corev1.Endpoints{}
			fakeDCSEP.ObjectMeta = naming.PatroniDistributedConfiguration(cluster)
			fakeDCSEP.ObjectMeta.Namespace = namespace
			assert.NilError(t, r.Client.Create(ctx, fakeDCSEP))
			fakeFailoverEP := &corev1.Endpoints{}
			fakeFailoverEP.ObjectMeta = naming.PatroniTrigger(cluster)
			fakeFailoverEP.ObjectMeta.Namespace = namespace
			assert.NilError(t, r.Client.Create(ctx, fakeFailoverEP))
		},
		result: testResult{
			foundUpgradeJob:          false,
			endpointCount:            3,
			expectedClusterCondition: nil,
		},
	}, {
		desc: "upgrade job only exists",
		createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
			job := generateJob(cluster.Name, initialize.Bool(false), initialize.Bool(false))
			assert.NilError(t, r.Client.Create(ctx, job))
		},
		result: testResult{
			foundUpgradeJob:          true,
			endpointCount:            0,
			expectedClusterCondition: nil,
		},
	}, {
		desc: "upgrade job completed condition true",
		createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
			if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires mocking of Job conditions")
			}
			job := generateJob(cluster.Name, initialize.Bool(true), nil)
			assert.NilError(t, r.Client.Create(ctx, job.DeepCopy()))
			assert.NilError(t, r.Client.Status().Update(ctx, job))
		},
		result: testResult{
			foundUpgradeJob: true,
			endpointCount:   0,
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeCompleted,
				Status:  metav1.ConditionTrue,
				Reason:  "PGUpgradeComplete",
				Message: "pg_upgrade completed successfully",
			},
		},
	}, {
		desc: "upgrade job completed condition false",
		createResources: func(t *testing.T, cluster *v1beta1.PostgresCluster) {
			if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
				t.Skip("requires mocking of Job conditions")
			}
			job := generateJob(cluster.Name, nil, initialize.Bool(true))
			assert.NilError(t, r.Client.Create(ctx, job.DeepCopy()))
			assert.NilError(t, r.Client.Status().Update(ctx, job))
		},
		result: testResult{
			foundUpgradeJob: true,
			endpointCount:   0,
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeCompleted,
				Status:  metav1.ConditionFalse,
				Reason:  "PGUpgradeFailed",
				Message: "pg_upgrade failed",
			},
		},
	}}

	for i, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {

			clusterName := "observe-upgrade-env" + strconv.Itoa(i)
			clusterUID := clusterName
			cluster := fakeUpgradeCluster(clusterName, namespace, clusterUID)
			tc.createResources(t, cluster)

			endpoints, job, err := r.observeUpgradeEnv(ctx, cluster)
			assert.NilError(t, err)

			assert.Assert(t, tc.result.foundUpgradeJob == (job != nil))
			assert.Assert(t, tc.result.endpointCount == len(endpoints))

			if tc.result.expectedClusterCondition != nil {
				condition := meta.FindStatusCondition(cluster.Status.Conditions,
					tc.result.expectedClusterCondition.Type)
				if assert.Check(t, condition != nil) {
					assert.Equal(t, tc.result.expectedClusterCondition.Status, condition.Status)
					assert.Equal(t, tc.result.expectedClusterCondition.Reason, condition.Reason)
					assert.Equal(t, tc.result.expectedClusterCondition.Message, condition.Message)
				}
			}
		})
	}
}

func TestPrepareForUpgrade(t *testing.T) {
	tEnv, tClient := setupKubernetes(t)

	// TODO(cbandy): Assume this should run alone for now.
	require.ParallelCapacity(t, 99)

	r := &Reconciler{}
	ctx, cancel := setupManager(t, tEnv.Config, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	ns := setupNamespace(t, tClient)

	generateJob := func(clusterName string) *batchv1.Job {

		cluster := &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: ns.GetName(),
			},
		}
		meta := naming.PGUpgradeJob(cluster)
		labels := naming.PGUpgradeJobLabels(cluster.Name)
		meta.Labels = labels
		meta.Annotations = map[string]string{naming.PGUpgradeVersion: "13"}

		upgradeJob := &batchv1.Job{
			ObjectMeta: meta,
			Spec: batchv1.JobSpec{
				Template: corev1.PodTemplateSpec{
					ObjectMeta: meta,
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{
							Image: "test",
							Name:  naming.ContainerPGUpgrade,
						}},
						RestartPolicy: corev1.RestartPolicyNever,
					},
				},
			},
		}

		return upgradeJob
	}

	generateRunner := func(name string) *appsv1.StatefulSet {
		return &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Name: name, Namespace: ns.Name,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{"test": "test"},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{"test": "test"},
					},
				},
			},
		}
	}

	generateDCS := func(cluster *v1beta1.PostgresCluster) (ep []corev1.Endpoints) {
		leaderEP := corev1.Endpoints{}
		leaderEP.ObjectMeta = naming.PatroniLeaderEndpoints(cluster)
		leaderEP.ObjectMeta.Namespace = ns.Name
		ep = append(ep, leaderEP)
		dcsEP := corev1.Endpoints{}
		dcsEP.ObjectMeta = naming.PatroniDistributedConfiguration(cluster)
		dcsEP.ObjectMeta.Namespace = ns.Name
		ep = append(ep, dcsEP)
		failoverEP := corev1.Endpoints{}
		failoverEP.ObjectMeta = naming.PatroniTrigger(cluster)
		failoverEP.ObjectMeta.Namespace = ns.Name
		ep = append(ep, failoverEP)
		return
	}

	generatePVC := func(instanceName string) *corev1.PersistentVolumeClaim {
		return &corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{
				Name:      instanceName,
				Namespace: ns.GetName(),
				Labels: map[string]string{
					naming.LabelData:     naming.DataPostgres,
					naming.LabelInstance: instanceName,
				},
			},
			Spec: corev1.PersistentVolumeClaimSpec{
				AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceStorage: resource.MustParse("1Gi"),
					},
				},
			},
		}
	}

	type testResources struct {
		upgradeJob *batchv1.Job
		endpoints  []corev1.Endpoints
		runners    []appsv1.StatefulSet
		pvcs       []corev1.PersistentVolumeClaim
	}

	type testResult struct {
		upgradeJobCount          int
		endpointCount            int
		runnerCount              int
		pvcCount                 int
		expectedClusterCondition *metav1.Condition
	}
	const primaryInstanceName = "primary-instance"
	const primaryInstanceSetName = "primary-instance-set"

	testCases := []struct {
		desc            string
		createResources func(t *testing.T,
			cluster *v1beta1.PostgresCluster) testResources
		instances *observedInstances
		result    testResult
	}{{
		desc: "remove upgrade jobs",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) testResources {
			job := generateJob(cluster.Name)
			assert.NilError(t, r.Client.Create(ctx, job))
			return testResources{
				upgradeJob: job,
			}
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "PreparingPGForPGUpgrade",
				Message: "Preparing cluster for upgrade: removing existing upgrade job",
			},
		},
	}, {
		desc: "remove runners",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) (tr testResources) {
			runner1 := generateRunner("runner1")
			runner2 := generateRunner("runner2")
			runner3 := generateRunner("runner3")
			assert.NilError(t, r.Client.Create(ctx, runner1))
			assert.NilError(t, r.Client.Create(ctx, runner2))
			assert.NilError(t, r.Client.Create(ctx, runner3))
			tr.runners = []appsv1.StatefulSet{*runner1, *runner2, *runner3}
			return
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "PreparingPGForPGUpgrade",
				Message: "Preparing cluster for upgrade: removing runners",
			},
		},
		instances: &observedInstances{forCluster: []*Instance{{
			Name: primaryInstanceName,
			Spec: &v1beta1.PostgresInstanceSetSpec{Name: primaryInstanceSetName},
			Runner: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner1", Namespace: ns.GetName(),
				},
			},
		}}},
	}, {
		desc: "remove endpoints",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) (tr testResources) {
			dcs := generateDCS(cluster)
			for i := range dcs {
				assert.NilError(t, r.Client.Create(ctx, &dcs[i]))
			}
			tr.endpoints = dcs
			return
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "PreparingPGForPGUpgrade",
				Message: "Preparing cluster for upgrade: removing DCS",
			},
		},
	}, {
		desc: "remove replica pvcs",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) (tr testResources) {
			createdPVCs := []corev1.PersistentVolumeClaim{}
			primaryPVC := generatePVC(primaryInstanceName)
			assert.NilError(t, r.Client.Create(ctx, primaryPVC))
			createdPVCs = append(createdPVCs, *primaryPVC)
			replicaPVC1 := generatePVC("replica1")
			assert.NilError(t, r.Client.Create(ctx, replicaPVC1))
			createdPVCs = append(createdPVCs, *replicaPVC1)
			replicaPVC2 := generatePVC("replica2")
			assert.NilError(t, r.Client.Create(ctx, replicaPVC2))
			createdPVCs = append(createdPVCs, *replicaPVC2)
			tr.pvcs = createdPVCs
			return
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  "PreparingPGForPGUpgrade",
				Message: "Preparing cluster for upgrade: removing replica PVCs",
			},
		},
	}, {
		desc: "cluster fully prepared, primary as startup instance",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) (tr testResources) {
			// set to ensure any previous upgrade completion condition is removed
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				Type:   ConditionPGUpgradeCompleted,
				Reason: "test",
			})
			return tr
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonReadyForUpgrade,
				Message: "Upgrading cluster postgres major version",
			},
		},
	}, {
		desc: "PG still running, shutdown cluster",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) (tr testResources) {
			return
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonClusterShutdown,
				Message: "Preparing cluster for upgrade: shutting down cluster",
			},
		},
		instances: &observedInstances{forCluster: []*Instance{{
			Name: primaryInstanceName,
			Spec: &v1beta1.PostgresInstanceSetSpec{Name: primaryInstanceSetName},
			Pods: []*corev1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{naming.LabelRole: naming.RolePatroniLeader},
				},
			}}},
		}},
	}, {
		desc: "pgBackRest repo host still running, shutdown cluster",
		createResources: func(t *testing.T,
			cluster *v1beta1.PostgresCluster) (tr testResources) {
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				Type:    ConditionRepoHostReady,
				Status:  metav1.ConditionTrue,
				Reason:  "test",
				Message: "test",
			})
			return
		},
		result: testResult{
			expectedClusterCondition: &metav1.Condition{
				Type:    ConditionPGUpgradeProgressing,
				Status:  metav1.ConditionTrue,
				Reason:  ReasonClusterShutdown,
				Message: "Preparing cluster for upgrade: shutting down cluster",
			},
		},
	}}

	for i, tc := range testCases {
		name := tc.desc
		t.Run(name, func(t *testing.T) {

			clusterName := "prepare-for-upgrade-" + strconv.Itoa(i)
			clusterUID := clusterName
			cluster := fakeUpgradeCluster(clusterName, ns.Name, clusterUID)
			cluster.Status.StartupInstance = primaryInstanceName
			cluster.Status.StartupInstanceSet = primaryInstanceSetName
			cluster.Status.Patroni.SystemIdentifier = "abcde12345"
			cluster.Status.Proxy.PGBouncer.PostgreSQLRevision = "abcde12345"
			cluster.Status.Monitoring.ExporterConfiguration = "abcde12345"
			meta.SetStatusCondition(&cluster.Status.Conditions, metav1.Condition{
				ObservedGeneration: cluster.GetGeneration(),
				Type:               ConditionPostgresDataInitialized,
				Status:             metav1.ConditionTrue,
				Reason:             "test",
				Message:            "test",
			})

			testResources := tc.createResources(t, cluster)

			fakeObserved := &observedInstances{}
			if tc.instances != nil {
				fakeObserved = tc.instances
			}

			assert.NilError(t, r.prepareForUpgrade(ctx, cluster, fakeObserved,
				testResources.endpoints, testResources.upgradeJob, testResources.pvcs))

			currentUpgradeJobCount := 0
			if testResources.upgradeJob != nil {
				err := r.Client.Get(ctx, client.ObjectKeyFromObject(testResources.upgradeJob),
					&batchv1.Job{})
				assert.NilError(t, client.IgnoreNotFound(err))
				if err == nil {
					currentUpgradeJobCount = 1
				}
			}
			assert.Equal(t, tc.result.upgradeJobCount, currentUpgradeJobCount)

			currentEndpointCount := 0
			for i := range testResources.endpoints {
				err := r.Client.Get(ctx, client.ObjectKeyFromObject(&testResources.endpoints[i]),
					&batchv1.Job{})
				assert.NilError(t, client.IgnoreNotFound(err))
				if err == nil {
					currentEndpointCount++
				}
			}
			assert.Assert(t, currentEndpointCount == tc.result.endpointCount)

			currentRunnerCount := 0
			for i := range testResources.runners {
				err := r.Client.Get(ctx, client.ObjectKeyFromObject(&testResources.runners[i]),
					&batchv1.Job{})
				assert.NilError(t, client.IgnoreNotFound(err))
				if err == nil {
					currentRunnerCount++
				}
			}
			assert.Assert(t, currentRunnerCount == tc.result.runnerCount)

			currentPVCCount := 0
			for i := range testResources.pvcs {
				err := r.Client.Get(ctx, client.ObjectKeyFromObject(&testResources.pvcs[i]),
					&batchv1.Job{})
				assert.NilError(t, client.IgnoreNotFound(err))
				if err == nil {
					currentPVCCount++
				}
			}
			assert.Assert(t, currentPVCCount == tc.result.pvcCount)

			if tc.result.expectedClusterCondition != nil {
				condition := meta.FindStatusCondition(cluster.Status.Conditions,
					tc.result.expectedClusterCondition.Type)
				if assert.Check(t, condition != nil) {
					assert.Equal(t, tc.result.expectedClusterCondition.Status, condition.Status)
					assert.Equal(t, tc.result.expectedClusterCondition.Reason, condition.Reason)
					assert.Equal(t, tc.result.expectedClusterCondition.Message, condition.Message)
				}
				if tc.result.expectedClusterCondition.Reason == ReasonReadyForUpgrade {
					assert.Assert(t, cluster.Status.Patroni.SystemIdentifier == "")
					assert.Assert(t, cluster.Status.Proxy.PGBouncer.PostgreSQLRevision == "")
					assert.Assert(t, cluster.Status.Monitoring.ExporterConfiguration == "")
					assert.Assert(t, meta.FindStatusCondition(cluster.Status.Conditions,
						ConditionPGUpgradeCompleted) == nil)
				}
			}

			// ensure the PostgresDataInitialized condition is never removed
			assert.Assert(t, meta.FindStatusCondition(cluster.Status.Conditions,
				ConditionPostgresDataInitialized) != nil)
		})
	}
}

func fakeUpgradeCluster(clusterName, namespace, clusterUID string) *v1beta1.PostgresCluster {
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: namespace,
			UID:       types.UID(clusterUID),
		},
		Spec: v1beta1.PostgresClusterSpec{
			Port:            initialize.Int32(5432),
			Shutdown:        initialize.Bool(false),
			PostgresVersion: 13,
			ImagePullSecrets: []corev1.LocalObjectReference{{
				Name: "myImagePullSecret"},
			},
			Image: "example.com/crunchy-postgres-ha:test",
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name: "instance1",
				DataVolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: map[corev1.ResourceName]resource.Quantity{
							corev1.ResourceStorage: resource.MustParse("1Gi"),
						},
					},
				},
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: corev1.PersistentVolumeClaimSpec{
								AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
								Resources: corev1.ResourceRequirements{
									Requests: map[corev1.ResourceName]resource.Quantity{
										corev1.ResourceStorage: resource.MustParse("1Gi"),
									},
								},
							},
						},
					}},
				},
			},
		},
	}

	return cluster
}

func TestReconcileInstanceSetPodDisruptionBudget(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	foundPDB := func(
		cluster *v1beta1.PostgresCluster,
		spec *v1beta1.PostgresInstanceSetSpec,
	) bool {
		got := &policyv1beta1.PodDisruptionBudget{}
		err := r.Client.Get(ctx,
			naming.AsObjectKey(naming.InstanceSet(cluster, spec)),
			got)
		return !apierrors.IsNotFound(err)

	}

	ns := setupNamespace(t, cc)

	t.Run("empty", func(t *testing.T) {
		cluster := &v1beta1.PostgresCluster{}
		spec := &v1beta1.PostgresInstanceSetSpec{}

		assert.Error(t, r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec),
			"Replicas should be defined")
	})

	t.Run("not created", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		spec := &cluster.Spec.InstanceSets[0]
		spec.MinAvailable = initialize.IntOrStringInt32(0)
		assert.NilError(t, r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec))
		assert.Assert(t, !foundPDB(cluster, spec))
	})

	t.Run("int created", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		spec := &cluster.Spec.InstanceSets[0]
		spec.MinAvailable = initialize.IntOrStringInt32(1)

		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		assert.NilError(t, r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec))
		assert.Assert(t, foundPDB(cluster, spec))

		t.Run("deleted", func(t *testing.T) {
			spec.MinAvailable = initialize.IntOrStringInt32(0)
			err := r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec)
			if apierrors.IsConflict(err) {
				// When running in an existing environment another controller will sometimes update
				// the object. This leads to an error where the ResourceVersion of the object does
				// not match what we expect. When we run into this conflict, try to reconcile the
				// object again.
				err = r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec)
			}
			assert.NilError(t, err, errors.Unwrap(err))
			assert.Assert(t, !foundPDB(cluster, spec))
		})
	})

	t.Run("str created", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		spec := &cluster.Spec.InstanceSets[0]
		spec.MinAvailable = initialize.IntOrStringString("50%")

		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		assert.NilError(t, r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec))
		assert.Assert(t, foundPDB(cluster, spec))

		t.Run("deleted", func(t *testing.T) {
			spec.MinAvailable = initialize.IntOrStringString("0%")
			err := r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec)
			if apierrors.IsConflict(err) {
				// When running in an existing environment another controller will sometimes update
				// the object. This leads to an error where the ResourceVersion of the object does
				// not match what we expect. When we run into this conflict, try to reconcile the
				// object again.
				err = r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec)
			}
			assert.NilError(t, err, errors.Unwrap(err))
			assert.Assert(t, !foundPDB(cluster, spec))
		})

		t.Run("delete with 00%", func(t *testing.T) {
			spec.MinAvailable = initialize.IntOrStringString("50%")

			assert.NilError(t, r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec))
			assert.Assert(t, foundPDB(cluster, spec))

			t.Run("deleted", func(t *testing.T) {
				spec.MinAvailable = initialize.IntOrStringString("00%")
				err := r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec)
				if apierrors.IsConflict(err) {
					// When running in an existing environment another controller will sometimes update
					// the object. This leads to an error where the ResourceVersion of the object does
					// not match what we expect. When we run into this conflict, try to reconcile the
					// object again.
					t.Log("conflict:", err)
					err = r.reconcileInstanceSetPodDisruptionBudget(ctx, cluster, spec)
				}
				assert.NilError(t, err, "\n%#v", errors.Unwrap(err))
				assert.Assert(t, !foundPDB(cluster, spec))
			})
		})
	})
}

func TestCleanupDisruptionBudgets(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	r := &Reconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := setupNamespace(t, cc)

	generatePDB := func(
		t *testing.T,
		cluster *v1beta1.PostgresCluster,
		spec *v1beta1.PostgresInstanceSetSpec,
		minAvailable *intstr.IntOrString,
	) *policyv1beta1.PodDisruptionBudget {
		meta := naming.InstanceSet(cluster, spec)
		meta.Labels = map[string]string{
			naming.LabelCluster:     cluster.Name,
			naming.LabelInstanceSet: spec.Name,
		}
		pdb, err := r.generatePodDisruptionBudget(
			cluster,
			meta,
			minAvailable,
			naming.ClusterInstanceSet(cluster.Name, spec.Name),
		)
		assert.NilError(t, err)
		return pdb
	}

	createPDB := func(
		pdb *policyv1beta1.PodDisruptionBudget,
	) error {
		return r.Client.Create(ctx, pdb)
	}

	foundPDB := func(
		pdb *policyv1beta1.PodDisruptionBudget,
	) bool {
		return !apierrors.IsNotFound(
			r.Client.Get(ctx, client.ObjectKeyFromObject(pdb),
				&policyv1beta1.PodDisruptionBudget{}))
	}

	t.Run("pdbs not found", func(t *testing.T) {
		cluster := testCluster()
		assert.NilError(t, r.cleanupPodDisruptionBudgets(ctx, cluster))
	})

	t.Run("pdbs found", func(t *testing.T) {
		cluster := testCluster()
		cluster.Namespace = ns.Name
		spec := &cluster.Spec.InstanceSets[0]
		spec.MinAvailable = initialize.IntOrStringInt32(1)

		assert.NilError(t, r.Client.Create(ctx, cluster))
		t.Cleanup(func() { assert.Check(t, r.Client.Delete(ctx, cluster)) })

		expectedPDB := generatePDB(t, cluster, spec,
			initialize.IntOrStringInt32(1))
		assert.NilError(t, createPDB(expectedPDB))

		t.Run("no instances were removed", func(t *testing.T) {
			assert.Assert(t, foundPDB(expectedPDB))
			assert.NilError(t, r.cleanupPodDisruptionBudgets(ctx, cluster))
			assert.Assert(t, foundPDB(expectedPDB))
		})

		t.Run("cleanup leftover pdb", func(t *testing.T) {
			leftoverPDB := generatePDB(t, cluster, &v1beta1.PostgresInstanceSetSpec{
				Name:     "old-instance",
				Replicas: initialize.Int32(1),
			}, initialize.IntOrStringInt32(1))
			assert.NilError(t, createPDB(leftoverPDB))

			assert.Assert(t, foundPDB(expectedPDB))
			assert.Assert(t, foundPDB(leftoverPDB))
			err := r.cleanupPodDisruptionBudgets(ctx, cluster)

			// The disruption controller updates the status of a PDB any time a
			// related Pod changes. When this happens, the resourceVersion of
			// the PDB does not match what we expect and we get a conflict. Retry.
			if apierrors.IsConflict(err) {
				t.Log("conflict:", err)
				err = r.cleanupPodDisruptionBudgets(ctx, cluster)
			}

			assert.NilError(t, err, "\n%#v", errors.Unwrap(err))
			assert.Assert(t, foundPDB(expectedPDB))
			assert.Assert(t, !foundPDB(leftoverPDB))
		})
	})
}
