/*
 Copyright 2021 Crunchy Data Solutions, Inc.
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

package postgrescluster

import (
	"errors"
	"fmt"
	"io"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestSafeHash32(t *testing.T) {
	expected := errors.New("whomp")

	_, err := safeHash32(func(io.Writer) error { return expected })
	assert.Equal(t, err, expected)

	stuff, err := safeHash32(func(w io.Writer) error {
		_, _ = w.Write([]byte(`some stuff`))
		return nil
	})
	assert.NilError(t, err)
	assert.Equal(t, stuff, "574b4c7d87", "expected alphanumeric")

	same, err := safeHash32(func(w io.Writer) error {
		_, _ = w.Write([]byte(`some stuff`))
		return nil
	})
	assert.NilError(t, err)
	assert.Equal(t, same, stuff, "expected deterministic hash")
}

func TestUpdateReconcileResult(t *testing.T) {

	testCases := []struct {
		currResult           reconcile.Result
		newResult            reconcile.Result
		requeueExpected      bool
		expectedRequeueAfter time.Duration
	}{{
		currResult:           reconcile.Result{},
		newResult:            reconcile.Result{},
		requeueExpected:      false,
		expectedRequeueAfter: 0,
	}, {
		currResult:           reconcile.Result{Requeue: false},
		newResult:            reconcile.Result{Requeue: true},
		requeueExpected:      true,
		expectedRequeueAfter: 0,
	}, {
		currResult:           reconcile.Result{Requeue: true},
		newResult:            reconcile.Result{Requeue: false},
		requeueExpected:      true,
		expectedRequeueAfter: 0,
	}, {
		currResult:           reconcile.Result{Requeue: true},
		newResult:            reconcile.Result{Requeue: true},
		requeueExpected:      true,
		expectedRequeueAfter: 0,
	}, {
		currResult:           reconcile.Result{Requeue: false},
		newResult:            reconcile.Result{Requeue: false},
		requeueExpected:      false,
		expectedRequeueAfter: 0,
	}, {
		currResult:           reconcile.Result{},
		newResult:            reconcile.Result{RequeueAfter: 5 * time.Second},
		requeueExpected:      false,
		expectedRequeueAfter: 5 * time.Second,
	}, {
		currResult:           reconcile.Result{RequeueAfter: 5 * time.Second},
		newResult:            reconcile.Result{},
		requeueExpected:      false,
		expectedRequeueAfter: 5 * time.Second,
	}, {
		currResult:           reconcile.Result{RequeueAfter: 1 * time.Second},
		newResult:            reconcile.Result{RequeueAfter: 5 * time.Second},
		requeueExpected:      false,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult:           reconcile.Result{RequeueAfter: 5 * time.Second},
		newResult:            reconcile.Result{RequeueAfter: 1 * time.Second},
		requeueExpected:      false,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult:           reconcile.Result{RequeueAfter: 5 * time.Second},
		newResult:            reconcile.Result{RequeueAfter: 5 * time.Second},
		requeueExpected:      false,
		expectedRequeueAfter: 5 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		},
		newResult: reconcile.Result{
			Requeue: true, RequeueAfter: 1 * time.Second,
		},
		requeueExpected:      true,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: true, RequeueAfter: 1 * time.Second,
		},
		newResult: reconcile.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		},
		requeueExpected:      true,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: false, RequeueAfter: 1 * time.Second,
		},
		newResult: reconcile.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		},
		requeueExpected:      true,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: true, RequeueAfter: 1 * time.Second,
		},
		newResult: reconcile.Result{
			Requeue: false, RequeueAfter: 5 * time.Second,
		},
		requeueExpected:      true,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: false, RequeueAfter: 5 * time.Second,
		},
		newResult: reconcile.Result{
			Requeue: false, RequeueAfter: 1 * time.Second,
		},
		requeueExpected:      false,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: false, RequeueAfter: 1 * time.Second,
		},
		newResult: reconcile.Result{
			Requeue: false, RequeueAfter: 5 * time.Second,
		},
		requeueExpected:      false,
		expectedRequeueAfter: 1 * time.Second,
	}, {
		currResult: reconcile.Result{},
		newResult: reconcile.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		},
		requeueExpected:      true,
		expectedRequeueAfter: 5 * time.Second,
	}, {
		currResult: reconcile.Result{
			Requeue: true, RequeueAfter: 5 * time.Second,
		},
		newResult:            reconcile.Result{},
		requeueExpected:      true,
		expectedRequeueAfter: 5 * time.Second,
	}}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("curr: %v, new: %v", tc.currResult, tc.newResult), func(t *testing.T) {
			result := updateReconcileResult(tc.currResult, tc.newResult)
			assert.Assert(t, result.Requeue == tc.requeueExpected)
			assert.Assert(t, result.RequeueAfter == tc.expectedRequeueAfter)
		})
	}
}
