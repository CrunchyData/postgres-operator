package pgadmin

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

      http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
*/

import (
	"testing"
	"time"
)

type testPair struct {
	Exp  time.Duration
	Iter int
}

func TestDoubleExp(t *testing.T) {
	bp := ExponentialBackoffPolicy{
		Base:  10 * time.Millisecond,
		Ratio: 2,
	}
	cases := []testPair{
		{Iter: 0, Exp: 10 * time.Millisecond},
		{Iter: 1, Exp: 20 * time.Millisecond},
		{Iter: 2, Exp: 40 * time.Millisecond},
		{Iter: 3, Exp: 80 * time.Millisecond},
		{Iter: 4, Exp: 160 * time.Millisecond},
		{Iter: 5, Exp: 320 * time.Millisecond},
		{Iter: 6, Exp: 640 * time.Millisecond},
		{Iter: 7, Exp: 1280 * time.Millisecond},
		{Iter: 8, Exp: 2560 * time.Millisecond},
		{Iter: 9, Exp: 5120 * time.Millisecond},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Iter); res != tCase.Exp {
			t.Logf("Expected %v, Got %v", tCase.Exp, res)
			t.Fail()
		}
	}
}

func TestCalcMax(t *testing.T) {
	const limit = 1279 * time.Millisecond
	bp := ExponentialBackoffPolicy{
		Base:    10 * time.Millisecond,
		Ratio:   2,
		Maximum: limit,
	}
	cases := []testPair{
		{Iter: 6, Exp: 640 * time.Millisecond},
		{Iter: 7, Exp: limit},
		{Iter: 8, Exp: limit},
		{Iter: 9, Exp: limit},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Iter); res != tCase.Exp {
			t.Logf("Expected %v, Got %v", tCase.Exp, res)
			t.Fail()
		}
	}
}

func TestSubscripts(t *testing.T) {
	cases := []struct {
		label string
		iter  int
		pol   SpecificBackoffPolicy
	}{
		{
			label: "nil",
			iter:  0,
			pol:   SpecificBackoffPolicy{},
		},
		{
			label: "zerolen",
			iter:  0,
			pol: SpecificBackoffPolicy{
				Times: []time.Duration{},
			},
		},
		{
			label: "negative",
			iter:  -42,
			pol: SpecificBackoffPolicy{
				Times: []time.Duration{
					9 * time.Second,
				},
			},
		},
	}

	for _, tCase := range cases {
		if d := tCase.pol.Duration(tCase.iter); d != 0 {
			t.Logf("Expected 0 from case, got %v", d)
			t.Fail()
		}
	}
}

func TestUniformPolicy(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			8 * time.Second,
		},
	}

	cases := []testPair{
		{Iter: 0, Exp: 8 * time.Second},
		{Iter: 1, Exp: 8 * time.Second},
		{Iter: 2, Exp: 8 * time.Second},
		{Iter: 3, Exp: 8 * time.Second},
		{Iter: 4, Exp: 8 * time.Second},
		{Iter: 5, Exp: 8 * time.Second},
		{Iter: 6, Exp: 8 * time.Second},
		{Iter: 7, Exp: 8 * time.Second},
		{Iter: 8, Exp: 8 * time.Second},
		{Iter: 9, Exp: 8 * time.Second},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Iter); res != tCase.Exp {
			t.Logf("Expected %v, Got %v", tCase.Exp, res)
			t.Fail()
		}
	}
}

func TestStatedPolicy(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			1 * time.Millisecond,
			1 * time.Millisecond,
			2 * time.Millisecond,
			3 * time.Millisecond,
			5 * time.Millisecond,
			8 * time.Millisecond,
			13 * time.Millisecond,
			21 * time.Millisecond,
			33 * time.Millisecond,
			54 * time.Millisecond,
		},
	}

	cases := []testPair{
		{Iter: 0, Exp: 1 * time.Millisecond},
		{Iter: 1, Exp: 1 * time.Millisecond},
		{Iter: 2, Exp: 2 * time.Millisecond},
		{Iter: 3, Exp: 3 * time.Millisecond},
		{Iter: 4, Exp: 5 * time.Millisecond},
		{Iter: 5, Exp: 8 * time.Millisecond},
		{Iter: 6, Exp: 13 * time.Millisecond},
		{Iter: 7, Exp: 21 * time.Millisecond},
		{Iter: 8, Exp: 33 * time.Millisecond},
		{Iter: 9, Exp: 54 * time.Millisecond},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Iter); res != tCase.Exp {
			t.Logf("Expected %v, Got %v", tCase.Exp, res)
			t.Fail()
		}
	}
}

func TestJitterFullLimits(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			10 * time.Second,
		},
		JitterMode: JitterFull,
	}

	for i := 0; i < 1000; i++ {
		if d := bp.Duration(i); d < 0 || d > 10*time.Second {
			t.Fatalf("On iteration %d, found unexpected value: %v\n", i, d)
		}
	}
}

func TestJitterFullExtents(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			10 * time.Second,
		},
		JitterMode: JitterFull,
	}

	var nearLow, nearHigh bool
	for i := 0; i < 1000; i++ {
		// See if we've had at least one value near the low limit
		if d := bp.Duration(i); !nearLow && d < 250*time.Millisecond {
			nearLow = true
		}
		// See if we've had at least one value near the high limit
		if d := bp.Duration(i); !nearHigh && d > 9750*time.Millisecond {
			nearHigh = true
		}
	}
	if !(nearLow && nearHigh) {
		t.Fatalf("Expected generated values near edges: near low [%t], near high [%t]", nearLow, nearHigh)
	}
}

func TestJitterCenterLimits(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			20 * time.Second,
		},
		JitterMode: JitterCenter,
	}

	for i := 0; i < 1000; i++ {
		if d := bp.Duration(i); d < 10*time.Second || d > 30*time.Second {
			t.Fatalf("On iteration %d, found unexpected value: %v\n", i, d)
		}
	}
}

func TestJitterCenterExtents(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			20 * time.Second,
		},
		JitterMode: JitterCenter,
	}

	var nearLow, nearHigh bool
	for i := 0; i < 1000; i++ {
		// See if we've had at least one value near the low limit
		if d := bp.Duration(i); !nearLow && d < 10250*time.Millisecond {
			nearLow = true
		}
		// See if we've had at least one value near the high limit
		if d := bp.Duration(i); !nearHigh && d > 29750*time.Millisecond {
			nearHigh = true
		}
	}
	if !(nearLow && nearHigh) {
		t.Fatalf("Expected generated values near edges: near low [%t], near high [%t]", nearLow, nearHigh)
	}
}

func TestJitterSmallLimits(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			20 * time.Second,
		},
		JitterMode: JitterSmall,
	}

	for i := 0; i < 1000; i++ {
		if d := bp.Duration(i); d < 15*time.Second || d > 25*time.Second {
			t.Fatalf("On iteration %d, found unexpected value: %v\n", i, d)
		}
	}
}

func TestJitterSmallExtents(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			20 * time.Second,
		},
		JitterMode: JitterSmall,
	}

	var nearLow, nearHigh bool
	for i := 0; i < 1000; i++ {
		// See if we've had at least one value near the low limit
		if d := bp.Duration(i); !nearLow && d < 15250*time.Millisecond {
			nearLow = true
		}
		// See if we've had at least one value near the high limit
		if d := bp.Duration(i); !nearHigh && d > 24750*time.Millisecond {
			nearHigh = true
		}
	}
	if !(nearLow && nearHigh) {
		t.Fatalf("Expected generated values near edges: near low [%t], near high [%t]", nearLow, nearHigh)
	}
}
