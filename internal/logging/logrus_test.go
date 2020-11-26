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

package logging_test

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/wojas/genericr"

	"github.com/crunchydata/postgres-operator/internal/logging"
)

func assertLogrusContains(t testing.TB, actual, expected string) {
	t.Helper()

	if !strings.Contains(actual, expected) {
		t.Fatalf("missing from logrus:\n%s", cmp.Diff(expected, strings.Fields(actual)))
	}
}

func TestLogrus(t *testing.T) {
	t.Parallel()

	out := new(bytes.Buffer)
	logrus := logging.Logrus(out, "v1", 1)

	// Default level is INFO.
	// Version field is always present.
	out.Reset()
	logrus(genericr.Entry{})
	assertLogrusContains(t, out.String(), `level=info version=v1`)

	// Configured level or higher is DEBUG.
	out.Reset()
	logrus(genericr.Entry{Level: 1})
	assertLogrusContains(t, out.String(), `level=debug`)
	out.Reset()
	logrus(genericr.Entry{Level: 2})
	assertLogrusContains(t, out.String(), `level=debug`)

	// Any error becomes ERROR level.
	out.Reset()
	logrus(genericr.Entry{Error: errors.New("dang")})
	assertLogrusContains(t, out.String(), `level=error error=dang`)

	out.Reset()
	logrus(genericr.Entry{Fields: []interface{}{"k1", "str", "k2", 13, "k3", false}})
	assertLogrusContains(t, out.String(), `k1=str k2=13 k3=false`)

	out.Reset()
	logrus(genericr.Entry{Message: "banana"})
	assertLogrusContains(t, out.String(), `msg=banana`)

	// Fields don't overwrite builtins.
	out.Reset()
	logrus(genericr.Entry{
		Message: "banana",
		Error:   errors.New("dang"),
		Fields: []interface{}{
			"error", "not-err",
			"level", "not-lvl",
			"msg", "not-msg",
		},
	})
	assertLogrusContains(t, out.String(), `level=error msg=banana error=dang`)
	assertLogrusContains(t, out.String(), `fields.error=not-err fields.level=not-lvl fields.msg=not-msg`)
}

func TestLogrusCaller(t *testing.T) {
	t.Parallel()

	out := new(bytes.Buffer)
	log := genericr.New(logging.Logrus(out, "v2", 2)).WithCaller(true)

	// Details come from the line of the logr.Logger call.
	_, _, baseline, _ := runtime.Caller(0)
	log.Info("")
	assertLogrusContains(t, out.String(), fmt.Sprintf(`file="internal/logging/logrus_test.go:%d"`, baseline+1))
	assertLogrusContains(t, out.String(), `func=logging_test.TestLogrusCaller`)

	// Fields don't overwrite builtins.
	out.Reset()
	_, _, baseline, _ = runtime.Caller(0)
	log.Info("", "file", "not-file", "func", "not-func")
	assertLogrusContains(t, out.String(), fmt.Sprintf(`file="internal/logging/logrus_test.go:%d"`, baseline+1))
	assertLogrusContains(t, out.String(), `func=logging_test.TestLogrusCaller`)
	assertLogrusContains(t, out.String(), `fields.file=not-file fields.func=not-func`)
}
