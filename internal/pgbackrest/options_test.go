// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"
)

func TestMultiSet(t *testing.T) {
	t.Parallel()

	ms := iniMultiSet{}
	assert.Equal(t, ms.String(), "")
	assert.DeepEqual(t, ms.Values("any"), []string(nil))

	ms.Add("x", "y")
	assert.DeepEqual(t, ms.Values("x"), []string{"y"})

	ms.Add("x", "a")
	assert.DeepEqual(t, ms.Values("x"), []string{"y", "a"})

	ms.Add("abc", "j'l")
	assert.DeepEqual(t, ms, iniMultiSet{
		"x":   []string{"y", "a"},
		"abc": []string{"j'l"},
	})
	assert.Equal(t, ms.String(),
		"abc = j'l\nx = y\nx = a\n")

	ms.Set("x", "n")
	assert.DeepEqual(t, ms.Values("x"), []string{"n"})
	assert.Equal(t, ms.String(),
		"abc = j'l\nx = n\n")

	ms.Set("x", "p", "q")
	assert.DeepEqual(t, ms.Values("x"), []string{"p", "q"})

	t.Run("PrettyYAML", func(t *testing.T) {
		b, err := yaml.Marshal(iniMultiSet{
			"x": []string{"y"},
			"z": []string{""},
		}.String())

		assert.NilError(t, err)
		assert.Assert(t, strings.HasPrefix(string(b), `|`),
			"expected literal block scalar, got:\n%s", b)
	})
}

func TestSectionSet(t *testing.T) {
	t.Parallel()

	sections := iniSectionSet{}
	assert.Equal(t, sections.String(), "")

	sections["db"] = iniMultiSet{"x": []string{"y"}}
	assert.Equal(t, sections.String(),
		"\n[db]\nx = y\n")

	sections["db:backup"] = iniMultiSet{"x": []string{"w"}}
	assert.Equal(t, sections.String(),
		"\n[db]\nx = y\n\n[db:backup]\nx = w\n",
		"expected subcommand after its stanza")

	sections["another"] = iniMultiSet{"x": []string{"z"}}
	assert.Equal(t, sections.String(),
		"\n[another]\nx = z\n\n[db]\nx = y\n\n[db:backup]\nx = w\n",
		"expected alphabetical stanzas")

	sections["global"] = iniMultiSet{"x": []string{"t"}}
	assert.Equal(t, sections.String(),
		"\n[global]\nx = t\n\n[another]\nx = z\n\n[db]\nx = y\n\n[db:backup]\nx = w\n",
		"expected global before stanzas")

	sections["global:command"] = iniMultiSet{"t": []string{"v"}}
	assert.Equal(t, sections.String(),
		strings.Join([]string{
			"\n[global]\nx = t\n",
			"\n[global:command]\nt = v\n",
			"\n[another]\nx = z\n",
			"\n[db]\nx = y\n",
			"\n[db:backup]\nx = w\n",
		}, ""),
		"expected global subcommand after global")

	t.Run("PrettyYAML", func(t *testing.T) {
		sections["last"] = iniMultiSet{"z": []string{""}}
		b, err := yaml.Marshal(sections.String())

		assert.NilError(t, err)
		assert.Assert(t, strings.HasPrefix(string(b), `|`),
			"expected literal block scalar, got:\n%s", b)
	})
}
