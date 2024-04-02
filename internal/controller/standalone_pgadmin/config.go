// Copyright 2023 - 2024 Crunchy Data Solutions, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
