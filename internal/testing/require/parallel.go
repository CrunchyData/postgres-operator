// Copyright 2022 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package require

import (
	"sync"
	"testing"
)

var capacity sync.Mutex

// ParallelCapacity calls t.Parallel then waits for needed capacity. There is
// no wait when needed is zero.
func ParallelCapacity(t *testing.T, needed int) {
	t.Helper()
	t.Parallel()

	if needed > 0 {
		// Assume capacity of one; allow only one caller at a time.
		// TODO: actually track how much capacity is available.
		capacity.Lock()
		t.Cleanup(capacity.Unlock)
	}
}
