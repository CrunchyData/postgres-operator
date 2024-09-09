// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePatroniLeaderLeaseService(t *testing.T) {
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{
		Client:   cc,
		Recorder: new(record.FakeRecorder),
	}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns1"
	cluster.Name = "pg2"
	cluster.Spec.Port = initialize.Int32(9876)

	alwaysExpect := func(t testing.TB, service *corev1.Service) {
		assert.Assert(t, cmp.MarshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, cmp.MarshalMatches(service.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/cluster: pg2
  postgres-operator.crunchydata.com/patroni: pg2-ha
name: pg2-ha
namespace: ns1
ownerReferences:
- apiVersion: postgres-operator.crunchydata.com/v1beta1
  blockOwnerDeletion: true
  controller: true
  kind: PostgresCluster
  name: pg2
  uid: ""
		`))

		// Always gets a ClusterIP (never None).
		assert.Equal(t, service.Spec.ClusterIP, "")
		assert.Assert(t, service.Spec.Selector == nil,
			"got %v", service.Spec.Selector)
	}

	t.Run("NoServiceSpec", func(t *testing.T) {
		service, err := reconciler.generatePatroniLeaderLeaseService(cluster)
		assert.NilError(t, err)
		alwaysExpect(t, service)
		// Defaults to ClusterIP.
		assert.Equal(t, service.Spec.Type, corev1.ServiceTypeClusterIP)
		assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: postgres
  port: 9876
  protocol: TCP
  targetPort: postgres
		`))
	})

	t.Run("AnnotationsLabels", func(t *testing.T) {
		cluster := cluster.DeepCopy()
		cluster.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{"a": "v1"},
			Labels:      map[string]string{"b": "v2"},
		}

		service, err := reconciler.generatePatroniLeaderLeaseService(cluster)
		assert.NilError(t, err)

		// Annotations present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Annotations, map[string]string{
			"a": "v1",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Labels, map[string]string{
			"b": "v2",
			"postgres-operator.crunchydata.com/cluster": "pg2",
			"postgres-operator.crunchydata.com/patroni": "pg2-ha",
		})

		// Labels not in the selector.
		assert.Assert(t, service.Spec.Selector == nil,
			"got %v", service.Spec.Selector)

		// Add metadata to individual service
		cluster.Spec.Service = &v1beta1.ServiceSpec{
			Metadata: &v1beta1.Metadata{
				Annotations: map[string]string{"c": "v3"},
				Labels: map[string]string{"d": "v4",
					"postgres-operator.crunchydata.com/cluster": "wrongName"},
			},
		}

		service, err = reconciler.generatePatroniLeaderLeaseService(cluster)
		assert.NilError(t, err)

		// Annotations present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Annotations, map[string]string{
			"a": "v1",
			"c": "v3",
		})

		// Labels present in the metadata.
		assert.DeepEqual(t, service.ObjectMeta.Labels, map[string]string{
			"b": "v2",
			"d": "v4",
			"postgres-operator.crunchydata.com/cluster": "pg2",
			"postgres-operator.crunchydata.com/patroni": "pg2-ha",
		})

		// Labels not in the selector.
		assert.Assert(t, service.Spec.Selector == nil,
			"got %v", service.Spec.Selector)
	})

	types := []struct {
		Type   string
		Expect func(testing.TB, *corev1.Service)
	}{
		{Type: "ClusterIP", Expect: func(t testing.TB, service *corev1.Service) {
			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeClusterIP)
		}},
		{Type: "NodePort", Expect: func(t testing.TB, service *corev1.Service) {
			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeNodePort)
		}},
		{Type: "LoadBalancer", Expect: func(t testing.TB, service *corev1.Service) {
			assert.Equal(t, service.Spec.Type, corev1.ServiceTypeLoadBalancer)
		}},
	}

	for _, test := range types {
		t.Run(test.Type, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Service = &v1beta1.ServiceSpec{Type: test.Type}

			service, err := reconciler.generatePatroniLeaderLeaseService(cluster)
			assert.NilError(t, err)
			alwaysExpect(t, service)
			test.Expect(t, service)
			assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: postgres
  port: 9876
  protocol: TCP
  targetPort: postgres
		`))
		})
	}

	typesAndPort := []struct {
		Description string
		Type        string
		NodePort    *int32
		Expect      func(testing.TB, *corev1.Service, error)
	}{
		{Description: "ClusterIP with Port 32000", Type: "ClusterIP",
			NodePort: initialize.Int32(32000), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.ErrorContains(t, err, "NodePort cannot be set with type ClusterIP on Service \"pg2-ha\"")
				assert.Assert(t, service == nil)
			}},
		{Description: "NodePort with Port 32001", Type: "NodePort",
			NodePort: initialize.Int32(32001), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.NilError(t, err)
				alwaysExpect(t, service)
				assert.Equal(t, service.Spec.Type, corev1.ServiceTypeNodePort)
				assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: postgres
  nodePort: 32001
  port: 9876
  protocol: TCP
  targetPort: postgres
`))
			}},
		{Description: "LoadBalancer with Port 32002", Type: "LoadBalancer",
			NodePort: initialize.Int32(32002), Expect: func(t testing.TB, service *corev1.Service, err error) {
				assert.Equal(t, service.Spec.Type, corev1.ServiceTypeLoadBalancer)
				assert.NilError(t, err)
				alwaysExpect(t, service)
				assert.Assert(t, cmp.MarshalMatches(service.Spec.Ports, `
- name: postgres
  nodePort: 32002
  port: 9876
  protocol: TCP
  targetPort: postgres
`))
			}},
	}

	for _, test := range typesAndPort {
		t.Run(test.Description, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Service = &v1beta1.ServiceSpec{Type: test.Type, NodePort: test.NodePort}

			service, err := reconciler.generatePatroniLeaderLeaseService(cluster)
			test.Expect(t, service, err)
		})
	}
}

func TestReconcilePatroniLeaderLease(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, cc)
	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := testCluster()
	cluster.Namespace = ns.Name
	assert.NilError(t, cc.Create(ctx, cluster))

	t.Run("NoServiceSpec", func(t *testing.T) {
		service, err := reconciler.reconcilePatroniLeaderLease(ctx, cluster)
		assert.NilError(t, err)
		assert.Assert(t, service != nil)
		t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, service)) })

		assert.Assert(t, service.Spec.ClusterIP != "",
			"expected to be assigned a ClusterIP")
	})

	serviceTypes := []string{"ClusterIP", "NodePort", "LoadBalancer"}

	// Confirm that each ServiceType can be reconciled.
	for _, serviceType := range serviceTypes {
		t.Run(serviceType, func(t *testing.T) {
			cluster := cluster.DeepCopy()
			cluster.Spec.Service = &v1beta1.ServiceSpec{Type: serviceType}

			service, err := reconciler.reconcilePatroniLeaderLease(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, service != nil)
			t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, service)) })

			assert.Assert(t, service.Spec.ClusterIP != "",
				"expected to be assigned a ClusterIP")
		})
	}

	// CRD validation looks only at the new/incoming value of fields. Confirm
	// that each ServiceType can change to any other ServiceType. Forbidding
	// certain transitions requires a validating webhook.
	serviceTypeChangeClusterCounter := 0
	for _, beforeType := range serviceTypes {
		for _, changeType := range serviceTypes {
			t.Run(beforeType+"To"+changeType, func(t *testing.T) {
				// Creating fresh clusters for these tests
				cluster := testCluster()
				cluster.Namespace = ns.Name

				// Note (dsessler): Adding a number to each cluster name to make cluster/service
				// names unique to work around an intermittent race condition where a service
				// from a prior test has not been deleted yet when the next test runs, causing
				// the test to fail due to non-matching IP addresses.
				cluster.Name += "-" + strconv.Itoa(serviceTypeChangeClusterCounter)
				assert.NilError(t, cc.Create(ctx, cluster))

				cluster.Spec.Service = &v1beta1.ServiceSpec{Type: beforeType}

				before, err := reconciler.reconcilePatroniLeaderLease(ctx, cluster)
				assert.NilError(t, err)
				t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, before)) })

				cluster.Spec.Service.Type = changeType

				after, err := reconciler.reconcilePatroniLeaderLease(ctx, cluster)

				// LoadBalancers are provisioned by a separate controller that
				// updates the Service soon after creation. The API may return
				// a conflict error when we race to update it, even though we
				// don't send a resourceVersion in our payload. Retry.
				if apierrors.IsConflict(err) {
					t.Log("conflict:", err)
					after, err = reconciler.reconcilePatroniLeaderLease(ctx, cluster)
				}

				assert.NilError(t, err, "\n%#v", errors.Unwrap(err))
				assert.Equal(t, after.Spec.ClusterIP, before.Spec.ClusterIP,
					"expected to keep the same ClusterIP")
				serviceTypeChangeClusterCounter++
			})
		}
	}
}

func TestPatroniReplicationSecret(t *testing.T) {
	// Garbage collector cleans up test resources before the test completes
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("USE_EXISTING_CLUSTER: Test fails due to garbage collection")
	}

	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ctx := context.Background()
	r := &Reconciler{Client: tClient, Owner: client.FieldOwner(t.Name())}

	// test postgrescluster values
	var (
		clusterName = "hippocluster"
		clusterUID  = types.UID("hippouid")
	)

	// create a PostgresCluster to test with
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: setupNamespace(t, tClient).Name,
			UID:       clusterUID,
		},
	}

	rootCA, err := r.reconcileRootCertificate(ctx, postgresCluster)
	assert.NilError(t, err)

	t.Run("reconcile", func(t *testing.T) {
		_, err = r.reconcileReplicationSecret(ctx, postgresCluster, rootCA)
		assert.NilError(t, err)
	})

	t.Run("validate", func(t *testing.T) {

		patroniReplicationSecret := &corev1.Secret{ObjectMeta: naming.ReplicationClientCertSecret(postgresCluster)}
		patroniReplicationSecret.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		err = r.Client.Get(ctx, client.ObjectKeyFromObject(patroniReplicationSecret), patroniReplicationSecret)
		assert.NilError(t, err)

		t.Run("ca.crt", func(t *testing.T) {

			clientCert, ok := patroniReplicationSecret.Data["ca.crt"]
			assert.Assert(t, ok)

			assert.Assert(t, strings.HasPrefix(string(clientCert), "-----BEGIN CERTIFICATE-----"))
			assert.Assert(t, strings.HasSuffix(string(clientCert), "-----END CERTIFICATE-----\n"))
		})

		t.Run("tls.crt", func(t *testing.T) {

			clientCert, ok := patroniReplicationSecret.Data["tls.crt"]
			assert.Assert(t, ok)

			assert.Assert(t, strings.HasPrefix(string(clientCert), "-----BEGIN CERTIFICATE-----"))
			assert.Assert(t, strings.HasSuffix(string(clientCert), "-----END CERTIFICATE-----\n"))
		})

		t.Run("tls.key", func(t *testing.T) {

			clientKey, ok := patroniReplicationSecret.Data["tls.key"]
			assert.Assert(t, ok)

			assert.Assert(t, strings.HasPrefix(string(clientKey), "-----BEGIN EC PRIVATE KEY-----"))
			assert.Assert(t, strings.HasSuffix(string(clientKey), "-----END EC PRIVATE KEY-----\n"))
		})

	})

	t.Run("check replication certificate secret projection", func(t *testing.T) {
		// example auto-generated secret projection
		testSecretProjection := &corev1.SecretProjection{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: naming.ReplicationClientCertSecret(postgresCluster).Name,
			},
			Items: []corev1.KeyToPath{
				{
					Key:  naming.ReplicationCert,
					Path: naming.ReplicationCertPath,
				},
				{
					Key:  naming.ReplicationPrivateKey,
					Path: naming.ReplicationPrivateKeyPath,
				},
				{
					Key:  naming.ReplicationCACert,
					Path: naming.ReplicationCACertPath,
				},
			},
		}

		rootCA, err := r.reconcileRootCertificate(ctx, postgresCluster)
		assert.NilError(t, err)

		testReplicationSecret, err := r.reconcileReplicationSecret(ctx, postgresCluster, rootCA)
		assert.NilError(t, err)

		t.Run("check standard secret projection", func(t *testing.T) {
			secretCertProj := replicationCertSecretProjection(testReplicationSecret)

			assert.DeepEqual(t, testSecretProjection, secretCertProj)
		})
	})

}

func TestReconcilePatroniStatus(t *testing.T) {
	ctx := context.Background()
	_, tClient := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	ns := setupNamespace(t, tClient)
	r := &Reconciler{Client: tClient, Owner: client.FieldOwner(t.Name())}

	systemIdentifier := "6952526174828511264"
	createResources := func(index, readyReplicas int,
		writeAnnotation bool) (*v1beta1.PostgresCluster, *observedInstances) {

		i := strconv.Itoa(index)
		clusterName := "patroni-status-" + i
		instanceName := "test-instance-" + i
		instanceSet := "set-" + i

		labels := map[string]string{
			naming.LabelCluster:     clusterName,
			naming.LabelInstanceSet: instanceSet,
			naming.LabelInstance:    instanceName,
		}

		postgresCluster := &v1beta1.PostgresCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: ns.Name,
			},
		}

		runner := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns.Name,
				Name:      instanceName,
				Labels:    labels,
			},
			Spec: appsv1.StatefulSetSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: labels,
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: labels,
					},
				},
			},
		}

		endpoints := &corev1.Endpoints{
			ObjectMeta: naming.PatroniDistributedConfiguration(postgresCluster),
		}
		if writeAnnotation {
			endpoints.ObjectMeta.Annotations = make(map[string]string)
			endpoints.ObjectMeta.Annotations["initialize"] = systemIdentifier
		}
		assert.NilError(t, tClient.Create(ctx, endpoints, &client.CreateOptions{}))

		instance := &Instance{
			Name: instanceName, Runner: runner,
		}
		for i := 0; i < readyReplicas; i++ {
			instance.Pods = append(instance.Pods, &corev1.Pod{
				Status: corev1.PodStatus{
					Conditions: []corev1.PodCondition{{
						Type:    corev1.PodReady,
						Status:  corev1.ConditionTrue,
						Reason:  "test",
						Message: "test",
					}},
				},
			})
		}
		observedInstances := &observedInstances{}
		observedInstances.forCluster = []*Instance{instance}

		return postgresCluster, observedInstances
	}

	testsCases := []struct {
		requeueExpected bool
		readyReplicas   int
		writeAnnotation bool
	}{
		{requeueExpected: false, readyReplicas: 1, writeAnnotation: true},
		{requeueExpected: true, readyReplicas: 1, writeAnnotation: false},
		{requeueExpected: false, readyReplicas: 0, writeAnnotation: false},
		{requeueExpected: false, readyReplicas: 0, writeAnnotation: false},
	}

	for i, tc := range testsCases {
		t.Run(fmt.Sprintf("%+v", tc), func(t *testing.T) {
			postgresCluster, observedInstances := createResources(i, tc.readyReplicas,
				tc.writeAnnotation)
			requeue, err := r.reconcilePatroniStatus(ctx, postgresCluster, observedInstances)
			if tc.requeueExpected {
				assert.NilError(t, err)
				assert.Equal(t, requeue, time.Second)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, requeue, time.Duration(0))
			}
		})
	}
}

func TestReconcilePatroniSwitchover(t *testing.T) {
	_, client := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	var called, failover, callError, callFails bool
	var timelineCallNoLeader, timelineCall bool
	r := Reconciler{
		Client: client,
		PodExec: func(ctx context.Context, namespace, pod, container string,
			stdin io.Reader, stdout, stderr io.Writer, command ...string) error {
			called = true
			switch {
			case timelineCall:
				timelineCall = false
				stdout.Write([]byte(`[{"Cluster": "hippo-ha", "Member": "hippo-instance1-67mc-0", "Host": "hippo-instance1-67mc-0.hippo-pods", "Role": "Leader", "State": "running", "TL": 4}, {"Cluster": "hippo-ha", "Member": "hippo-instance1-ltcf-0", "Host": "hippo-instance1-ltcf-0.hippo-pods", "Role": "Replica", "State": "running", "TL": 4, "Lag in MB": 0}]`))
			case timelineCallNoLeader:
				stdout.Write([]byte(`[{"Cluster": "hippo-ha", "Member": "hippo-instance1-ltcf-0", "Host": "hippo-instance1-ltcf-0.hippo-pods", "Role": "Replica", "State": "running", "TL": 4, "Lag in MB": 0}]`))
			case callError:
				return errors.New("boom")
			case callFails:
				stdout.Write([]byte("bang"))
			case failover:
				stdout.Write([]byte("failed over"))
			default:
				stdout.Write([]byte("switched over"))
			}
			return nil
		},
	}

	ctx := context.Background()

	getObserved := func() *observedInstances {
		instances := []*Instance{{
			Name: "target",
			Pods: []*corev1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
						State: corev1.ContainerState{
							Running: new(corev1.ContainerStateRunning),
						},
					}},
				},
			}},
			Runner: &appsv1.StatefulSet{},
		}, {
			Name: "other",
			Pods: []*corev1.Pod{{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
				},
				Status: corev1.PodStatus{
					ContainerStatuses: []corev1.ContainerStatus{{
						Name: naming.ContainerDatabase,
						State: corev1.ContainerState{
							Running: new(corev1.ContainerStateRunning),
						},
					}},
				},
			}},
			Runner: &appsv1.StatefulSet{},
		}}
		return &observedInstances{forCluster: instances}
	}

	t.Run("empty", func(t *testing.T) {
		cluster := testCluster()
		observed := newObservedInstances(cluster, nil, nil)
		assert.NilError(t, r.reconcilePatroniSwitchover(ctx, cluster, observed))
	})

	t.Run("early validation", func(t *testing.T) {
		for _, test := range []struct {
			desc    string
			enabled bool
			trigger string
			status  string
			soType  string
			target  string
			check   func(*testing.T, error, *v1beta1.PostgresCluster)
		}{
			{
				desc:    "Switchover not enabled",
				enabled: false,
				check: func(t *testing.T, err error, cluster *v1beta1.PostgresCluster) {
					assert.NilError(t, err)
					assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
					assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
				},
			},
			{
				desc:    "Switchover trigger annotation not found",
				enabled: true,
				check: func(t *testing.T, err error, cluster *v1beta1.PostgresCluster) {
					assert.NilError(t, err)
					assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
					assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
				},
			},
			{
				desc:    "Status matches trigger annotation",
				enabled: true, trigger: "triggered", status: "triggered",
				check: func(t *testing.T, err error, cluster *v1beta1.PostgresCluster) {
					assert.NilError(t, err)
					assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
					assert.Equal(t, *cluster.Status.Patroni.Switchover, "triggered")
				},
			},
			{
				desc:    "failover requested without a target",
				enabled: true, trigger: "triggered", soType: "Failover",
				check: func(t *testing.T, err error, cluster *v1beta1.PostgresCluster) {
					assert.Error(t, err, "TargetInstance required when running failover")
					assert.Equal(t, *cluster.Status.Patroni.SwitchoverTimeline, int64(2))
					assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
				},
			},
			{
				desc:    "target instance was specified but not found",
				enabled: true, trigger: "triggered", target: "bad-target",
				check: func(t *testing.T, err error, cluster *v1beta1.PostgresCluster) {
					assert.Error(t, err, "TargetInstance was specified but not found in the cluster")
					assert.Equal(t, *cluster.Status.Patroni.SwitchoverTimeline, int64(2))
					assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
				},
			},
		} {
			t.Run(test.desc, func(t *testing.T) {
				cluster := testCluster()
				cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
					Name:                "target",
					Replicas:            initialize.Int32(2),
					DataVolumeClaimSpec: testVolumeClaimSpec(),
				}}
				if test.enabled {
					cluster.Spec.Patroni = &v1beta1.PatroniSpec{
						Switchover: &v1beta1.PatroniSwitchover{
							Enabled: true,
						},
					}
				}
				if test.trigger != "" {
					cluster.Annotations = map[string]string{
						naming.PatroniSwitchover: test.trigger,
					}
				}
				if test.status != "" {
					cluster.Status = v1beta1.PostgresClusterStatus{
						Patroni: v1beta1.PatroniStatus{
							Switchover: initialize.String(test.status),
						},
					}
				}
				if test.soType != "" {
					cluster.Spec.Patroni.Switchover.Type = test.soType
				}
				if test.target != "" {
					cluster.Spec.Patroni.Switchover.TargetInstance = initialize.String(test.target)
				}
				cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(2)
				test.check(t, r.reconcilePatroniSwitchover(ctx, cluster, getObserved()), cluster)
			})
		}
	})

	t.Run("validate target instance", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled:        true,
				TargetInstance: initialize.String("target"),
			},
		}

		t.Run("has no pods", func(t *testing.T) {
			instances := []*Instance{{
				Name: "target",
			}, {
				Name: "target2",
			}}
			observed := &observedInstances{forCluster: instances}

			assert.Error(t, r.reconcilePatroniSwitchover(ctx, cluster, observed),
				"TargetInstance should have one pod. Pods (0)")
		})

		t.Run("not running", func(t *testing.T) {
			instances := []*Instance{{
				Name: "target",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
					},
				}},
				Runner: &appsv1.StatefulSet{},
			}, {Name: "other"}}
			instances[0].Pods[0].Status = corev1.PodStatus{
				ContainerStatuses: []corev1.ContainerStatus{{
					Name: naming.ContainerDatabase,
					State: corev1.ContainerState{
						Terminated: new(corev1.ContainerStateTerminated),
					},
				}},
			}
			observed := &observedInstances{forCluster: instances}

			assert.Error(t, r.reconcilePatroniSwitchover(ctx, cluster, observed),
				"Could not find a running pod when attempting switchover.")
		})
	})

	t.Run("need replica to switch", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled:        true,
				TargetInstance: initialize.String("target"),
			},
		}

		observed := &observedInstances{forCluster: []*Instance{{
			Name: "target",
		}}}
		assert.Error(t, r.reconcilePatroniSwitchover(ctx, cluster, observed),
			"Need more than one instance to switchover")
	})

	t.Run("timeline getting call errors", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		timelineCall, timelineCallNoLeader = false, false
		called, failover, callError, callFails = false, false, true, false
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.Error(t, err, "boom")
		assert.Assert(t, called)
		assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
	})

	t.Run("timeline getting call returns no leader", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		timelineCall, timelineCallNoLeader = false, true
		called, failover, callError, callFails = false, false, false, false
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.Error(t, err, "error getting and parsing current timeline")
		assert.Assert(t, called)
		assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
	})

	t.Run("timeline set", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, false, false
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.NilError(t, err)
		assert.Assert(t, called)
		assert.Equal(t, *cluster.Status.Patroni.SwitchoverTimeline, int64(4))
	})

	t.Run("timeline mismatch, timeline cleared", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(11)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, false, false
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.NilError(t, err)
		assert.Assert(t, called)
		assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
	})

	t.Run("timeline cleared when status is updated", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(11)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, false, false
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.NilError(t, err)
		assert.Assert(t, called)
		assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
	})

	t.Run("switchover call fails", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(4)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, false, true
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.Error(t, err, "unable to switchover")
		assert.Assert(t, called)
		assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
		assert.Equal(t, *cluster.Status.Patroni.SwitchoverTimeline, int64(4))
	})

	t.Run("switchover call errors", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(4)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, true, false
		err := r.reconcilePatroniSwitchover(ctx, cluster, getObserved())
		assert.Error(t, err, "boom")
		assert.Assert(t, called)
		assert.Assert(t, cluster.Status.Patroni.Switchover == nil)
	})

	t.Run("switchover called", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled: true,
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(4)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, false, false
		assert.NilError(t, r.reconcilePatroniSwitchover(ctx, cluster, getObserved()))
		assert.Assert(t, called)
		assert.Equal(t, *cluster.Status.Patroni.Switchover, "trigger")
		assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
	})

	t.Run("targeted switchover called", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled:        true,
				TargetInstance: initialize.String("target"),
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(4)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, false, false, false
		assert.NilError(t, r.reconcilePatroniSwitchover(ctx, cluster, getObserved()))
		assert.Assert(t, called)
		assert.Equal(t, *cluster.Status.Patroni.Switchover, "trigger")
		assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
	})

	t.Run("targeted failover called", func(t *testing.T) {
		cluster := testCluster()
		cluster.Annotations = map[string]string{
			naming.PatroniSwitchover: "trigger",
		}
		cluster.Spec.Patroni = &v1beta1.PatroniSpec{
			Switchover: &v1beta1.PatroniSwitchover{
				Enabled:        true,
				Type:           "Failover",
				TargetInstance: initialize.String("target"),
			},
		}
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{
			Name:                "target",
			Replicas:            initialize.Int32(2),
			DataVolumeClaimSpec: testVolumeClaimSpec(),
		}}
		cluster.Status.Patroni.SwitchoverTimeline = initialize.Int64(4)
		timelineCall, timelineCallNoLeader = true, false
		called, failover, callError, callFails = false, true, false, false
		assert.NilError(t, r.reconcilePatroniSwitchover(ctx, cluster, getObserved()))
		assert.Assert(t, called)
		assert.Equal(t, *cluster.Status.Patroni.Switchover, "trigger")
		assert.Assert(t, cluster.Status.Patroni.SwitchoverTimeline == nil)
	})
}
