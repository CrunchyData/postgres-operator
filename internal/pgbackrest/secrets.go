package pgbackrest

// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

import (
	"fmt"
	"os"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// setBackrestRepoContainerImageAndEnv sets the backrest container image and needed environment variables
func setBackrestRepoContainerImageAndEnv(cluster *v1beta1.PostgresCluster, container *corev1.Container,
	repoIndex int, podEnvVars []corev1.EnvVar) error {
	// create a corev1.EnvVar slice to store any environment variables generated
	var envVars []corev1.EnvVar
	var err error

	// if s3, gcs or azure is enabled, set proper env vars
	if cluster.Spec.Backups.PGBackRest.Repos[repoIndex].S3 != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  fmt.Sprintf("PGBACKREST_REPO%d_TYPE", repoIndex+1),
			Value: "s3",
		})
	} else if cluster.Spec.Backups.PGBackRest.Repos[repoIndex].GCS != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  fmt.Sprintf("PGBACKREST_REPO%d_TYPE", repoIndex+1),
			Value: "gcs",
		})
	} else if cluster.Spec.Backups.PGBackRest.Repos[repoIndex].Azure != nil {
		envVars = append(envVars, corev1.EnvVar{
			Name:  fmt.Sprintf("PGBACKREST_REPO%d_TYPE", repoIndex+1),
			Value: "azure",
		})

		// Only check the environment variable for AAD usage
		if os.Getenv("PGBACKREST_AZURE_USE_AAD") == "true" {
			envVars = append(envVars, corev1.EnvVar{
				Name:  "PGBACKREST_REPO_AZURE_USE_AAD",
				Value: "true",
			})
		}
	} else {
		envVars = append(envVars, corev1.EnvVar{
			Name:  fmt.Sprintf("PGBACKREST_REPO%d_TYPE", repoIndex+1),
			Value: "posix",
		})
	}

	// add the appropriate pgBackRest env vars to the existing container
	container.Env = append(container.Env, podEnvVars...)
	container.Env = append(container.Env, envVars...)

	return err
}
