package pgadmin

/*
 Copyright 2020 Crunchy Data Solutions, Inc.
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
	Test int
}

func TestDoubleExp(t *testing.T) {
	bp := ExponentialBackoffPolicy{
		Base:  10 * time.Millisecond,
		Ratio: 2,
	}
	cases := []testPair{
		{Test: 0, Exp: 10 * time.Millisecond},
		{Test: 1, Exp: 20 * time.Millisecond},
		{Test: 2, Exp: 40 * time.Millisecond},
		{Test: 3, Exp: 80 * time.Millisecond},
		{Test: 4, Exp: 160 * time.Millisecond},
		{Test: 5, Exp: 320 * time.Millisecond},
		{Test: 6, Exp: 640 * time.Millisecond},
		{Test: 7, Exp: 1280 * time.Millisecond},
		{Test: 8, Exp: 2560 * time.Millisecond},
		{Test: 9, Exp: 5120 * time.Millisecond},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Test); res != tCase.Exp {
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
		{Test: 6, Exp: 640 * time.Millisecond},
		{Test: 7, Exp: limit},
		{Test: 8, Exp: limit},
		{Test: 9, Exp: limit},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Test); res != tCase.Exp {
			t.Logf("Expected %v, Got %v", tCase.Exp, res)
			t.Fail()
		}
	}
}

func TestSubscripts(t *testing.T) {

}

func TestUniformPolicy(t *testing.T) {
	bp := SpecificBackoffPolicy{
		Times: []time.Duration{
			8 * time.Second,
		},
	}

	cases := []testPair{
		{Test: 0, Exp: 8 * time.Second},
		{Test: 1, Exp: 8 * time.Second},
		{Test: 2, Exp: 8 * time.Second},
		{Test: 3, Exp: 8 * time.Second},
		{Test: 4, Exp: 8 * time.Second},
		{Test: 5, Exp: 8 * time.Second},
		{Test: 6, Exp: 8 * time.Second},
		{Test: 7, Exp: 8 * time.Second},
		{Test: 8, Exp: 8 * time.Second},
		{Test: 9, Exp: 8 * time.Second},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Test); res != tCase.Exp {
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
		{Test: 0, Exp: 1 * time.Millisecond},
		{Test: 1, Exp: 1 * time.Millisecond},
		{Test: 2, Exp: 2 * time.Millisecond},
		{Test: 3, Exp: 3 * time.Millisecond},
		{Test: 4, Exp: 5 * time.Millisecond},
		{Test: 5, Exp: 8 * time.Millisecond},
		{Test: 6, Exp: 13 * time.Millisecond},
		{Test: 7, Exp: 21 * time.Millisecond},
		{Test: 8, Exp: 33 * time.Millisecond},
		{Test: 9, Exp: 54 * time.Millisecond},
	}

	for _, tCase := range cases {
		if res := bp.Duration(tCase.Test); res != tCase.Exp {
			t.Logf("Expected %v, Got %v", tCase.Exp, res)
			t.Fail()
		}
	}
}

func TestJitterFull(t *testing.T) {
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

func TestJitterCenter(t *testing.T) {
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

func TestJitterSmall(t *testing.T) {
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
