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

package events

import (
	"fmt"
	"testing"
	"time"

	"gotest.tools/v3/assert"
	corev1 "k8s.io/api/core/v1"
	eventsv1 "k8s.io/api/events/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/tools/record/util"
	"k8s.io/client-go/tools/reference"
)

// Recorder implements the interface for the deprecated v1.Event API.
// The zero value discards events.
// - https://pkg.go.dev/k8s.io/client-go@v0.24.1/tools/record#EventRecorder
type Recorder struct {
	Events []eventsv1.Event

	// eventf signature is intended to match the recorder for the events/v1 API.
	// - https://pkg.go.dev/k8s.io/client-go@v0.24.1/tools/events#EventRecorder
	eventf func(regarding, related runtime.Object, eventtype, reason, action, note string, args ...interface{})
}

// NewRecorder returns an EventRecorder for the deprecated v1.Event API.
func NewRecorder(t testing.TB, scheme *runtime.Scheme) *Recorder {
	t.Helper()

	var recorder Recorder

	// Construct an events/v1.Event and store it. This is a copy of the upstream
	// implementation except that t.Error is called rather than klog.
	// - https://releases.k8s.io/v1.24.1/staging/src/k8s.io/client-go/tools/events/event_recorder.go#L43-L92
	recorder.eventf = func(regarding, related runtime.Object, eventtype, reason, action, note string, args ...interface{}) {
		t.Helper()

		timestamp := metav1.MicroTime{Time: time.Now()}
		message := fmt.Sprintf(note, args...)

		refRegarding, err := reference.GetReference(scheme, regarding)
		assert.Check(t, err, "Could not construct reference to: '%#v'", regarding)

		var refRelated *corev1.ObjectReference
		if related != nil {
			refRelated, err = reference.GetReference(scheme, related)
			assert.Check(t, err, "Could not construct reference to: '%#v'", related)
		}

		assert.Check(t, util.ValidateEventType(eventtype), "Unsupported event type: '%v'", eventtype)

		namespace := refRegarding.Namespace
		if namespace == "" {
			namespace = metav1.NamespaceDefault
		}

		recorder.Events = append(recorder.Events, eventsv1.Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("%v.%x", refRegarding.Name, timestamp.UnixNano()),
				Namespace: namespace,
			},
			EventTime:           timestamp,
			Series:              nil,
			ReportingController: t.Name(),
			ReportingInstance:   t.Name() + "-{hostname}",
			Action:              action,
			Reason:              reason,
			Regarding:           *refRegarding,
			Related:             refRelated,
			Note:                message,
			Type:                eventtype,
		})
	}

	return &recorder
}

var _ record.EventRecorder = (*Recorder)(nil)

func (*Recorder) AnnotatedEventf(object runtime.Object, annotations map[string]string, eventtype, reason, messageFmt string, args ...interface{}) {
	panic("DEPRECATED: do not use AnnotatedEventf")
}
func (r *Recorder) Event(object runtime.Object, eventtype, reason, message string) {
	if r.eventf != nil {
		r.eventf(object, nil, eventtype, reason, "", message)
	}
}
func (r *Recorder) Eventf(object runtime.Object, eventtype, reason, messageFmt string, args ...interface{}) {
	if r.eventf != nil {
		r.eventf(object, nil, eventtype, reason, "", messageFmt, args...)
	}
}
