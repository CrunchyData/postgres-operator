// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"reflect"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/yaml"
)

func TestPostgresClusterWebhooks(t *testing.T) {
	var _ webhook.Defaulter = new(PostgresCluster)
}

func TestPostgresClusterDefault(t *testing.T) {
	t.Run("TypeMeta", func(t *testing.T) {
		var cluster PostgresCluster
		cluster.Default()

		assert.Equal(t, cluster.APIVersion, GroupVersion.String())
		assert.Equal(t, cluster.Kind, reflect.TypeOf(cluster).Name())
	})

	t.Run("no instance sets", func(t *testing.T) {
		var cluster PostgresCluster
		cluster.Default()

		b, err := yaml.Marshal(cluster)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  creationTimestamp: null
spec:
  backups:
    pgbackrest:
      repos: null
  config: {}
  instances: null
  patroni:
    leaderLeaseDurationSeconds: 30
    port: 8008
    syncPeriodSeconds: 10
  port: 5432
  postgresVersion: 0
status:
  monitoring: {}
  patroni: {}
  postgresVersion: 0
  proxy:
    pgBouncer: {}
								`)+"\n")
	})

	t.Run("one instance set", func(t *testing.T) {
		var cluster PostgresCluster
		cluster.Spec.InstanceSets = []PostgresInstanceSetSpec{{}}
		cluster.Default()

		b, err := yaml.Marshal(cluster)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  creationTimestamp: null
spec:
  backups:
    pgbackrest:
      repos: null
  config: {}
  instances:
  - dataVolumeClaimSpec:
      resources: {}
    name: "00"
    replicas: 1
    resources: {}
  patroni:
    leaderLeaseDurationSeconds: 30
    port: 8008
    syncPeriodSeconds: 10
  port: 5432
  postgresVersion: 0
status:
  monitoring: {}
  patroni: {}
  postgresVersion: 0
  proxy:
    pgBouncer: {}
								`)+"\n")
	})

	t.Run("empty proxy", func(t *testing.T) {
		var cluster PostgresCluster
		cluster.Spec.Proxy = new(PostgresProxySpec)
		cluster.Default()

		b, err := yaml.Marshal(cluster.Spec.Proxy)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), "pgBouncer: null\n")
	})

	t.Run("PgBouncer proxy", func(t *testing.T) {
		var cluster PostgresCluster
		cluster.Spec.Proxy = &PostgresProxySpec{PGBouncer: &PGBouncerPodSpec{}}
		cluster.Default()

		b, err := yaml.Marshal(cluster.Spec.Proxy)
		assert.NilError(t, err)
		assert.DeepEqual(t, string(b), strings.TrimSpace(`
pgBouncer:
  config: {}
  port: 5432
  replicas: 1
  resources: {}
		`)+"\n")
	})
}

func TestPostgresInstanceSetSpecDefault(t *testing.T) {
	var spec PostgresInstanceSetSpec
	spec.Default(5)

	b, err := yaml.Marshal(spec)
	assert.NilError(t, err)
	assert.DeepEqual(t, string(b), strings.TrimSpace(`
dataVolumeClaimSpec:
  resources: {}
name: "05"
replicas: 1
resources: {}
	`)+"\n")
}

func TestMetadataGetLabels(t *testing.T) {
	for _, test := range []struct {
		m           Metadata
		mp          *Metadata
		expect      map[string]string
		description string
	}{{
		expect:      map[string]string(nil),
		description: "meta is defined but unset",
	}, {
		m:           Metadata{},
		mp:          &Metadata{},
		expect:      map[string]string(nil),
		description: "metadata is empty",
	}, {
		m:           Metadata{Labels: map[string]string{}},
		mp:          &Metadata{Labels: map[string]string{}},
		expect:      map[string]string{},
		description: "metadata contains empty label set",
	}, {
		m: Metadata{Labels: map[string]string{
			"test": "label",
		}},
		mp: &Metadata{Labels: map[string]string{
			"test": "label",
		}},
		expect: map[string]string{
			"test": "label",
		},
		description: "metadata contains labels",
	}, {
		m: Metadata{Labels: map[string]string{
			"test":  "label",
			"test2": "label2",
		}},
		mp: &Metadata{Labels: map[string]string{
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
		m           Metadata
		mp          *Metadata
		expect      map[string]string
		description string
	}{{
		expect:      map[string]string(nil),
		description: "meta is defined but unset",
	}, {
		m:           Metadata{},
		mp:          &Metadata{},
		expect:      map[string]string(nil),
		description: "metadata is empty",
	}, {
		m:           Metadata{Annotations: map[string]string{}},
		mp:          &Metadata{Annotations: map[string]string{}},
		expect:      map[string]string{},
		description: "metadata contains empty annotation set",
	}, {
		m: Metadata{Annotations: map[string]string{
			"test": "annotation",
		}},
		mp: &Metadata{Annotations: map[string]string{
			"test": "annotation",
		}},
		expect: map[string]string{
			"test": "annotation",
		},
		description: "metadata contains annotations",
	}, {
		m: Metadata{Annotations: map[string]string{
			"test":  "annotation",
			"test2": "annotation2",
		}},
		mp: &Metadata{Annotations: map[string]string{
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
