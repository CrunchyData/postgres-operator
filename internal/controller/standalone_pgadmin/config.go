// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

// Include configs here used by multiple files
const (
	// ConfigMap keys used also in mounting volume to pod
	settingsConfigMapKey  = "pgadmin-settings.json"
	settingsClusterMapKey = "pgadmin-shared-clusters.json"
	gunicornConfigKey     = "gunicorn-config.json"

	// Port address used to define pod and service
	pgAdminPort = 5050

	// Directory for pgAdmin in container
	pgAdminDir = "/usr/local/lib/python3.11/site-packages/pgadmin4"
)
