// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package postgres

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/internal/testing/cmp"
)

func TestNewHBAs(t *testing.T) {
	matches := func(actual []HostBasedAuthentication, expected string) cmp.Comparison {
		printed := make([]string, len(actual))
		for i := range actual {
			printed[i] = actual[i].String()
		}

		parsed := strings.Split(strings.Trim(expected, "\t\n"), "\n")
		for i := range parsed {
			parsed[i] = strings.Join(strings.Fields(parsed[i]), " ")
		}

		return cmp.DeepEqual(printed, parsed)
	}

	hba := NewHBAs()
	assert.Assert(t, matches(hba.Mandatory, `
local    all          "postgres"      peer
hostssl  replication  "_crunchyrepl"  all   cert
hostssl  "postgres"   "_crunchyrepl"  all   cert
host     all          "_crunchyrepl"  all   reject
	`))
	assert.Assert(t, matches(hba.Default, `
hostssl  all  all  all  md5
	`))
}

func TestHostBasedAuthentication(t *testing.T) {
	assert.Equal(t, `local all "postgres" peer`,
		NewHBA().Local().User("postgres").Method("peer").String())

	assert.Equal(t, `host all all "::1/128" trust`,
		NewHBA().TCP().Network("::1/128").Method("trust").String())

	assert.Equal(t, `host replication "KD6-3.7" samenet scram-sha-256`,
		NewHBA().TCP().SameNetwork().Replication().
			User("KD6-3.7").Method("scram-sha-256").
			String())

	assert.Equal(t, `hostssl "data" +"admin" all md5  clientcert="verify-ca"`,
		NewHBA().TLS().Database("data").Role("admin").
			Method("md5").Options(map[string]string{"clientcert": "verify-ca"}).
			String())

	assert.Equal(t, `hostnossl all all all reject`,
		NewHBA().NoSSL().Method("reject").String())
}
