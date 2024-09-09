// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// TODO(benjaminjb): This file is duplicated test help functions
// that could probably be put into a separate test_helper package

var (
	//TODO(tjmoore4): With the new RELATED_IMAGES defaulting behavior, tests could be refactored
	// to reference those environment variables instead of hard coded image values
	CrunchyPostgresHAImage = "registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-13.6-1"
	CrunchyPGBackRestImage = "registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:ubi8-2.38-0"
	CrunchyPGBouncerImage  = "registry.developers.crunchydata.com/crunchydata/crunchy-pgbouncer:ubi8-1.16-2"
)

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
				},
			},
			Proxy: &v1beta1.PostgresProxySpec{
				PGBouncer: &v1beta1.PGBouncerPodSpec{
					Image: CrunchyPGBouncerImage,
				},
			},
		},
	}
	return cluster.DeepCopy()
}

func testVolumeClaimSpec() corev1.PersistentVolumeClaimSpec {
	// Defines a volume claim spec that can be used to create instances
	return corev1.PersistentVolumeClaimSpec{
		AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
		Resources: corev1.VolumeResourceRequirements{
			Requests: map[corev1.ResourceName]resource.Quantity{
				corev1.ResourceStorage: resource.MustParse("1Gi"),
			},
		},
	}
}
