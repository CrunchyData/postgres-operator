/*
 Copyright 2021 - 2023 Crunchy Data Solutions, Inc.
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

package pgmonitor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

func TestExecutorGetExporterSetupSQL(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		version := 12
		called := false
		exec := func(
			ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			called = true
			assert.DeepEqual(t, command, strings.Fields(
				fmt.Sprintf("cat /opt/cpm/conf/pg%d/setup.sql", version),
			))

			assert.Assert(t, stdin == nil, "expected no stdin, got %T", stdin)
			assert.Assert(t, stderr != nil, "should capture stderr")
			assert.Assert(t, stdout != nil, "should capture stdout")
			return nil
		}

		_, _, _ = Executor(exec).GetExporterSetupSQL(context.Background(), version)
		assert.Assert(t, called)
	})

	t.Run("Error", func(t *testing.T) {
		expected := errors.New("boom")
		_, _, actual := Executor(func(
			context.Context, io.Reader, io.Writer, io.Writer, ...string) error {
			return expected
		}).GetExporterSetupSQL(context.Background(), 0)

		assert.Equal(t, expected, actual)
	})

	t.Run("Result", func(t *testing.T) {
		stdout, _, _ := Executor(func(
			_ context.Context, _ io.Reader, stdout, stderr io.Writer, _ ...string) error {
			_, _ = stdout.Write([]byte(""))
			return nil
		}).GetExporterSetupSQL(context.Background(), 0)
		assert.Assert(t, stdout == "")

		stdout, _, _ = Executor(func(
			_ context.Context, _ io.Reader, stdout, stderr io.Writer, _ ...string) error {
			_, _ = stdout.Write([]byte("something"))
			return nil
		}).GetExporterSetupSQL(context.Background(), 0)
		assert.Assert(t, stdout != "")

		_, stderr, _ := Executor(func(
			_ context.Context, _ io.Reader, stdout, stderr io.Writer, _ ...string) error {
			_, _ = stderr.Write([]byte(""))
			return nil
		}).GetExporterSetupSQL(context.Background(), 0)

		assert.Assert(t, stderr == "")

		_, stderr, _ = Executor(func(
			_ context.Context, _ io.Reader, stdout, stderr io.Writer, _ ...string) error {
			_, _ = stderr.Write([]byte("something"))
			return nil
		}).GetExporterSetupSQL(context.Background(), 0)
		assert.Assert(t, stderr != "")

	})
}
