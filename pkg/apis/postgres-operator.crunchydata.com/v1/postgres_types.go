// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type PostgresConfigSpec struct {
	// Files to mount under "/etc/postgres".
	// ---
	// +optional
	Files []corev1.VolumeProjection `json:"files,omitempty"`

	// Configuration parameters for the PostgreSQL server. Some values will
	// be reloaded without validation and some cause PostgreSQL to restart.
	// Some values cannot be changed at all.
	// More info: https://www.postgresql.org/docs/current/runtime-config.html
	// ---
	//
	// Postgres 17 has something like 350+ built-in parameters, but typically
	// an administrator will change only a handful of these.
	// +kubebuilder:validation:MaxProperties=50
	//
	// # File Locations
	// - https://www.postgresql.org/docs/current/runtime-config-file-locations.html
	//
	// +kubebuilder:validation:XValidation:rule=`!has(self.config_file) && !has(self.data_directory)`,message=`cannot change PGDATA path: config_file, data_directory`
	// +kubebuilder:validation:XValidation:rule=`!has(self.external_pid_file)`,message=`cannot change external_pid_file`
	// +kubebuilder:validation:XValidation:rule=`!has(self.hba_file) && !has(self.ident_file)`,message=`cannot change authentication path: hba_file, ident_file`
	//
	// # Connections
	// - https://www.postgresql.org/docs/current/runtime-config-connection.html
	//
	// +kubebuilder:validation:XValidation:rule=`!has(self.listen_addresses)`,message=`network connectivity is always enabled: listen_addresses`
	// +kubebuilder:validation:XValidation:rule=`!has(self.port)`,message=`change port using .spec.port instead`
	// +kubebuilder:validation:XValidation:rule=`!has(self.ssl) && !self.exists(k, k.startsWith("ssl_") && !(k == 'ssl_groups' || k == 'ssl_ecdh_curve'))`,message=`TLS is always enabled`
	// +kubebuilder:validation:XValidation:rule=`!self.exists(k, k.startsWith("unix_socket_"))`,message=`domain socket paths cannot be changed`
	//
	// # Write Ahead Log
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html
	//
	// +kubebuilder:validation:XValidation:rule=`!has(self.wal_level) || self.wal_level in ["logical"]`,message=`wal_level must be "replica" or higher`
	// +kubebuilder:validation:XValidation:rule=`!has(self.wal_log_hints)`,message=`wal_log_hints are always enabled`
	// +kubebuilder:validation:XValidation:rule=`!has(self.archive_mode) && !has(self.archive_command) && !has(self.restore_command)`
	// +kubebuilder:validation:XValidation:rule=`!has(self.recovery_target) && !self.exists(k, k.startsWith("recovery_target_"))`
	//
	// # Replication
	// - https://www.postgresql.org/docs/current/runtime-config-replication.html
	//
	// +kubebuilder:validation:XValidation:rule=`!has(self.hot_standby)`,message=`hot_standby is always enabled`
	// +kubebuilder:validation:XValidation:rule=`!has(self.synchronous_standby_names)`
	// +kubebuilder:validation:XValidation:rule=`!has(self.primary_conninfo) && !has(self.primary_slot_name)`
	// +kubebuilder:validation:XValidation:rule=`!has(self.recovery_min_apply_delay)`,message=`delayed replication is not supported at this time`
	//
	// # Logging
	// - https://www.postgresql.org/docs/current/runtime-config-logging.html
	//
	// +kubebuilder:validation:XValidation:rule=`!has(self.cluster_name)`,message=`cluster_name is derived from the PostgresCluster name`
	// +kubebuilder:validation:XValidation:rule=`!has(self.logging_collector)`,message=`disabling logging_collector is unsafe`
	// +kubebuilder:validation:XValidation:rule=`!has(self.log_file_mode)`,message=`log_file_mode cannot be changed`
	//
	// +kubebuilder:validation:XValidation:fieldPath=`.log_directory`,message=`must start with "/pgdata/logs/postgres", "/pgtmp/logs/postgres", "/pgwal/logs/postgres", "/volumes", or be "log" to keep logs inside PGDATA`,rule=`self.?log_directory.optMap(v, type(v) == string && (v == "log" || v.startsWith("/volumes") || ["/pgdata","/pgtmp","/pgwal","/volumes"].exists(p, v == (p + "/logs/postgres") || v.startsWith(p + "/logs/postgres/")))).orValue(true)`
	//
	// +mapType=granular
	// +optional
	Parameters map[string]intstr.IntOrString `json:"parameters,omitempty"`
}
