// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"
	"maps"
	"slices"
	"strings"
)

// NewParameters returns ParameterSets required by this package.
func NewParameters() Parameters {
	parameters := Parameters{
		Mandatory: NewParameterSet(),
		Default:   NewParameterSet(),
	}

	// Use UNIX domain sockets for local connections.
	// PostgreSQL must be restarted when changing this value.
	parameters.Mandatory.Add("unix_socket_directories", SocketDirectory)

	// Enable logical replication in addition to streaming and WAL archiving.
	// PostgreSQL must be restarted when changing this value.
	// - https://www.postgresql.org/docs/current/runtime-config-wal.html#GUC-WAL-LEVEL
	// - https://www.postgresql.org/docs/current/runtime-config-replication.html
	// - https://www.postgresql.org/docs/current/logical-replication.html
	parameters.Mandatory.Add("wal_level", "logical")

	// Always enable SSL/TLS.
	// PostgreSQL must be reloaded when changing this value.
	// - https://www.postgresql.org/docs/current/ssl-tcp.html
	parameters.Mandatory.Add("ssl", "on")
	parameters.Mandatory.Add("ssl_cert_file", "/pgconf/tls/tls.crt")
	parameters.Mandatory.Add("ssl_key_file", "/pgconf/tls/tls.key")
	parameters.Mandatory.Add("ssl_ca_file", "/pgconf/tls/ca.crt")

	// Just-in-Time compilation can degrade performance unexpectedly. Allow
	// users to enable it for appropriate workloads.
	// - https://www.postgresql.org/docs/current/jit.html
	parameters.Default.Add("jit", "off")

	// SCRAM-SHA-256 is preferred over MD5, but allow users to disable it when
	// necessary. PostgreSQL 10 is the first to support SCRAM-SHA-256, and
	// PostgreSQL 14 makes it the default.
	// - https://www.postgresql.org/docs/current/auth-password.html
	parameters.Default.Add("password_encryption", "scram-sha-256")

	// Explicitly use Postgres' default log directory. This value is relative to the "data_directory" parameter.
	//
	// Controllers look at this parameter to grant group-write S_IWGRP on the directory.
	// Postgres does not grant this permission on the directories it creates.
	//
	// TODO(logs): A better default would be *outside* the data directory.
	// This parameter needs to be configurable and documented before the default can change.
	//
	// PostgreSQL must be reloaded when changing this parameter.
	parameters.Default.Add("log_directory", "log")

	// Pod "securityContext.fsGroup" ensures processes and filesystems agree on a GID;
	// use the same permissions for group and owner.
	// This allows every process in the pod to read Postgres log files.
	//
	// S_IRUSR, S_IWUSR: (0600) enable owner read and write permissions
	// S_IRGRP, S_IWGRP: (0060) enable group read and write permissions.
	//
	// PostgreSQL must be reloaded when changing this value.
	parameters.Mandatory.Add("log_file_mode", "0660")

	return parameters
}

// Parameters is a pairing of ParameterSets.
type Parameters struct{ Mandatory, Default *ParameterSet }

// ParameterSet is a collection of PostgreSQL parameters.
// - https://www.postgresql.org/docs/current/config-setting.html
type ParameterSet struct {
	values map[string]string
}

// NewParameterSet returns an empty ParameterSet.
func NewParameterSet() *ParameterSet {
	return &ParameterSet{
		values: make(map[string]string),
	}
}

// AsMap returns a copy of ps as a map.
func (ps *ParameterSet) AsMap() map[string]string {
	if ps == nil {
		return nil
	}

	return maps.Clone(ps.values)
}

// DeepCopy returns a copy of ps.
func (ps *ParameterSet) DeepCopy() *ParameterSet {
	if ps == nil {
		return nil
	}

	return &ParameterSet{
		values: maps.Clone(ps.values),
	}
}

// Add sets parameter name to value.
func (ps *ParameterSet) Add(name, value string) {
	ps.values[ps.normalize(name)] = value
}

// AppendToList adds each value to the right-hand side of parameter name
// as a comma-separated list without quoting.
func (ps *ParameterSet) AppendToList(name string, value ...string) {
	result := ps.Value(name)

	if len(value) > 0 {
		if len(result) > 0 {
			result += "," + strings.Join(value, ",")
		} else {
			result = strings.Join(value, ",")
		}
	}

	ps.Add(name, result)
}

// Get returns the value of parameter name and whether or not it was present in ps.
func (ps *ParameterSet) Get(name string) (string, bool) {
	value, ok := ps.values[ps.normalize(name)]
	return value, ok
}

// Has returns whether or not parameter name is present in ps.
func (ps *ParameterSet) Has(name string) bool {
	_, ok := ps.Get(name)
	return ok
}

func (*ParameterSet) normalize(name string) string {
	// All parameter names are case-insensitive.
	// -- https://www.postgresql.org/docs/current/config-setting.html
	return strings.ToLower(name)
}

// Value returns empty string or the value of parameter name if it is present in ps.
func (ps *ParameterSet) Value(name string) string {
	value, _ := ps.Get(name)
	return value
}

func (ps *ParameterSet) String() string {
	keys := make([]string, 0, len(ps.values))
	for k := range ps.values {
		keys = append(keys, k)
	}

	slices.Sort(keys)

	var b strings.Builder
	for _, k := range keys {
		_, _ = fmt.Fprintf(&b, "%s = '%s'\n", k, escapeParameterQuotes(ps.values[k]))
	}
	return b.String()
}

// escapeParameterQuotes is used by [ParameterSet.String].
var escapeParameterQuotes = strings.NewReplacer(`'`, `''`).Replace
