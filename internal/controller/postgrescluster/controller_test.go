// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgrescluster

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/pkg/errors"

	"go.opentelemetry.io/otel"
	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/registration"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestDeleteControlled(t *testing.T) {
	ctx := context.Background()
	_, cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	ns := setupNamespace(t, cc)
	reconciler := Reconciler{Client: cc}

	cluster := testCluster()
	cluster.Namespace = ns.Name
	cluster.Name = strings.ToLower(t.Name())
	assert.NilError(t, cc.Create(ctx, cluster))

	t.Run("NoOwnership", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "solo"

		assert.NilError(t, cc.Create(ctx, secret))

		// No-op when there's no ownership
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))
		assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(secret), secret))
	})

	t.Run("Owned", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "owned"

		assert.NilError(t, reconciler.setOwnerReference(cluster, secret))
		assert.NilError(t, cc.Create(ctx, secret))

		// No-op when not controlled by cluster.
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))
		assert.NilError(t, cc.Get(ctx, client.ObjectKeyFromObject(secret), secret))
	})

	t.Run("Controlled", func(t *testing.T) {
		secret := &corev1.Secret{}
		secret.Namespace = ns.Name
		secret.Name = "controlled"

		assert.NilError(t, reconciler.setControllerReference(cluster, secret))
		assert.NilError(t, cc.Create(ctx, secret))

		// Deletes when controlled by cluster.
		assert.NilError(t, reconciler.deleteControlled(ctx, cluster, secret))

		err := cc.Get(ctx, client.ObjectKeyFromObject(secret), secret)
		assert.Assert(t, apierrors.IsNotFound(err), "expected NotFound, got %#v", err)
	})
}

var olmClusterYAML = `
metadata:
  name: olm
spec:
  postgresVersion: 13
  image: postgres
  instances:
  - name: register-now
    dataVolumeClaimSpec:
      accessModes:
      - "ReadWriteMany"
      resources:
        requests:
          storage: 1Gi
  backups:
    pgbackrest:
      image: pgbackrest
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
`

var _ = Describe("PostgresCluster Reconciler", func() {
	var test struct {
		Namespace  *corev1.Namespace
		Reconciler Reconciler
		Recorder   *record.FakeRecorder
	}

	BeforeEach(func() {
		ctx := context.Background()

		test.Namespace = &corev1.Namespace{}
		test.Namespace.Name = "postgres-operator-test-" + rand.String(6)
		Expect(suite.Client.Create(ctx, test.Namespace)).To(Succeed())

		test.Recorder = record.NewFakeRecorder(100)
		test.Recorder.IncludeObject = true

		test.Reconciler.Client = suite.Client
		test.Reconciler.Owner = "asdf"
		test.Reconciler.Recorder = test.Recorder
		test.Reconciler.Registration = nil
		test.Reconciler.Tracer = otel.Tracer("asdf")
	})

	AfterEach(func() {
		ctx := context.Background()

		if test.Namespace != nil {
			Expect(suite.Client.Delete(ctx, test.Namespace)).To(Succeed())
		}
	})

	create := func(clusterYAML string) *v1beta1.PostgresCluster {
		ctx := context.Background()

		var cluster v1beta1.PostgresCluster
		Expect(yaml.Unmarshal([]byte(clusterYAML), &cluster)).To(Succeed())

		cluster.Namespace = test.Namespace.Name
		Expect(suite.Client.Create(ctx, &cluster)).To(Succeed())

		return &cluster
	}

	reconcile := func(cluster *v1beta1.PostgresCluster) reconcile.Result {
		ctx := context.Background()

		result, err := test.Reconciler.Reconcile(ctx,
			reconcile.Request{NamespacedName: client.ObjectKeyFromObject(cluster)},
		)
		Expect(err).ToNot(HaveOccurred(), func() string {
			var t interface{ StackTrace() errors.StackTrace }
			if errors.As(err, &t) {
				return fmt.Sprintf("[partial] error trace:%+v\n", t.StackTrace()[:1])
			}
			return ""
		})

		return result
	}

	Context("Cluster with Registration Requirement, no token", func() {
		var cluster *v1beta1.PostgresCluster

		BeforeEach(func() {
			test.Reconciler.Registration = registration.RegistrationFunc(
				func(record.EventRecorder, client.Object, *[]metav1.Condition) bool {
					return true
				})

			cluster = create(olmClusterYAML)
			Expect(reconcile(cluster)).To(BeZero())
		})

		AfterEach(func() {
			ctx := context.Background()

			if cluster != nil {
				Expect(client.IgnoreNotFound(
					suite.Client.Delete(ctx, cluster),
				)).To(Succeed())

				// Remove finalizers, if any, so the namespace can terminate.
				Expect(client.IgnoreNotFound(
					suite.Client.Patch(ctx, cluster, client.RawPatch(
						client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`))),
				)).To(Succeed())
			}
		})

		Specify("Cluster RegistrationRequired Status", func() {
			existing := &v1beta1.PostgresCluster{}
			Expect(suite.Client.Get(
				context.Background(), client.ObjectKeyFromObject(cluster), existing,
			)).To(Succeed())

			Expect(meta.IsStatusConditionFalse(existing.Status.Conditions, v1beta1.Registered)).To(BeTrue())

			event, ok := <-test.Recorder.Events
			Expect(ok).To(BeTrue())
			Expect(event).To(ContainSubstring("Register Soon"))
		})
	})

	Context("Cluster", func() {
		var cluster *v1beta1.PostgresCluster

		BeforeEach(func() {
			cluster = create(`
metadata:
  name: carlos
spec:
  postgresVersion: 13
  image: postgres
  instances:
  - name: samba
    dataVolumeClaimSpec:
      accessModes:
      - "ReadWriteMany"
      resources:
        requests:
          storage: 1Gi
  backups:
    pgbackrest:
      image: pgbackrest
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
`)
			Expect(reconcile(cluster)).To(BeZero())
		})

		AfterEach(func() {
			ctx := context.Background()

			if cluster != nil {
				Expect(client.IgnoreNotFound(
					suite.Client.Delete(ctx, cluster),
				)).To(Succeed())

				// Remove finalizers, if any, so the namespace can terminate.
				Expect(client.IgnoreNotFound(
					suite.Client.Patch(ctx, cluster, client.RawPatch(
						client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`))),
				)).To(Succeed())
			}
		})

		Specify("Cluster ConfigMap", func() {
			ccm := &corev1.ConfigMap{}
			Expect(suite.Client.Get(context.Background(), client.ObjectKey{
				Namespace: test.Namespace.Name, Name: "carlos-config",
			}, ccm)).To(Succeed())

			Expect(ccm.Labels[naming.LabelCluster]).To(Equal("carlos"))
			Expect(ccm.OwnerReferences).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Controller": PointTo(BeTrue()),
					"Name":       Equal(cluster.Name),
					"UID":        Equal(cluster.UID),
				}),
			))
			Expect(ccm.ManagedFields).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Manager":   Equal(string(test.Reconciler.Owner)),
					"Operation": Equal(metav1.ManagedFieldsOperationApply),
				}),
			))

			Expect(ccm.Data["patroni.yaml"]).ToNot(BeZero())
		})

		Specify("Cluster Pod Service", func() {
			cps := &corev1.Service{}
			Expect(suite.Client.Get(context.Background(), client.ObjectKey{
				Namespace: test.Namespace.Name, Name: "carlos-pods",
			}, cps)).To(Succeed())

			Expect(cps.Labels[naming.LabelCluster]).To(Equal("carlos"))
			Expect(cps.OwnerReferences).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Controller": PointTo(BeTrue()),
					"Name":       Equal(cluster.Name),
					"UID":        Equal(cluster.UID),
				}),
			))
			Expect(cps.ManagedFields).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Manager":   Equal(string(test.Reconciler.Owner)),
					"Operation": Equal(metav1.ManagedFieldsOperationApply),
				}),
			))

			Expect(cps.Spec.ClusterIP).To(Equal("None"), "headless")
			Expect(cps.Spec.PublishNotReadyAddresses).To(BeTrue())
			Expect(cps.Spec.Selector).To(Equal(map[string]string{
				naming.LabelCluster: "carlos",
			}))
		})

		Specify("Cluster Status", func() {
			existing := &v1beta1.PostgresCluster{}
			Expect(suite.Client.Get(
				context.Background(), client.ObjectKeyFromObject(cluster), existing,
			)).To(Succeed())

			Expect(existing.Status.ObservedGeneration).To(Equal(cluster.Generation))

			// The interaction between server-side apply and subresources can have
			// unexpected results. However we manipulate Status, the controller must
			// only ever take ownership of the "status" field or fields within it--
			// never the "spec" field. Some known issues are:
			// - https://issue.k8s.io/75564
			// - https://issue.k8s.io/82046
			//
			// The "metadata.finalizers" field is also okay.
			// - https://book.kubebuilder.io/reference/using-finalizers.html
			//
			// NOTE(cbandy): Kubernetes prior to v1.16.10 and v1.17.6 does not track
			// managed fields on the status subresource: https://issue.k8s.io/88901
			switch {
			case suite.ServerVersion.LessThan(version.MustParseGeneric("1.22")):

				// Kubernetes 1.22 began tracking subresources in managed fields.
				// - https://pr.k8s.io/100970
				Expect(existing.ManagedFields).To(ContainElement(
					MatchFields(IgnoreExtras, Fields{
						"Manager": Equal(string(test.Reconciler.Owner)),
						"FieldsV1": PointTo(MatchAllFields(Fields{
							"Raw": WithTransform(func(in []byte) (out map[string]interface{}) {
								Expect(yaml.Unmarshal(in, &out)).To(Succeed())
								return out
							}, MatchAllKeys(Keys{
								"f:metadata": MatchAllKeys(Keys{
									"f:finalizers": Not(BeZero()),
								}),
								"f:status": Not(BeZero()),
							})),
						})),
					}),
				), `controller should manage only "finalizers" and "status"`)

			default:
				Expect(existing.ManagedFields).To(ContainElements(
					MatchFields(IgnoreExtras, Fields{
						"Manager": Equal(string(test.Reconciler.Owner)),
						"FieldsV1": PointTo(MatchAllFields(Fields{
							"Raw": WithTransform(func(in []byte) (out map[string]interface{}) {
								Expect(yaml.Unmarshal(in, &out)).To(Succeed())
								return out
							}, MatchAllKeys(Keys{
								"f:metadata": MatchAllKeys(Keys{
									"f:finalizers": Not(BeZero()),
								}),
							})),
						})),
					}),
					MatchFields(IgnoreExtras, Fields{
						"Manager": Equal(string(test.Reconciler.Owner)),
						"FieldsV1": PointTo(MatchAllFields(Fields{
							"Raw": WithTransform(func(in []byte) (out map[string]interface{}) {
								Expect(yaml.Unmarshal(in, &out)).To(Succeed())
								return out
							}, MatchAllKeys(Keys{
								"f:status": Not(BeZero()),
							})),
						})),
					}),
				), `controller should manage only "finalizers" and "status"`)
			}
		})

		Specify("Patroni Distributed Configuration", func() {
			ds := &corev1.Service{}
			Expect(suite.Client.Get(context.Background(), client.ObjectKey{
				Namespace: test.Namespace.Name, Name: "carlos-ha-config",
			}, ds)).To(Succeed())

			Expect(ds.Labels[naming.LabelCluster]).To(Equal("carlos"))
			Expect(ds.Labels[naming.LabelPatroni]).To(Equal("carlos-ha"))
			Expect(ds.OwnerReferences).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Controller": PointTo(BeTrue()),
					"Name":       Equal(cluster.Name),
					"UID":        Equal(cluster.UID),
				}),
			))
			Expect(ds.ManagedFields).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Manager":   Equal(string(test.Reconciler.Owner)),
					"Operation": Equal(metav1.ManagedFieldsOperationApply),
				}),
			))

			Expect(ds.Spec.ClusterIP).To(Equal("None"), "headless")
			Expect(ds.Spec.Selector).To(BeNil(), "no endpoints")
		})
	})

	Context("Instance", func() {
		var (
			cluster   *v1beta1.PostgresCluster
			instances appsv1.StatefulSetList
			instance  appsv1.StatefulSet
		)

		BeforeEach(func() {
			cluster = create(`
metadata:
  name: carlos
spec:
  postgresVersion: 13
  image: postgres
  instances:
  - name: samba
    dataVolumeClaimSpec:
      accessModes:
      - "ReadWriteMany"
      resources:
        requests:
          storage: 1Gi
  backups:
    pgbackrest:
      image: pgbackrest
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
`)
			Expect(reconcile(cluster)).To(BeZero())

			Expect(suite.Client.List(context.Background(), &instances,
				client.InNamespace(test.Namespace.Name),
				client.MatchingLabels{
					naming.LabelCluster:     "carlos",
					naming.LabelInstanceSet: "samba",
				},
			)).To(Succeed())
			Expect(instances.Items).To(HaveLen(1))

			instance = instances.Items[0]
		})

		AfterEach(func() {
			ctx := context.Background()

			if cluster != nil {
				Expect(client.IgnoreNotFound(
					suite.Client.Delete(ctx, cluster),
				)).To(Succeed())

				// Remove finalizers, if any, so the namespace can terminate.
				Expect(client.IgnoreNotFound(
					suite.Client.Patch(ctx, cluster, client.RawPatch(
						client.Merge.Type(), []byte(`{"metadata":{"finalizers":[]}}`))),
				)).To(Succeed())
			}
		})

		Specify("Instance ConfigMap", func() {
			icm := &corev1.ConfigMap{}
			Expect(suite.Client.Get(context.Background(), client.ObjectKey{
				Namespace: test.Namespace.Name, Name: instance.Name + "-config",
			}, icm)).To(Succeed())

			Expect(icm.Labels[naming.LabelCluster]).To(Equal("carlos"))
			Expect(icm.Labels[naming.LabelInstance]).To(Equal(instance.Name))
			Expect(icm.OwnerReferences).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Controller": PointTo(BeTrue()),
					"Name":       Equal(cluster.Name),
					"UID":        Equal(cluster.UID),
				}),
			))
			Expect(icm.ManagedFields).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Manager":   Equal(string(test.Reconciler.Owner)),
					"Operation": Equal(metav1.ManagedFieldsOperationApply),
				}),
			))

			Expect(icm.Data["patroni.yaml"]).ToNot(BeZero())
		})

		Specify("Instance StatefulSet", func() {
			Expect(instance.Labels[naming.LabelCluster]).To(Equal("carlos"))
			Expect(instance.Labels[naming.LabelInstanceSet]).To(Equal("samba"))
			Expect(instance.Labels[naming.LabelInstance]).To(Equal(instance.Name))
			Expect(instance.OwnerReferences).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Controller": PointTo(BeTrue()),
					"Name":       Equal(cluster.Name),
					"UID":        Equal(cluster.UID),
				}),
			))
			Expect(instance.ManagedFields).To(ContainElement(
				MatchFields(IgnoreExtras, Fields{
					"Manager":   Equal(string(test.Reconciler.Owner)),
					"Operation": Equal(metav1.ManagedFieldsOperationApply),
				}),
			))

			Expect(instance.Spec).To(MatchFields(IgnoreExtras, Fields{
				"PodManagementPolicy":  Equal(appsv1.OrderedReadyPodManagement),
				"Replicas":             PointTo(BeEquivalentTo(1)),
				"RevisionHistoryLimit": PointTo(BeEquivalentTo(0)),
				"ServiceName":          Equal("carlos-pods"),
				"UpdateStrategy": Equal(appsv1.StatefulSetUpdateStrategy{
					Type: appsv1.OnDeleteStatefulSetStrategyType,
				}),
			}))
		})

		It("resets Instance StatefulSet.Spec.Replicas", func() {
			ctx := context.Background()
			patch := client.MergeFrom(instance.DeepCopy())
			*instance.Spec.Replicas = 2

			Expect(suite.Client.Patch(ctx, &instance, patch)).To(Succeed())
			Expect(instance.Spec.Replicas).To(PointTo(BeEquivalentTo(2)))

			Expect(reconcile(cluster)).To(BeZero())
			Expect(suite.Client.Get(
				ctx, client.ObjectKeyFromObject(&instance), &instance,
			)).To(Succeed())
			Expect(instance.Spec.Replicas).To(PointTo(BeEquivalentTo(1)))
		})
	})
})
