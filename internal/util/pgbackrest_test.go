// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestGetPGBackRestLogPathForInstance(t *testing.T) {
	t.Run("NoSpecPath", func(t *testing.T) {
		postgrescluster := &v1beta1.PostgresCluster{
			Spec: v1beta1.PostgresClusterSpec{
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{},
				},
			},
		}
		assert.Equal(t, GetPGBackRestLogPathForInstance(postgrescluster), naming.PGBackRestPGDataLogPath)
	})

	t.Run("SpecPathSet", func(t *testing.T) {
		postgrescluster := &v1beta1.PostgresCluster{
			Spec: v1beta1.PostgresClusterSpec{
				Backups: v1beta1.Backups{
					PGBackRest: v1beta1.PGBackRestArchive{
						Log: &v1beta1.LoggingConfiguration{
							Path: "/volumes/test/log",
						},
					},
				},
			},
		}
		assert.Equal(t, GetPGBackRestLogPathForInstance(postgrescluster), "/volumes/test/log")
	})
}
