// Copyright 2021 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"fmt"
	"sort"
	"strings"
)

// iniMultiSet represents the key-value pairs in a pgBackRest config file section.
type iniMultiSet map[string][]string

func (ms iniMultiSet) String() string {
	keys := make([]string, 0, len(ms))
	for k := range ms {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		for _, v := range ms[k] {
			if len(v) <= 0 {
				_, _ = fmt.Fprintf(&b, "%s =\n", k)
			} else {
				_, _ = fmt.Fprintf(&b, "%s = %s\n", k, v)
			}
		}
	}
	return b.String()
}

// Add associates value with key, appending it to any values already associated
// with key. The key is case-sensitive.
func (ms iniMultiSet) Add(key, value string) {
	ms[key] = append(ms[key], value)
}

// Set replaces the values associated with key. The key is case-sensitive.
func (ms iniMultiSet) Set(key string, values ...string) {
	ms[key] = make([]string, len(values))
	copy(ms[key], values)
}

// Values returns all values associated with the given key.
// The key is case-sensitive. The returned slice is not a copy.
func (ms iniMultiSet) Values(key string) []string {
	return ms[key]
}

// iniSectionSet represents the different sections in a pgBackRest config file.
type iniSectionSet map[string]iniMultiSet

func (sections iniSectionSet) String() string {
	global := make([]string, 0, len(sections))
	stanza := make([]string, 0, len(sections))

	for k := range sections {
		if k == "global" || strings.HasPrefix(k, "global:") {
			global = append(global, k)
		} else {
			stanza = append(stanza, k)
		}
	}

	sort.Strings(global)
	sort.Strings(stanza)

	var b strings.Builder
	for _, k := range global {
		_, _ = fmt.Fprintf(&b, "\n[%s]\n%s", k, sections[k])
	}
	for _, k := range stanza {
		_, _ = fmt.Fprintf(&b, "\n[%s]\n%s", k, sections[k])
	}
	return b.String()
}
