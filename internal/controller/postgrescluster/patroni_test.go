//go:build envtest
// +build envtest

package postgrescluster

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

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/pkg/errors"
	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGeneratePatroniLeaderLeaseService(t *testing.T) {
	env, cc, _ := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	reconciler := &Reconciler{Client: cc}

	cluster := &v1beta1.PostgresCluster{}
	cluster.Namespace = "ns1"
	cluster.Name = "pg2"
	cluster.Spec.Port = initialize.Int32(9876)

	alwaysExpect := func(t testing.TB, service *corev1.Service) {
		assert.Assert(t, marshalMatches(service.TypeMeta, `
apiVersion: v1
kind: Service
		`))
		assert.Assert(t, marshalMatches(service.ObjectMeta, `
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
		assert.Assert(t, marshalMatches(service.Spec.Ports, `
- name: postgres
  port: 9876
  protocol: TCP
  targetPort: postgres
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
		})
	}
}

func TestReconcilePatroniLeaderLease(t *testing.T) {
	ctx := context.Background()
	env, cc, _ := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, env) })

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, cc.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, ns)) })

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
	for _, beforeType := range serviceTypes {
		for _, changeType := range serviceTypes {
			t.Run(beforeType+"To"+changeType, func(t *testing.T) {
				cluster := cluster.DeepCopy()
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
			})
		}
	}
}

func TestPatroniReplicationSecret(t *testing.T) {
	// Garbage collector cleans up test resources before the test completes
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("USE_EXISTING_CLUSTER: Test fails due to garbage collection")
	}

	// setup the test environment and ensure a clean teardown
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })

	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   tClient,
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	// test postgrescluster values
	var (
		clusterName = "hippocluster"
		clusterUID  = types.UID("hippouid")
	)

	ns := &corev1.Namespace{}
	ns.GenerateName = "postgres-operator-test-"
	ns.Labels = labels.Set{"postgres-operator-test": t.Name()}
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

	// create a PostgresCluster to test with
	postgresCluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      clusterName,
			Namespace: ns.Name,
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
	tEnv, tClient, cfg := setupTestEnv(t, ControllerName)
	t.Cleanup(func() { teardownTestEnv(t, tEnv) })
	r := &Reconciler{}
	ctx, cancel := setupManager(t, cfg, func(mgr manager.Manager) {
		r = &Reconciler{
			Client:   mgr.GetClient(),
			Recorder: mgr.GetEventRecorderFor(ControllerName),
			Tracer:   otel.Tracer(ControllerName),
			Owner:    ControllerName,
		}
	})
	t.Cleanup(func() { teardownManager(cancel, t) })

	namespace := "test-reconcile-patroni-status"
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
				Namespace: namespace,
			},
		}

		runner := &appsv1.StatefulSet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
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

	ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}}
	assert.NilError(t, tClient.Create(ctx, ns))
	t.Cleanup(func() { assert.Check(t, tClient.Delete(ctx, ns)) })

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
			result, err := r.reconcilePatroniStatus(ctx, postgresCluster, observedInstances)
			if tc.requeueExpected {
				assert.NilError(t, err)
				assert.Assert(t, result.RequeueAfter == 1*time.Second)
			} else {
				assert.NilError(t, err)
				assert.DeepEqual(t, result, reconcile.Result{})
			}
		})
	}
}
