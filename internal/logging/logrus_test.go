// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package logging

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/pkg/errors"
	"gotest.tools/v3/assert"
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
	logrus := Logrus(out, "v1", 1, 2)

	// Configured verbosity discards.
	assert.Assert(t, logrus.Enabled(1))
	assert.Assert(t, logrus.Enabled(2))
	assert.Assert(t, !logrus.Enabled(3))

	// Default level is INFO.
	// Version field is always present.
	out.Reset()
	logrus.Info(0, "")
	assertLogrusContains(t, out.String(), `level=info version=v1`)

	// Configured level or higher is DEBUG.
	out.Reset()
	logrus.Info(1, "")
	assertLogrusContains(t, out.String(), `level=debug`)
	out.Reset()
	logrus.Info(2, "")
	assertLogrusContains(t, out.String(), `level=debug`)

	// Any error is ERROR level.
	out.Reset()
	logrus.Error(fmt.Errorf("%s", "dang"), "")
	assertLogrusContains(t, out.String(), `level=error error=dang`)

	// A wrapped error includes one frame of its stack.
	out.Reset()
	_, _, baseline, _ := runtime.Caller(0)
	logrus.Error(errors.New("dang"), "")
	assertLogrusContains(t, out.String(), fmt.Sprintf(`file="internal/logging/logrus_test.go:%d"`, baseline+1))
	assertLogrusContains(t, out.String(), `func=logging.TestLogrus`)

	out.Reset()
	logrus.Info(0, "", "k1", "str", "k2", 13, "k3", false)
	assertLogrusContains(t, out.String(), `k1=str k2=13 k3=false`)

	out.Reset()
	logrus.Info(0, "banana")
	assertLogrusContains(t, out.String(), `msg=banana`)

	// Fields don't overwrite builtins.
	out.Reset()
	logrus.Error(errors.New("dang"), "banana",
		"error", "not-err",
		"file", "not-file",
		"func", "not-func",
		"level", "not-lvl",
		"msg", "not-msg",
	)
	assertLogrusContains(t, out.String(), `level=error msg=banana error=dang`)
	assertLogrusContains(t, out.String(), `fields.error=not-err fields.file=not-file fields.func=not-func`)
	assertLogrusContains(t, out.String(), `fields.level=not-lvl fields.msg=not-msg`)
}
