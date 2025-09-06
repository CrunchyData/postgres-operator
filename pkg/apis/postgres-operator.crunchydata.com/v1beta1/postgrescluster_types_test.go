// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1_test

import (
	"reflect"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgresClusterDefault(t *testing.T) {
	t.Run("TypeMeta", func(t *testing.T) {
		var cluster v1beta1.PostgresCluster
		cluster.Default()

		assert.Equal(t, cluster.APIVersion, v1beta1.GroupVersion.String())
		assert.Equal(t, cluster.Kind, reflect.TypeOf(cluster).Name())
	})

	t.Run("no instance sets", func(t *testing.T) {
		var cluster v1beta1.PostgresCluster
		cluster.Default()

		assert.Assert(t, MarshalsTo(cluster, `
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
spec:
  instances: null
  patroni:
    leaderLeaseDurationSeconds: 30
    port: 8008
    syncPeriodSeconds: 10
  port: 5432
  postgresVersion: 0
		`))
	})

	t.Run("one instance set", func(t *testing.T) {
		var cluster v1beta1.PostgresCluster
		cluster.Spec.InstanceSets = []v1beta1.PostgresInstanceSetSpec{{}}
		cluster.Default()

		assert.Assert(t, MarshalsTo(cluster, `
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
spec:
  instances:
  - dataVolumeClaimSpec:
      resources: {}
    name: "00"
    replicas: 1
  patroni:
    leaderLeaseDurationSeconds: 30
    port: 8008
    syncPeriodSeconds: 10
  port: 5432
  postgresVersion: 0
		`))
	})

	t.Run("empty proxy", func(t *testing.T) {
		var cluster v1beta1.PostgresCluster
		cluster.Spec.Proxy = new(v1beta1.PostgresProxySpec)
		cluster.Default()

		b, err := yaml.Marshal(cluster.Spec.Proxy)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), "pgBouncer: null\n")
	})

	t.Run("PgBouncer proxy", func(t *testing.T) {
		var cluster v1beta1.PostgresCluster
		cluster.Spec.Proxy = &v1beta1.PostgresProxySpec{PGBouncer: &v1beta1.PGBouncerPodSpec{}}
		cluster.Default()

		assert.Assert(t, MarshalsTo(cluster.Spec.Proxy, `
pgBouncer:
  port: 5432
  replicas: 1
		`))
	})
}

func TestPostgresInstanceSetSpecDefault(t *testing.T) {
	var spec v1beta1.PostgresInstanceSetSpec
	spec.Default(5)

	assert.Assert(t, MarshalsTo(spec, `
dataVolumeClaimSpec:
  resources: {}
name: "05"
replicas: 1
	`))
}

func TestMetadataGetLabels(t *testing.T) {
	for _, test := range []struct {
		m           v1beta1.Metadata
		mp          *v1beta1.Metadata
		expect      map[string]string
		description string
	}{{
		expect:      map[string]string(nil),
		description: "meta is defined but unset",
	}, {
		m:           v1beta1.Metadata{},
		mp:          &v1beta1.Metadata{},
		expect:      map[string]string(nil),
		description: "metadata is empty",
	}, {
		m:           v1beta1.Metadata{Labels: map[string]string{}},
		mp:          &v1beta1.Metadata{Labels: map[string]string{}},
		expect:      map[string]string{},
		description: "metadata contains empty label set",
	}, {
		m: v1beta1.Metadata{Labels: map[string]string{
			"test": "label",
		}},
		mp: &v1beta1.Metadata{Labels: map[string]string{
			"test": "label",
		}},
		expect: map[string]string{
			"test": "label",
		},
		description: "metadata contains labels",
	}, {
		m: v1beta1.Metadata{Labels: map[string]string{
			"test":  "label",
			"test2": "label2",
		}},
		mp: &v1beta1.Metadata{Labels: map[string]string{
			"test":  "label",
			"test2": "label2",
		}},
		expect: map[string]string{
			"test":  "label",
			"test2": "label2",
		},
		description: "metadata contains multiple labels",
	}} {
		t.Run(test.description, func(t *testing.T) {
			assert.DeepEqual(t, test.m.GetLabelsOrNil(), test.expect)
			assert.DeepEqual(t, test.mp.GetLabelsOrNil(), test.expect)
		})
	}
}

func TestMetadataGetAnnotations(t *testing.T) {
	for _, test := range []struct {
		m           v1beta1.Metadata
		mp          *v1beta1.Metadata
		expect      map[string]string
		description string
	}{{
		expect:      map[string]string(nil),
		description: "meta is defined but unset",
	}, {
		m:           v1beta1.Metadata{},
		mp:          &v1beta1.Metadata{},
		expect:      map[string]string(nil),
		description: "metadata is empty",
	}, {
		m:           v1beta1.Metadata{Annotations: map[string]string{}},
		mp:          &v1beta1.Metadata{Annotations: map[string]string{}},
		expect:      map[string]string{},
		description: "metadata contains empty annotation set",
	}, {
		m: v1beta1.Metadata{Annotations: map[string]string{
			"test": "annotation",
		}},
		mp: &v1beta1.Metadata{Annotations: map[string]string{
			"test": "annotation",
		}},
		expect: map[string]string{
			"test": "annotation",
		},
		description: "metadata contains annotations",
	}, {
		m: v1beta1.Metadata{Annotations: map[string]string{
			"test":  "annotation",
			"test2": "annotation2",
		}},
		mp: &v1beta1.Metadata{Annotations: map[string]string{
			"test":  "annotation",
			"test2": "annotation2",
		}},
		expect: map[string]string{
			"test":  "annotation",
			"test2": "annotation2",
		},
		description: "metadata contains multiple annotations",
	}} {
		t.Run(test.description, func(t *testing.T) {
			assert.DeepEqual(t, test.m.GetAnnotationsOrNil(), test.expect)
			assert.DeepEqual(t, test.mp.GetAnnotationsOrNil(), test.expect)
		})
	}
}
