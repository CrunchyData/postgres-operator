// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/postgres"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestPostgreSQLParameters(t *testing.T) {
	cluster := new(v1beta1.PostgresCluster)
	parameters := new(postgres.Parameters)

	PostgreSQL(cluster, parameters, true)
	assert.DeepEqual(t, parameters.Mandatory.AsMap(), map[string]string{
		"archive_mode":    "on",
		"archive_command": `pgbackrest --stanza=db archive-push "%p"`,
		"restore_command": `pgbackrest --stanza=db archive-get %f "%p"`,
	})

	assert.DeepEqual(t, parameters.Default.AsMap(), map[string]string{
		"archive_timeout": "60s",
	})

	PostgreSQL(cluster, parameters, false)
	assert.DeepEqual(t, parameters.Mandatory.AsMap(), map[string]string{
		"archive_mode":    "on",
		"archive_command": "true",
		"restore_command": `pgbackrest --stanza=db archive-get %f "%p"`,
	})

	cluster.Spec.Standby = &v1beta1.PostgresStandbySpec{
		Enabled:  true,
		RepoName: "repo99",
	}

	PostgreSQL(cluster, parameters, true)
	assert.DeepEqual(t, parameters.Mandatory.AsMap(), map[string]string{
		"archive_mode":    "on",
		"archive_command": `pgbackrest --stanza=db archive-push "%p"`,
		"restore_command": `pgbackrest --stanza=db archive-get %f "%p" --repo=99`,
	})
}
