// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestReconcilePGAdminStatefulSet(t *testing.T) {
	ctx := context.Background()
	cc := setupKubernetes(t)
	require.ParallelCapacity(t, 1)

	reconciler := &PGAdminReconciler{
		Client: cc,
		Owner:  client.FieldOwner(t.Name()),
	}

	ns := setupNamespace(t, cc)
	pgadmin := new(v1beta1.PGAdmin)
	pgadmin.Name = "test-standalone-pgadmin"
	pgadmin.Namespace = ns.Name

	assert.NilError(t, cc.Create(ctx, pgadmin))
	t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, pgadmin)) })

	configmap := &corev1.ConfigMap{}
	configmap.Name = "test-cm"

	pvc := &corev1.PersistentVolumeClaim{}
	pvc.Name = "test-pvc"

	t.Run("verify StatefulSet", func(t *testing.T) {
		err := reconciler.reconcilePGAdminStatefulSet(ctx, pgadmin, configmap, pvc)
		assert.NilError(t, err)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelStandalonePGAdmin: pgadmin.Name,
			},
		})
		assert.NilError(t, err)

		list := appsv1.StatefulSetList{}
		assert.NilError(t, cc.List(ctx, &list, client.InNamespace(pgadmin.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		assert.Equal(t, len(list.Items), 1)

		template := list.Items[0].Spec.Template.DeepCopy()

		// Containers and Volumes should be populated.
		assert.Assert(t, len(template.Spec.Containers) != 0)
		assert.Assert(t, len(template.Spec.Volumes) != 0)

		// Ignore Containers and Volumes in the comparison below.
		template.Spec.Containers = nil
		template.Spec.InitContainers = nil
		template.Spec.Volumes = nil

		assert.Assert(t, cmp.MarshalMatches(template.ObjectMeta, `
creationTimestamp: null
labels:
  postgres-operator.crunchydata.com/data: pgadmin
  postgres-operator.crunchydata.com/pgadmin: test-standalone-pgadmin
  postgres-operator.crunchydata.com/role: pgadmin
		`))

		compare := `
automountServiceAccountToken: false
containers: null
dnsPolicy: ClusterFirst
enableServiceLinks: false
restartPolicy: Always
schedulerName: default-scheduler
securityContext:
  fsGroup: 2
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
		`

		assert.Assert(t, cmp.MarshalMatches(template.Spec, compare))
	})

	t.Run("verify customized deployment", func(t *testing.T) {

		custompgadmin := new(v1beta1.PGAdmin)

		// add pod level customizations
		custompgadmin.Name = "custom-pgadmin"
		custompgadmin.Namespace = ns.Name

		// annotation and label
		custompgadmin.Spec.Metadata = &v1beta1.Metadata{
			Annotations: map[string]string{
				"annotation1": "annotationvalue",
			},
			Labels: map[string]string{
				"label1": "labelvalue",
			},
		}

		// scheduling constraints
		custompgadmin.Spec.Affinity = &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
					NodeSelectorTerms: []corev1.NodeSelectorTerm{{
						MatchExpressions: []corev1.NodeSelectorRequirement{{
							Key:      "key",
							Operator: "Exists",
						}},
					}},
				},
			},
		}
		custompgadmin.Spec.Tolerations = []corev1.Toleration{
			{Key: "sometoleration"},
		}

		if pgadmin.Spec.PriorityClassName != nil {
			custompgadmin.Spec.PriorityClassName = initialize.String("testpriorityclass")
		}

		// set an image pull secret
		custompgadmin.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{
			Name: "myImagePullSecret"}}

		assert.NilError(t, cc.Create(ctx, custompgadmin))
		t.Cleanup(func() { assert.Check(t, cc.Delete(ctx, custompgadmin)) })

		err := reconciler.reconcilePGAdminStatefulSet(ctx, custompgadmin, configmap, pvc)
		assert.NilError(t, err)

		selector, err := naming.AsSelector(metav1.LabelSelector{
			MatchLabels: map[string]string{
				naming.LabelStandalonePGAdmin: custompgadmin.Name,
			},
		})
		assert.NilError(t, err)

		list := appsv1.StatefulSetList{}
		assert.NilError(t, cc.List(ctx, &list, client.InNamespace(custompgadmin.Namespace),
			client.MatchingLabelsSelector{Selector: selector}))
		assert.Equal(t, len(list.Items), 1)

		template := list.Items[0].Spec.Template.DeepCopy()

		// Containers and Volumes should be populated.
		assert.Assert(t, len(template.Spec.Containers) != 0)

		// Ignore Containers and Volumes in the comparison below.
		template.Spec.Containers = nil
		template.Spec.InitContainers = nil
		template.Spec.Volumes = nil

		assert.Assert(t, cmp.MarshalMatches(template.ObjectMeta, `
annotations:
  annotation1: annotationvalue
creationTimestamp: null
labels:
  label1: labelvalue
  postgres-operator.crunchydata.com/data: pgadmin
  postgres-operator.crunchydata.com/pgadmin: custom-pgadmin
  postgres-operator.crunchydata.com/role: pgadmin
		`))

		compare := `
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
      - matchExpressions:
        - key: key
          operator: Exists
automountServiceAccountToken: false
containers: null
dnsPolicy: ClusterFirst
enableServiceLinks: false
imagePullSecrets:
- name: myImagePullSecret
restartPolicy: Always
schedulerName: default-scheduler
securityContext:
  fsGroup: 2
  fsGroupChangePolicy: OnRootMismatch
terminationGracePeriodSeconds: 30
tolerations:
- key: sometoleration
`

		assert.Assert(t, cmp.MarshalMatches(template.Spec, compare))
	})
}
