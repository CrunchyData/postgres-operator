// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
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
	matches := func(actual []*HostBasedAuthentication, expected string) cmp.Comparison {
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
local    all          "postgres"      "peer"
hostssl  replication  "_crunchyrepl"  all   "cert"
hostssl  "postgres"   "_crunchyrepl"  all   "cert"
host     all          "_crunchyrepl"  all   "reject"
	`))
	assert.Assert(t, matches(hba.Default, `
hostssl  all  all  all  "scram-sha-256"
	`))
}

func TestHostBasedAuthentication(t *testing.T) {
	assert.Equal(t, `local all "postgres","pgo" "peer"`,
		NewHBA().Local().Users("postgres", "pgo").Method("peer").String())

	assert.Equal(t, `host all all "::1/128" "trust"`,
		NewHBA().TCP().Network("::1/128").Method("trust").String())

	assert.Equal(t, `host replication "KD6-3.7" samenet "scram-sha-256"`,
		NewHBA().TCP().SameNetwork().Replication().
			Users("KD6-3.7").Method("scram-sha-256").
			String())

	assert.Equal(t, `hostssl "data","bits" all all "md5"  "clientcert"="verify-ca"`,
		NewHBA().TLS().Databases("data", "bits").
			Method("md5").Options(map[string]string{"clientcert": "verify-ca"}).
			String())

	assert.Equal(t, `hostnossl all all all "reject"`,
		NewHBA().NoSSL().Method("reject").String())

	t.Run("OptionsSorted", func(t *testing.T) {
		assert.Equal(t, `hostssl all all all "ldap"  "ldapbasedn"="dc=example,dc=org" "ldapserver"="example.org"`,
			NewHBA().TLS().Method("ldap").Options(map[string]string{
				"ldapserver": "example.org",
				"ldapbasedn": "dc=example,dc=org",
			}).String())
	})

	t.Run("SpecialCharactersEscaped", func(t *testing.T) {
		// Databases; slash U+002F triggers regex escaping; regex characters themselves do not
		assert.Equal(t, `local "/^[/]asdf_[+][?]1234$","/^[/][*][$]$","+*$" all`,
			NewHBA().Local().Databases(`/asdf_+?1234`, `/*$`, `+*$`).String())

		// Users; slash U+002F triggers regex escaping; regex characters themselves do not
		assert.Equal(t, `local all "/^[/]asdf_[+][?]1234$","/^[/][*][$]$","+*$"`,
			NewHBA().Local().Users(`/asdf_+?1234`, `/*$`, `+*$`).String())
	})
}

func TestOrderedHBAs(t *testing.T) {
	ordered := new(OrderedHBAs)

	// The zero value is empty.
	assert.Equal(t, ordered.Length(), 0)
	assert.Assert(t, cmp.Len(ordered.AsStrings(), 0))

	// Append can be called without arguments.
	ordered.Append()
	ordered.AppendUnstructured()
	assert.Assert(t, cmp.Len(ordered.AsStrings(), 0))

	// Append adds to the end of the slice.
	ordered.Append(NewHBA())
	assert.Equal(t, ordered.Length(), 1)
	assert.DeepEqual(t, ordered.AsStrings(), []string{
		`all all all`,
	})

	// AppendUnstructured adds to the end of the slice.
	ordered.AppendUnstructured("could be anything, really")
	assert.Equal(t, ordered.Length(), 2)
	assert.DeepEqual(t, ordered.AsStrings(), []string{
		`all all all`,
		`could be anything, really`,
	})

	// Append and AppendUnstructured do not have a separate order.
	ordered.Append(NewHBA().Users("zoro"))
	assert.Equal(t, ordered.Length(), 3)
	assert.DeepEqual(t, ordered.AsStrings(), []string{
		`all all all`,
		`could be anything, really`,
		`all "zoro" all`,
	})

	t.Run("NilPointersIgnored", func(t *testing.T) {
		rules := new(OrderedHBAs)
		rules.Append(
			NewHBA(), nil,
			NewHBA(), nil,
		)
		assert.DeepEqual(t, rules.AsStrings(), []string{
			`all all all`,
			`all all all`,
		})
	})

	// See [internal/testing/validation.TestPostgresAuthenticationRules]
	t.Run("NoInclude", func(t *testing.T) {
		rules := new(OrderedHBAs)
		rules.AppendUnstructured(
			`one`,
			`include "/etc/passwd"`,
			`   include_dir /tmp`,
			`include_if_exists postgresql.auto.conf`,
			`two`,
		)
		assert.DeepEqual(t, rules.AsStrings(), []string{
			`one`,
			`two`,
		})
	})

	t.Run("SpecialCharactersStripped", func(t *testing.T) {
		rules := new(OrderedHBAs)
		rules.AppendUnstructured(
			" \n\t things \n\n\n",
			`with # comment`,
			" \n\t \\\\ \f", // entirely special characters
			`trailing slashes \\\`,
			"multiple \\\n lines okay",
		)
		assert.DeepEqual(t, rules.AsStrings(), []string{
			`things`,
			`with # comment`,
			`trailing slashes`,
			"multiple \\\n lines okay",
		})
	})
}
