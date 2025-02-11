// Copyright 2022 - 2025 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package v1beta1

import (
	"reflect"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/yaml"
)

func TestDurationYAML(t *testing.T) {
	t.Parallel()

	t.Run("Zero", func(t *testing.T) {
		zero, err := yaml.Marshal(Duration{})
		assert.NilError(t, err)
		assert.DeepEqual(t, zero, []byte(`"0"`+"\n"))

		var parsed Duration
		assert.NilError(t, yaml.Unmarshal(zero, &parsed))
		assert.Equal(t, parsed.AsDuration().Duration, 0*time.Second)
	})

	t.Run("Small", func(t *testing.T) {
		var parsed Duration
		assert.NilError(t, yaml.Unmarshal([]byte(`3ns`), &parsed))
		assert.Equal(t, parsed.AsDuration().Duration, 3*time.Nanosecond)

		b, err := yaml.Marshal(parsed)
		assert.NilError(t, err)
		assert.DeepEqual(t, b, []byte(`3ns`+"\n"))
	})

	t.Run("Large", func(t *testing.T) {
		var parsed Duration
		assert.NilError(t, yaml.Unmarshal([]byte(`52 weeks`), &parsed))
		assert.Equal(t, parsed.AsDuration().Duration, 364*24*time.Hour)

		b, err := yaml.Marshal(parsed)
		assert.NilError(t, err)
		assert.DeepEqual(t, b, []byte(`52 weeks`+"\n"))
	})

	t.Run("UnitsIn", func(t *testing.T) {
		for _, tt := range []struct {
			input  string
			result time.Duration
		}{
			// These can be unmarshaled:
			{"1 ns", time.Nanosecond},
			{"2 nano", 2 * time.Nanosecond},
			{"3 nanos", 3 * time.Nanosecond},
			{"4 nanosec", 4 * time.Nanosecond},
			{"5 nanosecs", 5 * time.Nanosecond},
			{"6 nanopants", 6 * time.Nanosecond},

			{"1 us", time.Microsecond},
			{"2 µs", 2 * time.Microsecond},
			{"3 micro", 3 * time.Microsecond},
			{"4 micros", 4 * time.Microsecond},
			{"5 micrometer", 5 * time.Microsecond},

			{"1 ms", time.Millisecond},
			{"2 milli", 2 * time.Millisecond},
			{"3 millis", 3 * time.Millisecond},
			{"4 millisec", 4 * time.Millisecond},
			{"5 millisecs", 5 * time.Millisecond},
			{"6 millipede", 6 * time.Millisecond},

			{"1s", time.Second},
			{"2 sec", 2 * time.Second},
			{"3 secs", 3 * time.Second},
			{"4 seconds", 4 * time.Second},
			{"5 security", 5 * time.Second},

			{"1m", time.Minute},
			{"2 min", 2 * time.Minute},
			{"3 mins", 3 * time.Minute},
			{"4 minutia", 4 * time.Minute},
			{"5 mininture", 5 * time.Minute},

			{"1h", time.Hour},
			{"2 hr", 2 * time.Hour},
			{"3 hour", 3 * time.Hour},
			{"4 hours", 4 * time.Hour},
			{"5 hourglass", 5 * time.Hour},

			{"1d", 24 * time.Hour},
			{"2 day", 2 * 24 * time.Hour},
			{"3 days", 3 * 24 * time.Hour},
			{"4 dayrock", 4 * 24 * time.Hour},

			{"1w", 7 * 24 * time.Hour},
			{"2 wk", 2 * 7 * 24 * time.Hour},
			{"3 week", 3 * 7 * 24 * time.Hour},
			{"4 weeks", 4 * 7 * 24 * time.Hour},
			{"5 weekpasta", 5 * 7 * 24 * time.Hour},
		} {
			var parsed Duration
			assert.NilError(t, yaml.Unmarshal([]byte(tt.input), &parsed))
			assert.Equal(t, parsed.AsDuration().Duration, tt.result)
		}

		for _, tt := range []string{
			// These cannot be unmarshaled:
			"1 nss",
			"2 uss",
			"3 usec",
			"4 usecs",
			"5 µsec",
			"6 mss",
			"7 hs",
			"8 hrs",
			"9 ds",
			"10 ws",
			"11 wks",
		} {
			assert.ErrorContains(t,
				yaml.Unmarshal([]byte(tt), new(Duration)), "unable to parse")
		}
	})
}

func TestSchemalessObjectDeepCopy(t *testing.T) {
	t.Parallel()

	var z SchemalessObject
	assert.DeepEqual(t, z, z.DeepCopy())

	var one SchemalessObject
	assert.NilError(t, yaml.Unmarshal(
		[]byte(`{ str: value, num: 1, arr: [a, 2, true] }`), &one,
	))

	// reflect and go-cmp agree the original and copy are equivalent.
	same := one.DeepCopy()
	assert.DeepEqual(t, one, same)
	assert.Assert(t, reflect.DeepEqual(one, same))

	// Changes to the copy do not affect the original.
	{
		change := one.DeepCopy()
		change["str"] = "banana"
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
	{
		change := one.DeepCopy()
		change["num"] = 99
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
	{
		change := one.DeepCopy()
		change["arr"].([]any)[0] = "rock"
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
	{
		change := one.DeepCopy()
		change["arr"] = append(change["arr"].([]any), "more")
		assert.Assert(t, reflect.DeepEqual(one, same))
		assert.Assert(t, !reflect.DeepEqual(one, change))
	}
}
