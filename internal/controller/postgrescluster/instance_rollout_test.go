package postgrescluster

import (
	"context"
	"encoding/json"
	"testing"

	"go.opentelemetry.io/otel/oteltest"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilerRolloutInstances(t *testing.T) {
	ctx := context.Background()
	reconciler := &Reconciler{Tracer: oteltest.DefaultTracer()}

	accumulate := func(on *[]*Instance) func(context.Context, *Instance) error {
		return func(_ context.Context, i *Instance) error { *on = append(*on, i); return nil }
	}

	logSpanAttributes := func(t testing.TB) {
		recorder := new(oteltest.StandardSpanRecorder)
		provider := oteltest.NewTracerProvider(oteltest.WithSpanRecorder(recorder))

		former := reconciler.Tracer
		reconciler.Tracer = provider.Tracer("")

		t.Cleanup(func() {
			reconciler.Tracer = former
			for _, span := range recorder.Completed() {
				b, _ := json.Marshal(span.Attributes())
				t.Log(span.Name(), string(b))
			}
		})
	}

	// Nothing specified, nothing observed, nothing to do.
	t.Run("Empty", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		observed := new(observedInstances)

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed,
			func(context.Context, *Instance) error {
				t.Fatal("expected no redeploys")
				return nil
			}))
	})

	// Single healthy instance; nothing to do.
	t.Run("Steady", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(1)},
		}
		instances := []*Instance{
			{
				Name: "one",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash":               "gamma",
							"postgres-operator.crunchydata.com/role": "master",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed,
			func(context.Context, *Instance) error {
				t.Fatal("expected no redeploys")
				return nil
			}))
	})

	// Single healthy instance, Pod does not match PodTemplate.
	t.Run("SingletonOutdated", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(1)},
		}
		instances := []*Instance{
			{
				Name: "one",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash":               "beta",
							"postgres-operator.crunchydata.com/role": "master",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		var redeploys []*Instance

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed, accumulate(&redeploys)))
		assert.Equal(t, len(redeploys), 1)
		assert.Equal(t, redeploys[0].Name, "one")
	})

	// Two ready instances do not match PodTemplate, no primary.
	t.Run("ManyOutdated", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(2)},
		}
		instances := []*Instance{
			{
				Name: "one",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
			{
				Name: "two",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		var redeploys []*Instance

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed, accumulate(&redeploys)))
		assert.Equal(t, len(redeploys), 1)
		assert.Equal(t, redeploys[0].Name, "one", `expected the "lowest" name`)
	})

	// Two ready instances do not match PodTemplate, with primary. The replica is redeployed.
	t.Run("ManyOutdatedWithPrimary", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(2)},
		}
		instances := []*Instance{
			{
				Name: "outdated",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash":               "beta",
							"postgres-operator.crunchydata.com/role": "master",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
			{
				Name: "not-primary",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		var redeploys []*Instance

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed, accumulate(&redeploys)))
		assert.Equal(t, len(redeploys), 1)
		assert.Equal(t, redeploys[0].Name, "not-primary")
	})

	// Two instances do not match PodTemplate, one is not ready. Redeploy that one.
	t.Run("ManyOutdatedWithNotReady", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(2)},
		}
		instances := []*Instance{
			{
				Name: "outdated",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
			{
				Name: "not-ready",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionFalse,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		var redeploys []*Instance

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed, accumulate(&redeploys)))
		assert.Equal(t, len(redeploys), 1)
		assert.Equal(t, redeploys[0].Name, "not-ready")
	})

	// Two instances do not match PodTemplate, one is terminating. Do nothing.
	t.Run("ManyOutdatedWithTerminating", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(2)},
		}
		instances := []*Instance{
			{
				Name: "outdated",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
			{
				Name: "terminating",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						DeletionTimestamp: new(metav1.Time),
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed,
			func(context.Context, *Instance) error {
				t.Fatal("expected no redeploys")
				return nil
			}))
	})

	// Two instances do not match PodTemplate, one is orphaned. Do nothing.
	t.Run("ManyOutdatedWithOrphan", func(t *testing.T) {
		cluster := new(v1beta1.PostgresCluster)
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{
			{Name: "00", Replicas: initialize.Int32(2)},
		}
		instances := []*Instance{
			{
				Name: "outdated",
				Spec: &cluster.Spec.InstanceSets[0],
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
			{
				Name: "orphan",
				Pods: []*corev1.Pod{{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"controller-revision-hash": "beta",
						},
					},
					Status: corev1.PodStatus{
						Conditions: []corev1.PodCondition{{
							Type:   corev1.PodReady,
							Status: corev1.ConditionTrue,
						}},
					},
				}},
				Runner: &appsv1.StatefulSet{
					ObjectMeta: metav1.ObjectMeta{
						Generation: 1,
					},
					Status: appsv1.StatefulSetStatus{
						ObservedGeneration: 1,
						UpdateRevision:     "gamma",
					},
				},
			},
		}
		observed := &observedInstances{forCluster: instances}

		logSpanAttributes(t)
		assert.NilError(t, reconciler.rolloutInstances(ctx, cluster, observed,
			func(context.Context, *Instance) error {
				t.Fatal("expected no redeploys")
				return nil
			}))
	})
}
