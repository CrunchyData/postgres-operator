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

package logging

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	logrtest "github.com/go-logr/logr/testing"
	"github.com/wojas/genericr"
	"gotest.tools/v3/assert"
)

func TestFromContext(t *testing.T) {
	global = logrtest.NullLogger{}
	assert.Assert(t, global == logrtest.NullLogger{}, "expected type to be comparable")

	// Defaults to global.
	log := FromContext(context.Background())
	assert.Equal(t, log, global)

	// Retrieves from NewContext.
	double := struct{ logr.Logger }{logrtest.NullLogger{}}
	log = FromContext(NewContext(context.Background(), double))
	assert.Equal(t, log, double)
}

func TestSetLogFunc(t *testing.T) {
	var calls []string

	SetLogFunc(0, func(input genericr.Entry) {
		calls = append(calls, input.Message)
	})

	global.Info("called")
	assert.DeepEqual(t, calls, []string{"called"})
}
