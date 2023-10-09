// Copyright 2023 Crunchy Data Solutions, Inc.
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

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"

	corev1 "k8s.io/api/core/v1"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={get}
// +kubebuilder:rbac:groups="",resources="configmaps",verbs={create,delete,patch}

// reconcilePGAdminConfigMap writes the ConfigMap for pgAdmin.
func (r *PGAdminReconciler) reconcilePGAdminConfigMap(
	ctx context.Context, pgadmin *v1beta1.PGAdmin,
	clusters map[string]*v1beta1.PostgresClusterList,
) (*corev1.ConfigMap, error) {
	configmap, err := configmap(pgadmin, clusters)
	if err == nil {
		err = errors.WithStack(r.setControllerReference(pgadmin, configmap))
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, configmap))
	}

	return configmap, err
}

// configmap returns a v1.ConfigMap for pgAdmin.
func configmap(pgadmin *v1beta1.PGAdmin,
	clusters map[string]*v1beta1.PostgresClusterList,
) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	configmap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	configmap.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	configmap.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		map[string]string{
			naming.LabelStandalonePGAdmin: pgadmin.Name,
			naming.LabelRole:              naming.RolePGAdmin,
		})

	// TODO(tjmoore4): Populate configuration details.
	initialize.StringMap(&configmap.Data)
	configSettings, err := generateConfig(pgadmin)
	if err == nil {
		configmap.Data[settingsConfigMapKey] = configSettings
	}

	clusterSettings, err := generateClusterConfig(clusters)
	if err == nil {
		configmap.Data[settingsClusterMapKey] = clusterSettings
	}

	return configmap, err
}

// systemSettings returns pgAdmin settings as a value that can be marshaled to JSON.
func systemSettings(pgadmin *v1beta1.PGAdmin) map[string]interface{} {
	settings := *pgadmin.Spec.Config.Settings.DeepCopy()
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// SERVER_MODE must always be enabled when running on a webserver.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/config.py#L105
	settings["SERVER_MODE"] = true
	settings["UPGRADE_CHECK_ENABLED"] = false
	settings["UPGRADE_CHECK_URL"] = ""
	settings["UPGRADE_CHECK_KEY"] = ""

	return settings
}

// generateConfig generates the config settings for the pgAdmin
func generateConfig(pgadmin *v1beta1.PGAdmin) (string, error) {
	// To avoid spurious reconciles, the following value must not change when
	// the spec does not change. [json.Encoder] and [json.Marshal] do this by
	// emitting map keys in sorted order. Indent so the value is not rendered
	// as one long line by `kubectl`.
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(systemSettings(pgadmin))

	return buffer.String(), err
}

// generateClusterConfig generates the settings for the servers registered in pgAdmin.
// pgAdmin's `setup.py --load-server` function ingests this list of servers as JSON,
// in the following form:
//
//	{
//		"Servers": {
//			"1": {
//				"Name": "Minimally Defined Server",
//				"Group": "Server Group 1",
//				"Port": 5432,
//				"Username": "postgres",
//				"Host": "localhost",
//				"SSLMode": "prefer",
//				"MaintenanceDB": "postgres"
//			},
//			"2": { ... }
//		}
//	}
func generateClusterConfig(
	clusters map[string]*v1beta1.PostgresClusterList,
) (string, error) {
	// To avoid spurious reconciles, the following value must not change when
	// the spec does not change. [json.Encoder] and [json.Marshal] do this by
	// emitting map keys in sorted order. Indent so the value is not rendered
	// as one long line by `kubectl`.
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	// To avoid spurious reconciles, we want to keep the `clusters` order consistent
	// which we can do by
	// a) sorting the ServerGroup name used as a key; and
	// b) sorting the clusters by name;
	keys := []string{}
	for key := range clusters {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	clusterServers := map[int]any{}
	// Because we allow multiple ServerGroups to be defined, we use `currentOffset` to keep
	// track of the last number added to the `Servers` group
	var currentOffset = 0
	for _, serverGroupName := range keys {
		sort.Slice(clusters[serverGroupName].Items,
			func(i, j int) bool {
				return clusters[serverGroupName].Items[i].Name < clusters[serverGroupName].Items[j].Name
			})
		for i, cluster := range clusters[serverGroupName].Items {
			object := map[string]any{
				"Name":          cluster.Name,
				"Group":         serverGroupName,
				"Host":          fmt.Sprintf("%s-primary.%s.svc", cluster.Name, cluster.Namespace),
				"Port":          5432,
				"MaintenanceDB": "postgres",
				"Username":      cluster.Name,
				// `SSLMode` and some other settings may need to be set by the user in the future
				"SSLMode": "prefer",
				"Shared":  true,
			}
			clusterServers[i+1+currentOffset] = object
		}
		currentOffset = len(clusters[serverGroupName].Items) + currentOffset
	}
	servers := map[string]any{
		"Servers": clusterServers,
	}
	err := encoder.Encode(servers)
	return buffer.String(), err
}
