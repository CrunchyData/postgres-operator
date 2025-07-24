// Copyright 2023 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package standalone_pgadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"

	"github.com/crunchydata/postgres-operator/internal/collector"
	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

// +kubebuilder:rbac:groups="",resources="configmaps",verbs={get}
// +kubebuilder:rbac:groups="",resources="configmaps",verbs={create,delete,patch}

// reconcilePGAdminConfigMap writes the ConfigMap for pgAdmin.
func (r *PGAdminReconciler) reconcilePGAdminConfigMap(
	ctx context.Context, pgadmin *v1beta1.PGAdmin,
	clusters map[string][]*v1beta1.PostgresCluster,
) (*corev1.ConfigMap, error) {
	configmap, err := configmap(ctx, pgadmin, clusters)
	if err != nil {
		return configmap, err
	}

	err = collector.EnablePgAdminLogging(ctx, pgadmin.Spec.Instrumentation, configmap)

	if err == nil {
		err = errors.WithStack(r.setControllerReference(pgadmin, configmap))
	}
	if err == nil {
		err = errors.WithStack(r.apply(ctx, configmap))
	}

	return configmap, err
}

// configmap returns a v1.ConfigMap for pgAdmin.
func configmap(ctx context.Context, pgadmin *v1beta1.PGAdmin,
	clusters map[string][]*v1beta1.PostgresCluster,
) (*corev1.ConfigMap, error) {
	configmap := &corev1.ConfigMap{ObjectMeta: naming.StandalonePGAdmin(pgadmin)}
	configmap.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))

	configmap.Annotations = pgadmin.Spec.Metadata.GetAnnotationsOrNil()
	configmap.Labels = naming.Merge(
		pgadmin.Spec.Metadata.GetLabelsOrNil(),
		naming.StandalonePGAdminLabels(pgadmin.Name))

	// TODO(tjmoore4): Populate configuration details.
	initialize.Map(&configmap.Data)
	var (
		logRetention             bool
		maxBackupRetentionNumber = 1
		// One day in minutes for pgadmin rotation
		pgAdminRetentionPeriod = 24 * 60
		// Daily rotation for gunicorn rotation
		gunicornRetentionPeriod = "D"
	)
	// If OTel logs feature gate is enabled, we want to change the pgAdmin/gunicorn logging
	if collector.OpenTelemetryLogsEnabled(ctx, pgadmin) {
		logRetention = true

		// If the user has set a retention period, we will use those values for log rotation,
		// which is otherwise managed by python.
		if pgadmin.Spec.Instrumentation.Logs != nil &&
			pgadmin.Spec.Instrumentation.Logs.RetentionPeriod != nil {

			retentionNumber, period := collector.ParseDurationForLogrotate(pgadmin.Spec.Instrumentation.Logs.RetentionPeriod.AsDuration())
			// `LOG_ROTATION_MAX_LOG_FILES`` in pgadmin refers to the already rotated logs.
			// `backupCount` for gunicorn is similar.
			// Our retention unit is for total number of log files, so subtract 1 to account
			// for the currently-used log file.
			maxBackupRetentionNumber = retentionNumber - 1
			if period == "hourly" {
				// If the period is hourly, set the pgadmin
				// and gunicorn retention periods to hourly.
				pgAdminRetentionPeriod = 60
				gunicornRetentionPeriod = "H"
			}
		}
	}
	configSettings, err := generateConfig(pgadmin, logRetention, maxBackupRetentionNumber, pgAdminRetentionPeriod)
	if err == nil {
		configmap.Data[settingsConfigMapKey] = configSettings
	}

	clusterSettings, err := generateClusterConfig(clusters)
	if err == nil {
		configmap.Data[settingsClusterMapKey] = clusterSettings
	}

	gunicornSettings, err := generateGunicornConfig(pgadmin,
		logRetention, maxBackupRetentionNumber, gunicornRetentionPeriod)
	if err == nil {
		configmap.Data[gunicornConfigKey] = gunicornSettings
	}

	return configmap, err
}

// generateConfigs generates the config settings for the pgAdmin and gunicorn
func generateConfig(pgadmin *v1beta1.PGAdmin,
	logRetention bool, maxBackupRetentionNumber, pgAdminRetentionPeriod int) (
	string, error) {
	settings := map[string]any{
		// Bind to all IPv4 addresses by default. "0.0.0.0" here represents INADDR_ANY.
		// - https://flask.palletsprojects.com/en/2.2.x/api/#flask.Flask.run
		// - https://flask.palletsprojects.com/en/2.3.x/api/#flask.Flask.run
		"DEFAULT_SERVER": "0.0.0.0",
	}

	// Copy any specified settings over the defaults.
	maps.Copy(settings, pgadmin.Spec.Config.Settings)

	// Write mandatory settings over any specified ones.
	// SERVER_MODE must always be enabled when running on a webserver.
	// - https://github.com/pgadmin-org/pgadmin4/blob/REL-7_7/web/config.py#L110
	settings["SERVER_MODE"] = true
	settings["UPGRADE_CHECK_ENABLED"] = false
	settings["UPGRADE_CHECK_URL"] = ""
	settings["UPGRADE_CHECK_KEY"] = ""
	settings["DATA_DIR"] = dataMountPath
	settings["LOG_FILE"] = LogFileAbsolutePath

	if logRetention {
		settings["LOG_ROTATION_AGE"] = pgAdminRetentionPeriod
		settings["LOG_ROTATION_MAX_LOG_FILES"] = maxBackupRetentionNumber
		settings["JSON_LOGGER"] = true
		settings["CONSOLE_LOG_LEVEL"] = "WARNING"
		settings["FILE_LOG_LEVEL"] = "INFO"
		settings["FILE_LOG_FORMAT_JSON"] = map[string]string{
			"time":    "created",
			"name":    "name",
			"level":   "levelname",
			"message": "message",
		}
	}

	// To avoid spurious reconciles, the following value must not change when
	// the spec does not change. [json.Encoder] and [json.Marshal] do this by
	// emitting map keys in sorted order. Indent so the value is not rendered
	// as one long line by `kubectl`.
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(settings)

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
	clusters map[string][]*v1beta1.PostgresCluster,
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
	for _, serverGroupName := range keys {
		slices.SortFunc(clusters[serverGroupName], func(a, b *v1beta1.PostgresCluster) int {
			return strings.Compare(a.Name, b.Name)
		})
		for _, cluster := range clusters[serverGroupName] {
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
			clusterServers[len(clusterServers)+1] = object
		}
	}
	servers := map[string]any{
		"Servers": clusterServers,
	}
	err := encoder.Encode(servers)
	return buffer.String(), err
}

// generateGunicornConfig generates the config settings for the gunicorn server
// - https://docs.gunicorn.org/en/latest/settings.html
func generateGunicornConfig(pgadmin *v1beta1.PGAdmin,
	logRetention bool, maxBackupRetentionNumber int, gunicornRetentionPeriod string,
) (string, error) {
	settings := map[string]any{
		// Bind to all IPv4 addresses and set 25 threads by default.
		// - https://docs.gunicorn.org/en/latest/settings.html#bind
		// - https://docs.gunicorn.org/en/latest/settings.html#threads
		"bind":    "0.0.0.0:" + strconv.Itoa(pgAdminPort),
		"threads": 25,
	}

	// Copy any specified settings over the defaults.
	maps.Copy(settings, pgadmin.Spec.Config.Gunicorn)

	// Write mandatory settings over any specified ones.
	// - https://docs.gunicorn.org/en/latest/settings.html#workers
	settings["workers"] = 1
	// Gunicorn logging dict settings
	logSettings := map[string]any{}

	// If OTel logs feature gate is enabled, we want to change the gunicorn logging
	if logRetention {

		// Gunicorn uses the Python logging package, which sets the following attributes:
		// https://docs.python.org/3/library/logging.html#logrecord-attributes.
		// JsonFormatter is used to format the log: https://pypi.org/project/jsonformatter/
		// We override the gunicorn defaults (using `logconfig_dict`) to set our own file handler.
		// - https://docs.gunicorn.org/en/stable/settings.html#logconfig-dict
		// - https://github.com/benoitc/gunicorn/blob/23.0.0/gunicorn/glogging.py#L47
		logSettings = map[string]any{

			"loggers": map[string]any{
				"gunicorn.access": map[string]any{
					"handlers":  []string{"file"},
					"level":     "INFO",
					"propagate": true,
					"qualname":  "gunicorn.access",
				},
				"gunicorn.error": map[string]any{
					"handlers":  []string{"file"},
					"level":     "INFO",
					"propagate": true,
					"qualname":  "gunicorn.error",
				},
			},
			"handlers": map[string]any{
				"file": map[string]any{
					"class":       "logging.handlers.TimedRotatingFileHandler",
					"filename":    GunicornLogFileAbsolutePath,
					"backupCount": maxBackupRetentionNumber,
					"interval":    1,
					"when":        gunicornRetentionPeriod,
					"formatter":   "json",
				},
				"console": map[string]any{
					"class":     "logging.StreamHandler",
					"formatter": "generic",
					"stream":    "ext://sys.stdout",
				},
			},
			"formatters": map[string]any{
				"generic": map[string]any{
					"class":   "logging.Formatter",
					"datefmt": "[%Y-%m-%d %H:%M:%S %z]",
					"format":  "%(asctime)s [%(process)d] [%(levelname)s] %(message)s",
				},
				"json": map[string]any{
					"class":      "jsonformatter.JsonFormatter",
					"separators": []string{",", ":"},
					"format": map[string]string{
						"time":    "created",
						"name":    "name",
						"level":   "levelname",
						"message": "message",
					},
				},
			},
		}
	}
	settings["logconfig_dict"] = logSettings

	// To avoid spurious reconciles, the following value must not change when
	// the spec does not change. [json.Encoder] and [json.Marshal] do this by
	// emitting map keys in sorted order. Indent so the value is not rendered
	// as one long line by `kubectl`.
	buffer := new(bytes.Buffer)
	encoder := json.NewEncoder(buffer)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	err := encoder.Encode(settings)

	return buffer.String(), err
}
