// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type PostgresAuthenticationSpec struct {
	// Postgres compares every new connection to these rules in the order they are
	// defined. The first rule that matches determines if and how the connection
	// must then authenticate. Connections that match no rules are disconnected.
	//
	// When this is omitted or empty, Postgres accepts encrypted connections to any
	// database from users that have a password. To refuse all network connections,
	// set this to one rule that matches "host" connections to the "reject" method.
	//
	// More info: https://www.postgresql.org/docs/current/auth-pg-hba-conf.html
	// ---
	// +kubebuilder:validation:MaxItems=10
	// +listType=atomic
	// +optional
	Rules []PostgresHBARuleSpec `json:"rules,omitempty"`
}

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
	// +kubebuilder:validation:XValidation:rule=`!has(self.ssl) && !self.exists(k, k.startsWith("ssl_"))`,message=`TLS is always enabled`
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
	// +mapType=granular
	// +optional
	Parameters map[string]intstr.IntOrString `json:"parameters,omitempty"`
}

// ---
type PostgresHBARule struct {
	// The connection transport this rule matches. Typical values are:
	//  1. "host" for network connections that may or may not be encrypted.
	//  2. "hostssl" for network connections encrypted using TLS.
	//  3. "hostgssenc" for network connections encrypted using GSSAPI.
	// ---
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=20
	// +kubebuilder:validation:Pattern=`^[-a-z0-9]+$`
	// +optional
	Connection string `json:"connection,omitempty"`

	// Which databases this rule matches. When omitted or empty, this rule matches all databases.
	// ---
	// +kubebuilder:validation:MaxItems=20
	// +listType=atomic
	// +optional
	Databases []PostgresIdentifier `json:"databases,omitempty"`

	// The authentication method to use when a connection matches this rule.
	// The special value "reject" refuses connections that match this rule.
	//
	// More info: https://www.postgresql.org/docs/current/auth-methods.html
	// ---
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=20
	// +kubebuilder:validation:Pattern=`^[-a-z0-9]+$`
	// +kubebuilder:validation:XValidation:rule=`self != "trust"`,message=`the "trust" method is unsafe`
	// +optional
	Method string `json:"method,omitempty"`

	// Additional settings for this rule or its authentication method.
	// ---
	// +kubebuilder:validation:MaxProperties=20
	// +mapType=atomic
	// +optional
	Options map[string]intstr.IntOrString `json:"options,omitempty"`

	// Which user names this rule matches. When omitted or empty, this rule matches all users.
	// ---
	// +kubebuilder:validation:MaxItems=20
	// +listType=atomic
	// +optional
	Users []PostgresIdentifier `json:"users,omitempty"`
}

// ---
// Emulate OpenAPI "anyOf" aka Kubernetes union.
// +kubebuilder:validation:XValidation:rule=`[has(self.hba), has(self.connection) || has(self.databases) || has(self.method) || has(self.options) || has(self.users)].exists_one(b,b)`,message=`"hba" cannot be combined with other fields`
// +kubebuilder:validation:XValidation:rule=`has(self.hba) || (has(self.connection) && has(self.method))`,message=`"connection" and "method" are required`
//
// Some authentication methods *must* be further configured via options.
//
// https://git.postgresql.org/gitweb/?p=postgresql.git;hb=refs/tags/REL_10_0;f=src/backend/libpq/hba.c#l1501
// https://git.postgresql.org/gitweb/?p=postgresql.git;hb=refs/tags/REL_17_0;f=src/backend/libpq/hba.c#l1886
// +kubebuilder:validation:XValidation:message=`the "ldap" method requires an "ldapbasedn", "ldapprefix", or "ldapsuffix" option`,rule=`has(self.hba) || self.method != "ldap" || (has(self.options) && ["ldapbasedn","ldapprefix","ldapsuffix"].exists(k, k in self.options))`
// +kubebuilder:validation:XValidation:message=`cannot use "ldapbasedn", "ldapbinddn", "ldapbindpasswd", "ldapsearchattribute", or "ldapsearchfilter" options with "ldapprefix" or "ldapsuffix" options`,rule=`has(self.hba) || self.method != "ldap" || !has(self.options) || 2 > size([["ldapprefix","ldapsuffix"], ["ldapbasedn","ldapbinddn","ldapbindpasswd","ldapsearchattribute","ldapsearchfilter"]].filter(a, a.exists(k, k in self.options)))`
//
// https://git.postgresql.org/gitweb/?p=postgresql.git;hb=refs/tags/REL_10_0;f=src/backend/libpq/hba.c#l1539
// https://git.postgresql.org/gitweb/?p=postgresql.git;hb=refs/tags/REL_17_0;f=src/backend/libpq/hba.c#l1945
// +kubebuilder:validation:XValidation:message=`the "radius" method requires "radiusservers" and "radiussecrets" options`,rule=`has(self.hba) || self.method != "radius" || (has(self.options) && ["radiusservers","radiussecrets"].all(k, k in self.options))`
//
// +structType=atomic
type PostgresHBARuleSpec struct {
	// One line of the "pg_hba.conf" file. Changes to this value will be automatically reloaded without validation.
	// ---
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=100
	// +kubebuilder:validation:Pattern=`^[[:print:]]+$`
	// +kubebuilder:validation:XValidation:rule=`!self.trim().startsWith("include")`,message=`cannot include other files`
	// +optional
	HBA string `json:"hba,omitempty"`

	PostgresHBARule `json:",inline"`
}

// ---
// PostgreSQL identifiers are limited in length but may contain any character.
// - https://www.postgresql.org/docs/current/sql-syntax-lexical.html#SQL-SYNTAX-IDENTIFIERS
// +kubebuilder:validation:MinLength=1
// +kubebuilder:validation:MaxLength=63
type PostgresIdentifier = string

type PostgresPasswordSpec struct {
	// Type of password to generate. Defaults to ASCII. Valid options are ASCII
	// and AlphaNumeric.
	// "ASCII" passwords contain letters, numbers, and symbols from the US-ASCII character set.
	// "AlphaNumeric" passwords contain letters and numbers from the US-ASCII character set.
	// ---
	// Kubernetes assumes the evaluation cost of an enum value is very large.
	// TODO(k8s-1.29): Drop MaxLength after Kubernetes 1.29; https://issue.k8s.io/119511
	// +kubebuilder:validation:MaxLength=15
	//
	// +kubebuilder:default=ASCII
	// +kubebuilder:validation:Enum={ASCII,AlphaNumeric}
	// +required
	Type string `json:"type"`
}

// PostgresPasswordSpec types.
const (
	PostgresPasswordTypeAlphaNumeric = "AlphaNumeric"
	PostgresPasswordTypeASCII        = "ASCII"
)

type PostgresUserSpec struct {
	// The name of this PostgreSQL user. The value may contain only lowercase
	// letters, numbers, and hyphen so that it fits into Kubernetes metadata.
	// ---
	// This value goes into the name of a corev1.Secret and a label value, so
	// it must match both IsDNS1123Subdomain and IsValidLabelValue.
	// - https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsDNS1123Subdomain
	// - https://pkg.go.dev/k8s.io/apimachinery/pkg/util/validation#IsValidLabelValue
	//
	// This is IsDNS1123Subdomain without any dots, U+002E:
	// +kubebuilder:validation:Pattern=`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`
	//
	// +required
	Name PostgresIdentifier `json:"name"`

	// Databases to which this user can connect and create objects. Removing a
	// database from this list does NOT revoke access. This field is ignored for
	// the "postgres" user.
	// ---
	// +listType=set
	// +optional
	Databases []PostgresIdentifier `json:"databases,omitempty"`

	// ALTER ROLE options except for PASSWORD. This field is ignored for the
	// "postgres" user.
	// More info: https://www.postgresql.org/docs/current/role-attributes.html
	// ---
	// +kubebuilder:validation:MaxLength=200
	// +kubebuilder:validation:Pattern=`^[^;]*$`
	// +kubebuilder:validation:XValidation:rule=`!self.matches("(?i:PASSWORD)")`,message="cannot assign password"
	// +kubebuilder:validation:XValidation:rule=`!self.matches("(?:--|/[*]|[*]/)")`,message="cannot contain comments"
	// +optional
	Options string `json:"options,omitempty"`

	// Properties of the password generated for this user.
	// ---
	// +optional
	Password *PostgresPasswordSpec `json:"password,omitempty"`
}
