// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"fmt"
	"strings"
)

// NewHBAs returns HostBasedAuthentication records required by this package.
func NewHBAs() HBAs {
	return HBAs{
		Mandatory: []HostBasedAuthentication{
			// The "postgres" superuser must always be able to connect locally.
			*NewHBA().Local().User("postgres").Method("peer"),

			// The replication user must always connect over TLS using certificate
			// authentication. Patroni also connects to the "postgres" database
			// when calling `pg_rewind`.
			// - https://www.postgresql.org/docs/current/warm-standby.html#STREAMING-REPLICATION-AUTHENTICATION
			*NewHBA().TLS().User(ReplicationUser).Method("cert").Replication(),
			*NewHBA().TLS().User(ReplicationUser).Method("cert").Database("postgres"),
			*NewHBA().TCP().User(ReplicationUser).Method("reject"),
		},

		Default: []HostBasedAuthentication{
			// Allow TLS connections to any database using passwords. The "md5"
			// authentication method automatically verifies passwords encrypted
			// using either MD5 or SCRAM-SHA-256.
			// - https://www.postgresql.org/docs/current/auth-password.html
			*NewHBA().TLS().Method("md5"),
		},
	}
}

// HBAs is a pairing of HostBasedAuthentication records.
type HBAs struct{ Mandatory, Default []HostBasedAuthentication }

// HostBasedAuthentication represents a single record for pg_hba.conf.
// - https://www.postgresql.org/docs/current/auth-pg-hba-conf.html
type HostBasedAuthentication struct {
	origin, database, user, address, method, options string
}

// NewHBA returns an HBA record that matches all databases, networks, and users.
func NewHBA() *HostBasedAuthentication {
	return new(HostBasedAuthentication).AllDatabases().AllNetworks().AllUsers()
}

func (HostBasedAuthentication) quote(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

// AllDatabases makes hba match connections made to any database.
func (hba *HostBasedAuthentication) AllDatabases() *HostBasedAuthentication {
	hba.database = "all"
	return hba
}

// AllNetworks makes hba match connection attempts made from any IP address.
func (hba *HostBasedAuthentication) AllNetworks() *HostBasedAuthentication {
	hba.address = "all"
	return hba
}

// AllUsers makes hba match connections made by any user.
func (hba *HostBasedAuthentication) AllUsers() *HostBasedAuthentication {
	hba.user = "all"
	return hba
}

// Database makes hba match connections made to a specific database.
func (hba *HostBasedAuthentication) Database(name string) *HostBasedAuthentication {
	hba.database = hba.quote(name)
	return hba
}

// Local makes hba match connection attempts using Unix-domain sockets.
func (hba *HostBasedAuthentication) Local() *HostBasedAuthentication {
	hba.origin = "local"
	return hba
}

// Method specifies the authentication method to use when a connection matches hba.
func (hba *HostBasedAuthentication) Method(name string) *HostBasedAuthentication {
	hba.method = name
	return hba
}

// Network makes hba match connection attempts from a block of IP addresses in CIDR notation.
func (hba *HostBasedAuthentication) Network(block string) *HostBasedAuthentication {
	hba.address = hba.quote(block)
	return hba
}

// NoSSL makes hba match connection attempts made over TCP/IP without SSL.
func (hba *HostBasedAuthentication) NoSSL() *HostBasedAuthentication {
	hba.origin = "hostnossl"
	return hba
}

// Options specifies any options for the authentication method.
func (hba *HostBasedAuthentication) Options(opts map[string]string) *HostBasedAuthentication {
	hba.options = ""
	for k, v := range opts {
		hba.options = fmt.Sprintf("%s %s=%s", hba.options, k, hba.quote(v))
	}
	return hba
}

// Replication makes hba match physical replication connections.
func (hba *HostBasedAuthentication) Replication() *HostBasedAuthentication {
	hba.database = "replication"
	return hba
}

// Role makes hba match connections by users that are members of a specific role.
func (hba *HostBasedAuthentication) Role(name string) *HostBasedAuthentication {
	hba.user = "+" + hba.quote(name)
	return hba
}

// SameNetwork makes hba match connection attempts from IP addresses in any
// subnet to which the server is directly connected.
func (hba *HostBasedAuthentication) SameNetwork() *HostBasedAuthentication {
	hba.address = "samenet"
	return hba
}

// TLS makes hba match connection attempts made using TCP/IP with TLS.
func (hba *HostBasedAuthentication) TLS() *HostBasedAuthentication {
	hba.origin = "hostssl"
	return hba
}

// TCP makes hba match connection attempts made using TCP/IP, with or without SSL.
func (hba *HostBasedAuthentication) TCP() *HostBasedAuthentication {
	hba.origin = "host"
	return hba
}

// User makes hba match connections by a specific user.
func (hba *HostBasedAuthentication) User(name string) *HostBasedAuthentication {
	hba.user = hba.quote(name)
	return hba
}

// String returns hba formatted for the pg_hba.conf file without a newline.
func (hba HostBasedAuthentication) String() string {
	if hba.origin == "local" {
		return strings.TrimSpace(fmt.Sprintf("local %s %s %s %s",
			hba.database, hba.user, hba.method, hba.options))
	}

	return strings.TrimSpace(fmt.Sprintf("%s %s %s %s %s %s",
		hba.origin, hba.database, hba.user, hba.address, hba.method, hba.options))
}
