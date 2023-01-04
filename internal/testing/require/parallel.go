/*
 Copyright 2022 - 2023 Crunchy Data Solutions, Inc.
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
