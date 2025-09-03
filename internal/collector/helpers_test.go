// Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package collector

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

var (
	// TODO (testing): With the new RELATED_IMAGES defaulting behavior, tests could be refactored
	// to reference those environment variables instead of hard coded image values
	CrunchyPostgresHAImage = "registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-13.6-1"
	CrunchyPGBackRestImage = "registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:ubi8-2.38-0"
	CrunchyPGBouncerImage  = "registry.developers.crunchydata.com/crunchydata/crunchy-pgbouncer:ubi8-1.16-2"
)

func testInstrumentationSpec() *v1beta1.InstrumentationSpec {
	spec := v1beta1.InstrumentationSpec{
		Config: &v1beta1.InstrumentationConfigSpec{
			Exporters: map[string]any{
				"googlecloud": map[string]any{
					"log": map[string]any{
						"default_log_name": "opentelemetry.io/collector-exported-log",
					},
					"project": "google-project-name",
				},
			},
		},
		Logs: &v1beta1.InstrumentationLogsSpec{
			Exporters: []string{"googlecloud"},
		},
		Metrics: &v1beta1.InstrumentationMetricsSpec{
			Exporters: []string{"googlecloud"},
		},
	}

	return spec.DeepCopy()
}

// Copied from postgrescluster package
func testVolumeClaimSpec() v1beta1.VolumeClaimSpec {
	// Defines a volume claim spec that can be used to create instances
	return v1beta1.VolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}

// Copied from postgrescluster package (and then edited)
func testCluster() *v1beta1.PostgresCluster {
	// Defines a base cluster spec that can be used by tests to generate a
	// cluster with an expected number of instances
	cluster := v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: "hippo",
		},
		Spec: v1beta1.PostgresClusterSpec{
			PostgresVersion: 13,
			Image:           CrunchyPostgresHAImage,
			ImagePullSecrets: []corev1.LocalObjectReference{{
				Name: "myImagePullSecret"},
			},
			InstanceSets: []v1beta1.PostgresInstanceSetSpec{{
				Name:                "instance1",
				Replicas:            initialize.Int32(1),
				DataVolumeClaimSpec: testVolumeClaimSpec(),
			}},
			Backups: v1beta1.Backups{
				PGBackRest: v1beta1.PGBackRestArchive{
					Image: CrunchyPGBackRestImage,
					Repos: []v1beta1.PGBackRestRepo{{
						Name: "repo1",
						Volume: &v1beta1.RepoPVC{
							VolumeClaimSpec: testVolumeClaimSpec(),
						},
					}},
					RepoHost: &v1beta1.PGBackRestRepoHost{},
				},
			},
			Proxy: &v1beta1.PostgresProxySpec{
				PGBouncer: &v1beta1.PGBouncerPodSpec{
					Image: CrunchyPGBouncerImage,
				},
			},
			Instrumentation: testInstrumentationSpec(),
		},
	}
	return cluster.DeepCopy()
}
