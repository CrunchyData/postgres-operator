/*
Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

package runtime

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// ErrorWithBackoff returns a Result and error that indicate a non-nil err
// should be logged and measured and its [reconcile.Request] should be retried
// later. When err is nil, nothing is logged and the Request is not retried.
// When err unwraps to [reconcile.TerminalError], the Request is not retried.
func ErrorWithBackoff(err error) (reconcile.Result, error) {
	// Result should be zero to avoid warning messages.
	return reconcile.Result{}, err

	// When error is not nil and not a TerminalError, the controller-runtime Controller
	// puts [reconcile.Request] back into the workqueue using AddRateLimited.
	// - https://github.com/kubernetes-sigs/controller-runtime/blob/v0.18.4/pkg/internal/controller/controller.go#L317
	// - https://pkg.go.dev/k8s.io/client-go/util/workqueue#RateLimitingInterface
}

// ErrorWithoutBackoff returns a Result and error that indicate a non-nil err
// should be logged and measured without retrying its [reconcile.Request].
// When err is nil, nothing is logged and the Request is not retried.
func ErrorWithoutBackoff(err error) (reconcile.Result, error) {
	if err != nil {
		err = reconcile.TerminalError(err)
	}

	// Result should be zero to avoid warning messages.
	return reconcile.Result{}, err

	// When error is a TerminalError, the controller-runtime Controller increments
	// a counter rather than interact with the workqueue.
	// - https://github.com/kubernetes-sigs/controller-runtime/blob/v0.18.4/pkg/internal/controller/controller.go#L314
}

// RequeueWithBackoff returns a Result that indicates a [reconcile.Request]
// should be retried later.
func RequeueWithBackoff() reconcile.Result {
	return reconcile.Result{Requeue: true}

	// When [reconcile.Result].Requeue is true, the controller-runtime Controller
	// puts [reconcile.Request] back into the workqueue using AddRateLimited.
	// - https://github.com/kubernetes-sigs/controller-runtime/blob/v0.18.4/pkg/internal/controller/controller.go#L334
	// - https://pkg.go.dev/k8s.io/client-go/util/workqueue#RateLimitingInterface
}

// RequeueWithoutBackoff returns a Result that indicates a [reconcile.Request]
// should be retried on or before delay.
func RequeueWithoutBackoff(delay time.Duration) reconcile.Result {
	// RequeueAfter must be positive to not backoff.
	if delay <= 0 {
		delay = time.Nanosecond
	}

	// RequeueAfter implies Requeue, but set both to remove any ambiguity.
	return reconcile.Result{Requeue: true, RequeueAfter: delay}

	// When [reconcile.Result].RequeueAfter is positive, the controller-runtime Controller
	// puts [reconcile.Request] back into the workqueue using AddAfter.
	// - https://github.com/kubernetes-sigs/controller-runtime/blob/v0.18.4/pkg/internal/controller/controller.go#L325
	// - https://pkg.go.dev/k8s.io/client-go/util/workqueue#DelayingInterface
}
