// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/feature"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func testExporterCollectorsAnnotation(t *testing.T, ctx context.Context, cluster *v1beta1.PostgresCluster, queriesConfig, webConfig *corev1.ConfigMap) {
	t.Helper()

	t.Run("ExporterCollectorsAnnotation", func(t *testing.T) {
		t.Run("UnexpectedValue", func(t *testing.T) {
			template := new(corev1.PodTemplateSpec)
			cluster := cluster.DeepCopy()
			cluster.SetAnnotations(map[string]string{
				naming.PostgresExporterCollectorsAnnotation: "wrong-value",
			})

			assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, queriesConfig, webConfig))

			assert.Equal(t, len(template.Spec.Containers), 1)
			container := template.Spec.Containers[0]

			command := strings.Join(container.Command, "\n")
			assert.Assert(t, cmp.Contains(command, "postgres_exporter"))
			assert.Assert(t, !strings.Contains(command, "collector"))
		})

		t.Run("ExpectedValueNone", func(t *testing.T) {
			template := new(corev1.PodTemplateSpec)
			cluster := cluster.DeepCopy()
			cluster.SetAnnotations(map[string]string{
				naming.PostgresExporterCollectorsAnnotation: "None",
			})

			assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, queriesConfig, webConfig))

			assert.Equal(t, len(template.Spec.Containers), 1)
			container := template.Spec.Containers[0]

			command := strings.Join(container.Command, "\n")
			assert.Assert(t, cmp.Contains(command, "postgres_exporter"))
			assert.Assert(t, cmp.Contains(command, "--[no-]collector"))

			t.Run("LowercaseToo", func(t *testing.T) {
				template := new(corev1.PodTemplateSpec)
				cluster.SetAnnotations(map[string]string{
					naming.PostgresExporterCollectorsAnnotation: "none",
				})

				assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, queriesConfig, webConfig))
				assert.Assert(t, cmp.Contains(strings.Join(template.Spec.Containers[0].Command, "\n"), "--[no-]collector"))
			})
		})
	})
}

func TestAddPGMonitorExporterToInstancePodSpec(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	image := "test/image:tag"

	cluster := &v1beta1.PostgresCluster{}
	cluster.Name = "pg1"
	cluster.Spec.Port = initialize.Int32(5432)
	cluster.Spec.ImagePullPolicy = corev1.PullAlways

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("100m"),
		},
	}

	exporterQueriesConfig := new(corev1.ConfigMap)
	exporterQueriesConfig.Name = "query-conf"

	t.Run("ExporterDisabled", func(t *testing.T) {
		template := &corev1.PodTemplateSpec{}
		assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, nil, nil))
		assert.DeepEqual(t, template, &corev1.PodTemplateSpec{})
	})

	t.Run("ExporterEnabled", func(t *testing.T) {
		cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image:     image,
					Resources: resources,
				},
			},
		}
		template := &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: naming.ContainerDatabase,
				}},
			},
		}

		assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, exporterQueriesConfig, nil))

		assert.Equal(t, len(template.Spec.Containers), 2)
		container := template.Spec.Containers[1]

		command := strings.Join(container.Command, "\n")
		assert.Assert(t, cmp.Contains(command, "postgres_exporter"))
		assert.Assert(t, cmp.Contains(command, "--extend.query-path"))
		assert.Assert(t, cmp.Contains(command, "--web.listen-address"))

		// Exclude command from the following comparison.
		container.Command = nil
		assert.Assert(t, cmp.MarshalMatches(container, `
env:
- name: DATA_SOURCE_URI
  value: localhost:5432/postgres
- name: DATA_SOURCE_USER
  value: ccp_monitoring
- name: DATA_SOURCE_PASS_FILE
  value: /opt/crunchy/password
image: test/image:tag
imagePullPolicy: Always
name: exporter
ports:
- containerPort: 9187
  name: exporter
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
- mountPath: /conf
  name: exporter-config
- mountPath: /opt/crunchy/
  name: monitoring-secret
		`))

		assert.Assert(t, cmp.MarshalMatches(template.Spec.Volumes, `
- name: exporter-config
  projected:
    sources:
    - configMap:
        name: query-conf
- name: monitoring-secret
  secret:
    secretName: pg1-monitoring
		`))

		testExporterCollectorsAnnotation(t, ctx, cluster, exporterQueriesConfig, nil)
	})

	t.Run("CustomConfigAppendCustomQueriesOff", func(t *testing.T) {
		cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image:     image,
					Resources: resources,
					Configuration: []corev1.VolumeProjection{{ConfigMap: &corev1.ConfigMapProjection{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "exporter-custom-config-test",
						},
					}},
					},
				},
			},
		}
		template := &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: naming.ContainerDatabase,
				}},
			},
		}

		assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, exporterQueriesConfig, nil))

		assert.Equal(t, len(template.Spec.Containers), 2)
		container := template.Spec.Containers[1]

		assert.Assert(t, len(template.Spec.Volumes) > 0)
		assert.Assert(t, cmp.MarshalMatches(template.Spec.Volumes[0], `
name: exporter-config
projected:
  sources:
  - configMap:
      name: exporter-custom-config-test
		`))

		assert.Assert(t, len(container.VolumeMounts) > 0)
		assert.Assert(t, cmp.MarshalMatches(container.VolumeMounts[0], `
mountPath: /conf
name: exporter-config
		`))
	})

	t.Run("CustomConfigAppendCustomQueriesOn", func(t *testing.T) {
		gate := feature.NewGate()
		assert.NilError(t, gate.SetFromMap(map[string]bool{
			feature.AppendCustomQueries: true,
		}))
		ctx := feature.NewContext(ctx, gate)

		cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image:     image,
					Resources: resources,
					Configuration: []corev1.VolumeProjection{{ConfigMap: &corev1.ConfigMapProjection{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "exporter-custom-config-test",
						},
					}},
					},
				},
			},
		}
		template := &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: naming.ContainerDatabase,
				}},
			},
		}

		assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, exporterQueriesConfig, nil))

		assert.Equal(t, len(template.Spec.Containers), 2)
		container := template.Spec.Containers[1]

		assert.Assert(t, len(template.Spec.Volumes) > 0)
		assert.Assert(t, cmp.MarshalMatches(template.Spec.Volumes[0], `
name: exporter-config
projected:
  sources:
  - configMap:
      name: exporter-custom-config-test
  - configMap:
      name: query-conf
		`))

		assert.Assert(t, len(container.VolumeMounts) > 0)
		assert.Assert(t, cmp.MarshalMatches(container.VolumeMounts[0], `
mountPath: /conf
name: exporter-config
		`))
	})

	t.Run("CustomTLS", func(t *testing.T) {
		cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					CustomTLSSecret: &corev1.SecretProjection{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "custom-exporter-certs",
						},
					},
				},
			},
		}
		template := &corev1.PodTemplateSpec{
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{
					Name: naming.ContainerDatabase,
				}},
			},
		}

		testConfigMap := new(corev1.ConfigMap)
		testConfigMap.Name = "test-web-conf"

		assert.NilError(t, addPGMonitorExporterToInstancePodSpec(ctx, cluster, template, exporterQueriesConfig, testConfigMap))

		assert.Equal(t, len(template.Spec.Containers), 2)
		container := template.Spec.Containers[1]

		assert.Assert(t, len(template.Spec.Volumes) > 2, "Expected the original two volumes")
		assert.Assert(t, cmp.MarshalMatches(template.Spec.Volumes[2:], `
- name: exporter-certs
  projected:
    sources:
    - secret:
        name: custom-exporter-certs
- configMap:
    name: test-web-conf
  name: web-config
		`))

		assert.Assert(t, len(container.VolumeMounts) > 2, "Expected the original two mounts")
		assert.Assert(t, cmp.MarshalMatches(container.VolumeMounts[2:], `
- mountPath: /certs
  name: exporter-certs
- mountPath: /web-config
  name: web-config
		`))

		command := strings.Join(container.Command, "\n")
		assert.Assert(t, cmp.Contains(command, "postgres_exporter"))
		assert.Assert(t, cmp.Contains(command, "--web.config.file"))

		testExporterCollectorsAnnotation(t, ctx, cluster, exporterQueriesConfig, testConfigMap)
	})
}

// TestReconcilePGMonitorExporterSetupErrors tests how reconcilePGMonitorExporter
// reacts when the kubernetes resources are in different states (e.g., checks
// what happens when the database pod is terminating)
func TestReconcilePGMonitorExporterSetupErrors(t *testing.T) {
	if os.Getenv("QUERIES_CONFIG_DIR") == "" {
		t.Skip("QUERIES_CONFIG_DIR must be set")
	}

	for _, test := range []struct {
		name          string
		podExecCalled bool
		status        v1beta1.MonitoringStatus
		monitoring    *v1beta1.MonitoringSpec
		instances     []*Instance
		secret        *corev1.Secret
	}{{
		name:          "Terminating",
		podExecCalled: false,
		instances: []*Instance{
			{
				Name: "daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name:              "daisy-pod",
						Annotations:       map[string]string{"status": `{"role":"master"}`},
						DeletionTimestamp: &metav1.Time{},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		},
	}, {
		name:          "NotWritable",
		podExecCalled: false,
		instances: []*Instance{
			{
				Name: "daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name: "daisy-pod",
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		},
	}, {
		name:          "NotRunning",
		podExecCalled: false,
		instances: []*Instance{
			{
				Name: "daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "daisy-pod",
						Annotations: map[string]string{"status": `{"role":"master"}`},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		},
	}, {
		name:          "ExporterNotRunning",
		podExecCalled: false,
		monitoring: &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		},
		instances: []*Instance{
			{
				Name: "daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "daisy-pod",
						Annotations: map[string]string{"status": `{"role":"master"}`},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name:  naming.ContainerDatabase,
							State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		},
	}, {
		name:          "ExporterImageIDNotFound",
		podExecCalled: false,
		monitoring: &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		},
		instances: []*Instance{
			{
				Name: "daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "daisy-pod",
						Annotations: map[string]string{"status": `{"role":"master"}`},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name:  naming.ContainerDatabase,
							State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						}, {
							Name:  naming.ContainerPGMonitorExporter,
							State: corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		},
	}, {
		name:          "NoError",
		podExecCalled: true,
		monitoring: &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		},
		instances: []*Instance{
			{
				Name: "daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "daisy-pod",
						Annotations: map[string]string{"status": `{"role":"master"}`},
					},
					Status: corev1.PodStatus{
						ContainerStatuses: []corev1.ContainerStatus{{
							Name:    naming.ContainerDatabase,
							State:   corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
							ImageID: "image@sha123",
						}, {
							Name:    naming.ContainerPGMonitorExporter,
							State:   corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
							ImageID: "image@sha123",
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		},
		secret: &corev1.Secret{
			Data: map[string][]byte{
				"verifier": []byte("blah"),
			},
		},
	}} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			var called bool
			reconciler := &Reconciler{
				PodExec: func(ctx context.Context, namespace, pod, container string, stdin io.Reader,
					stdout, stderr io.Writer, command ...string) error {
					called = true
					return nil
				},
			}

			cluster := &v1beta1.PostgresCluster{}
			cluster.Spec.PostgresVersion = 15
			cluster.Spec.Monitoring = test.monitoring
			cluster.Status.Monitoring.ExporterConfiguration = test.status.ExporterConfiguration
			observed := &observedInstances{forCluster: test.instances}

			assert.NilError(t, reconciler.reconcilePGMonitorExporter(ctx,
				cluster, observed, test.secret))
			assert.Equal(t, called, test.podExecCalled)
		})
	}
}

func TestReconcilePGMonitorExporter(t *testing.T) {
	ctx := context.Background()
	var called bool
	reconciler := &Reconciler{
		PodExec: func(ctx context.Context, namespace, pod, container string, stdin io.Reader,
			stdout, stderr io.Writer, command ...string) error {
			called = true
			return nil
		},
	}

	t.Run("UninstallWhenSecretNil", func(t *testing.T) {
		cluster := &v1beta1.PostgresCluster{}
		cluster.Status.Monitoring.ExporterConfiguration = "installed"
		instances := []*Instance{
			{
				Name: "one-daisy",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Name:        "one-daisy-pod",
						Annotations: map[string]string{"status": `{"role":"master"}`},
					},
					Status: corev1.PodStatus{
						Phase: corev1.PodRunning,
						ContainerStatuses: []corev1.ContainerStatus{{
							Name:    naming.ContainerDatabase,
							ImageID: "dont-care",
							State: corev1.ContainerState{
								Running: &corev1.ContainerStateRunning{},
							},
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{},
			},
		}
		observed := &observedInstances{forCluster: instances}

		called = false
		assert.NilError(t, reconciler.reconcilePGMonitorExporter(ctx,
			cluster, observed, nil))
		assert.Assert(t, called, "PodExec was not called.")
		assert.Assert(t, cluster.Status.Monitoring.ExporterConfiguration != "", "ExporterConfiguration was empty.")
	})
}

// TestReconcilePGMonitorExporterStatus checks that the exporter status is updated
// when it should be. Because the status updated when we update the setup sql from
// pgmonitor (by using podExec), we check if podExec is called when a change is needed.
func TestReconcilePGMonitorExporterStatus(t *testing.T) {
	if os.Getenv("QUERIES_CONFIG_DIR") == "" {
		t.Skip("QUERIES_CONFIG_DIR must be set")
	}

	for _, test := range []struct {
		name                        string
		exporterEnabled             bool
		podExecCalled               bool
		status                      v1beta1.MonitoringStatus
		statusChangedAfterReconcile bool
	}{{
		name:                        "Disabled",
		podExecCalled:               true,
		statusChangedAfterReconcile: true,
	}, {
		name:                        "Disabled Uninstall",
		podExecCalled:               true,
		status:                      v1beta1.MonitoringStatus{ExporterConfiguration: "installed"},
		statusChangedAfterReconcile: true,
	}, {
		name:                        "Enabled",
		exporterEnabled:             true,
		podExecCalled:               true,
		statusChangedAfterReconcile: true,
	}, {
		name:                        "Enabled Update",
		exporterEnabled:             true,
		podExecCalled:               true,
		status:                      v1beta1.MonitoringStatus{ExporterConfiguration: "installed"},
		statusChangedAfterReconcile: true,
	}, {
		name:            "Enabled NoUpdate",
		exporterEnabled: true,
		podExecCalled:   false,
		// Status was generated manually for this test case
		// TODO (jmckulk): add code to generate status
		status:                      v1beta1.MonitoringStatus{ExporterConfiguration: "7cdb484b6c"},
		statusChangedAfterReconcile: false,
	}} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			var (
				called bool
				secret *corev1.Secret
			)

			// Create reconciler with mock PodExec function
			reconciler := &Reconciler{
				PodExec: func(ctx context.Context, namespace, pod, container string, stdin io.Reader,
					stdout, stderr io.Writer, command ...string) error {
					called = true
					return nil
				},
			}

			// Create the test cluster spec with the exporter status set
			cluster := &v1beta1.PostgresCluster{}
			cluster.Spec.PostgresVersion = 15
			cluster.Status.Monitoring.ExporterConfiguration = test.status.ExporterConfiguration

			// Mock up an instances that will be defined in the cluster. The instances should
			// have all necessary fields that will be needed to reconcile the exporter
			instances := []*Instance{
				{
					Name: "daisy",
					Pods: []*corev1.Pod{{
						ObjectMeta: metav1.ObjectMeta{
							Name:        "daisy-pod",
							Annotations: map[string]string{"status": `{"role":"master"}`},
						},
						Status: corev1.PodStatus{
							ContainerStatuses: []corev1.ContainerStatus{{
								Name:    naming.ContainerDatabase,
								State:   corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
								ImageID: "image@sha123",
							}},
						},
					}},
					Runner: &appsv1.StatefulSet{},
				},
			}

			if test.exporterEnabled {
				// When testing with exporter enabled update the spec with exporter fields
				cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
					PGMonitor: &v1beta1.PGMonitorSpec{
						Exporter: &v1beta1.ExporterSpec{
							Image: "image",
						},
					},
				}

				// Update mock instances to include the exporter container
				instances[0].Pods[0].Status.ContainerStatuses = append(
					instances[0].Pods[0].Status.ContainerStatuses, corev1.ContainerStatus{
						Name:    naming.ContainerPGMonitorExporter,
						State:   corev1.ContainerState{Running: &corev1.ContainerStateRunning{}},
						ImageID: "image@sha123",
					})

				secret = &corev1.Secret{
					Data: map[string][]byte{
						"verifier": []byte("blah"),
					},
				}
			}

			// Mock up observed instances based on our mock instances
			observed := &observedInstances{forCluster: instances}

			// Check that we can reconcile with the test resources
			assert.NilError(t, reconciler.reconcilePGMonitorExporter(ctx,
				cluster, observed, secret))
			// Check that the exporter status changes when it needs to
			assert.Assert(t, test.statusChangedAfterReconcile == (cluster.Status.Monitoring.ExporterConfiguration != test.status.ExporterConfiguration),
				"got %v", cluster.Status.Monitoring.ExporterConfiguration)
			// Check that pod exec is called correctly
			assert.Equal(t, called, test.podExecCalled)
		})
	}
}

// TestReconcileMonitoringSecret checks that the secret intent returned by reconcileMonitoringSecret
// is correct. If exporter is enabled, the return shouldn't be nil. If the exporter is disabled, the
// return should be nil.
func TestReconcileMonitoringSecret(t *testing.T) {
	// TODO (jmckulk): debug test with existing cluster
	// Seems to be an issue when running with other tests
	if strings.EqualFold(os.Getenv("USE_EXISTING_CLUSTER"), "true") {
		t.Skip("Test failing with existing cluster")
	}

	ctx := context.Background()

	// Kubernetes is required because reconcileMonitoringSecret
	// (1) uses the client to get existing secrets
	// (2) sets the controller reference on the new secret
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := testCluster()
	cluster.Default()
	cluster.UID = types.UID("hippouid")
	cluster.Namespace = setupNamespace(t, cc).Name

	// If the exporter is disabled then the secret should not exist
	// Existing secrets should be removed
	t.Run("ExporterDisabled", func(t *testing.T) {
		t.Run("NotExisting", func(t *testing.T) {
			secret, err := reconciler.reconcileMonitoringSecret(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, secret == nil, "Monitoring secret was not nil.")
		})

		t.Run("Existing", func(t *testing.T) {
			cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
				PGMonitor: &v1beta1.PGMonitorSpec{
					Exporter: &v1beta1.ExporterSpec{Image: "image"}}}
			existing, err := reconciler.reconcileMonitoringSecret(ctx, cluster)
			assert.NilError(t, err, "error in test; existing secret not created")
			assert.Assert(t, existing != nil, "error in test; existing secret not created")

			cluster.Spec.Monitoring = nil
			actual, err := reconciler.reconcileMonitoringSecret(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, actual == nil, "Monitoring secret still exists after turning exporter off.")
		})
	})

	// If the exporter is enabled then a monitoring secret should exist
	// It will need to be created or left in place with existing password
	t.Run("ExporterEnabled", func(t *testing.T) {
		var (
			existing, actual *corev1.Secret
			err              error
		)

		// Enable monitoring in the test cluster spec
		cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		}

		t.Run("NotExisting", func(t *testing.T) {
			existing, err = reconciler.reconcileMonitoringSecret(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, existing != nil, "Monitoring secret does not exist.")
		})

		t.Run("Existing", func(t *testing.T) {
			actual, err = reconciler.reconcileMonitoringSecret(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, bytes.Equal(actual.Data["password"], existing.Data["password"]), "Passwords do not match.")
		})
	})
}

// TestReconcileExporterQueriesConfig checks that the ConfigMap intent returned by
// reconcileExporterQueriesConfig is correct. If exporter is enabled, the return
// shouldn't be nil. If the exporter is disabled, the return should be nil.
func TestReconcileExporterQueriesConfig(t *testing.T) {
	ctx := context.Background()

	// Kubernetes is required because reconcileExporterQueriesConfig
	// (1) uses the client to get existing ConfigMaps
	// (2) sets the controller reference on the new ConfigMap
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 0)

	reconciler := &Reconciler{Client: cc, Owner: client.FieldOwner(t.Name())}

	cluster := testCluster()
	cluster.Default()
	cluster.UID = types.UID("hippouid")
	cluster.Namespace = setupNamespace(t, cc).Name

	t.Run("ExporterDisabled", func(t *testing.T) {
		t.Run("NotExisting", func(t *testing.T) {
			queriesConfig, err := reconciler.reconcileExporterQueriesConfig(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, queriesConfig == nil, "Default queries ConfigMap is present.")
		})

		t.Run("Existing", func(t *testing.T) {
			cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
				PGMonitor: &v1beta1.PGMonitorSpec{
					Exporter: &v1beta1.ExporterSpec{Image: "image"}}}
			existing, err := reconciler.reconcileExporterQueriesConfig(ctx, cluster)
			assert.NilError(t, err, "error in test; existing config not created")
			assert.Assert(t, existing != nil, "error in test; existing config not created")

			cluster.Spec.Monitoring = nil
			actual, err := reconciler.reconcileExporterQueriesConfig(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, actual == nil, "Default queries config still present after disabling exporter.")
		})
	})

	t.Run("ExporterEnabled", func(t *testing.T) {
		var (
			existing, actual *corev1.ConfigMap
			err              error
		)

		// Enable monitoring in the test cluster spec
		cluster.Spec.Monitoring = &v1beta1.MonitoringSpec{
			PGMonitor: &v1beta1.PGMonitorSpec{
				Exporter: &v1beta1.ExporterSpec{
					Image: "image",
				},
			},
		}

		t.Run("NotExisting", func(t *testing.T) {
			existing, err = reconciler.reconcileExporterQueriesConfig(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, existing != nil, "Default queries config does not exist.")
		})

		t.Run("Existing", func(t *testing.T) {
			actual, err = reconciler.reconcileExporterQueriesConfig(ctx, cluster)
			assert.NilError(t, err)
			assert.Assert(t, actual.Data["defaultQueries.yml"] == existing.Data["defaultQueries.yml"], "Data does not align.")
		})
	})
}
