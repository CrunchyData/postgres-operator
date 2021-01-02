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

// This file should one day be refactored into a library with other alike tools
//

import (
	"math"
	"math/rand" // Good enough for jitter purposes, don't need crypto/rand
	"time"
)

// Jitter is an enum representing a distinct jitter mode
type Jitter int

const (
	// JitterNone performs no Jitter, with multiple clients, can be bursty
	JitterNone Jitter = iota
	// JitterFull represents a jitter range of (0, Duration)
	JitterFull
	// JitterCenter represents a jitter range of (0.5 Duration, 1.5 Duration)
	// That is, full, but centered on the value
	JitterCenter
	// JitterSmall represents a jitter range of 0.75 Duration, 1.25 Duration)
	JitterSmall
)

// Apply provides a new time with respect to t based on the jitter mode
// #nosec: G404
func (jm Jitter) Apply(t time.Duration) time.Duration {
	switch jm {
	case JitterNone: // being explicit in case default case changes
		return t
	case JitterFull:
		return time.Duration(rand.Float64() * float64(t))
	case JitterCenter:
		return time.Duration(float64(t/2) + (rand.Float64() * float64(t)))
	case JitterSmall:
		return time.Duration(float64(3*t/4) + (rand.Float64() * float64(t) / 2))
	default:
		return t
	}
}

// Backoff interface provides increasing length delays for event spacing
type Backoff interface {
	Duration(round int) time.Duration
}

// SpecificBackoffPolicy allows manually specifying retry times
type SpecificBackoffPolicy struct {
	Times      []time.Duration
	JitterMode Jitter
}

func (sbp SpecificBackoffPolicy) Duration(n int) time.Duration {
	if l := len(sbp.Times); sbp.Times == nil || n < 0 || l == 0 {
		return time.Duration(0)
	} else if n >= l {
		n = l - 1
	}

	return sbp.JitterMode.Apply(sbp.Times[n])
}

// ExponentialBackoffPolicy provides an exponential backoff based on:
// Base * (Ratio ^ Iteration)
//
// For example a base of 10ms, ratio of 2, and no jitter would produce:
// 10ms, 20ms, 40ms, 80ms, 160ms, 320ms, 640ms, 1.28s, 2.56s...
//
type ExponentialBackoffPolicy struct {
	Ratio      float64
	Base       time.Duration
	Maximum    time.Duration
	JitterMode Jitter
}

func (cbp ExponentialBackoffPolicy) Duration(n int) time.Duration {
	d := time.Duration(math.Pow(cbp.Ratio, float64(n)) * float64(cbp.Base))

	if j := cbp.JitterMode.Apply(d); cbp.Maximum > 0 && j > cbp.Maximum {
		return cbp.Maximum
	} else {
		return j
	}
}
