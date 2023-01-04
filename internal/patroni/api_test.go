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

package patroni

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
)

// This example demonstrates how Executor can work with exec.Cmd.
func ExampleExecutor_execCmd() {
	_ = Executor(func(
		ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		// #nosec G204 Nothing calls the function defined in this example.
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
		return cmd.Run()
	})
}

func TestExecutorChangePrimaryAndWait(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		called := false
		exec := func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			called = true
			assert.DeepEqual(t, command, strings.Fields(
				`patronictl switchover --scheduled=now --force --master=old --candidate=new`,
			))
			assert.Assert(t, stdin == nil, "expected no stdin, got %T", stdin)
			assert.Assert(t, stderr != nil, "should capture stderr")
			assert.Assert(t, stdout != nil, "should capture stdout")
			return nil
		}

		_, _ = Executor(exec).ChangePrimaryAndWait(context.Background(), "old", "new")
		assert.Assert(t, called)
	})

	t.Run("Error", func(t *testing.T) {
		expected := errors.New("bang")
		_, actual := Executor(func(
			context.Context, io.Reader, io.Writer, io.Writer, ...string,
		) error {
			return expected
		}).ChangePrimaryAndWait(context.Background(), "any", "thing")

		assert.Equal(t, expected, actual)
	})

	t.Run("Result", func(t *testing.T) {
		success, _ := Executor(func(
			_ context.Context, _ io.Reader, stdout, _ io.Writer, _ ...string,
		) error {
			_, _ = stdout.Write([]byte(`no luck`))
			return nil
		}).ChangePrimaryAndWait(context.Background(), "any", "thing")

		assert.Assert(t, !success, "expected failure message to become false")

		success, _ = Executor(func(
			_ context.Context, _ io.Reader, stdout, _ io.Writer, _ ...string,
		) error {
			_, _ = stdout.Write([]byte(`Successfully switched over to something`))
			return nil
		}).ChangePrimaryAndWait(context.Background(), "any", "thing")

		assert.Assert(t, success, "expected success message to become true")
	})
}

func TestExecutorSwitchoverAndWait(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		called := false
		exec := func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			called = true
			assert.DeepEqual(t, command, strings.Fields(
				`patronictl switchover --scheduled=now --force --candidate=new`,
			))
			assert.Assert(t, stdin == nil, "expected no stdin, got %T", stdin)
			assert.Assert(t, stderr != nil, "should capture stderr")
			assert.Assert(t, stdout != nil, "should capture stdout")
			return nil
		}

		_, _ = Executor(exec).SwitchoverAndWait(context.Background(), "new")
		assert.Assert(t, called)
	})

	t.Run("Error", func(t *testing.T) {
		expected := errors.New("bang")
		_, actual := Executor(func(
			context.Context, io.Reader, io.Writer, io.Writer, ...string,
		) error {
			return expected
		}).SwitchoverAndWait(context.Background(), "next")

		assert.Equal(t, expected, actual)
	})

	t.Run("Result", func(t *testing.T) {
		success, _ := Executor(func(
			_ context.Context, _ io.Reader, stdout, _ io.Writer, _ ...string,
		) error {
			_, _ = stdout.Write([]byte(`no luck`))
			return nil
		}).SwitchoverAndWait(context.Background(), "next")

		assert.Assert(t, !success, "expected failure message to become false")

		success, _ = Executor(func(
			_ context.Context, _ io.Reader, stdout, _ io.Writer, _ ...string,
		) error {
			_, _ = stdout.Write([]byte(`Successfully switched over to something`))
			return nil
		}).SwitchoverAndWait(context.Background(), "next")

		assert.Assert(t, success, "expected success message to become true")
	})
}

func TestExecutorFailoverAndWait(t *testing.T) {
	t.Run("Arguments", func(t *testing.T) {
		called := false
		exec := func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			called = true
			assert.DeepEqual(t, command, strings.Fields(
				`patronictl failover --force --candidate=new`,
			))
			assert.Assert(t, stdin == nil, "expected no stdin, got %T", stdin)
			assert.Assert(t, stderr != nil, "should capture stderr")
			assert.Assert(t, stdout != nil, "should capture stdout")
			return nil
		}

		_, _ = Executor(exec).FailoverAndWait(context.Background(), "new")
		assert.Assert(t, called)
	})

	t.Run("Error", func(t *testing.T) {
		expected := errors.New("bang")
		_, actual := Executor(func(
			context.Context, io.Reader, io.Writer, io.Writer, ...string,
		) error {
			return expected
		}).FailoverAndWait(context.Background(), "next")

		assert.Equal(t, expected, actual)
	})

	t.Run("Result", func(t *testing.T) {
		success, _ := Executor(func(
			_ context.Context, _ io.Reader, stdout, _ io.Writer, _ ...string,
		) error {
			_, _ = stdout.Write([]byte(`no luck`))
			return nil
		}).FailoverAndWait(context.Background(), "next")

		assert.Assert(t, !success, "expected failure message to become false")

		success, _ = Executor(func(
			_ context.Context, _ io.Reader, stdout, _ io.Writer, _ ...string,
		) error {
			_, _ = stdout.Write([]byte(`Successfully failed over to something`))
			return nil
		}).FailoverAndWait(context.Background(), "next")

		assert.Assert(t, success, "expected success message to become true")
	})
}

func TestExecutorReplaceConfiguration(t *testing.T) {
	expected := errors.New("bang")
	exec := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		assert.DeepEqual(t, command, strings.Fields(
			`patronictl edit-config --replace=- --force`,
		))
		str, ok := stdin.(fmt.Stringer)
		assert.Assert(t, ok, "bug in test: wanted to call String()")
		assert.Equal(t, str.String(), `{"some":"values"}`+"\n", "should send JSON on stdin")
		assert.Assert(t, stderr != nil, "should capture stderr")
		assert.Assert(t, stdout != nil, "should capture stdout")
		return expected
	}

	actual := Executor(exec).ReplaceConfiguration(
		context.Background(), map[string]interface{}{"some": "values"})

	assert.Equal(t, expected, actual, "should call exec")
}

func TestExecutorRestartPendingMembers(t *testing.T) {
	expected := errors.New("oop")
	exec := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		assert.DeepEqual(t, command, strings.Fields(
			`patronictl restart --pending --force --role=sock-role shoe-scope`,
		))
		assert.Assert(t, stdin == nil, "expected no stdin, got %T", stdin)
		assert.Assert(t, stderr != nil, "should capture stderr")
		assert.Assert(t, stdout != nil, "should capture stdout")
		return expected
	}

	actual := Executor(exec).RestartPendingMembers(
		context.Background(), "sock-role", "shoe-scope")

	assert.Equal(t, expected, actual, "should call exec")
}

func TestExecutorGetTimeline(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		expected := errors.New("bang")
		tl, actual := Executor(func(
			context.Context, io.Reader, io.Writer, io.Writer, ...string,
		) error {
			return expected
		}).GetTimeline(context.Background())

		assert.Equal(t, expected, actual)
		assert.Equal(t, tl, int64(0))
	})

	t.Run("Stderr", func(t *testing.T) {
		tl, actual := Executor(func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			stderr.Write([]byte(`no luck`))
			return nil
		}).GetTimeline(context.Background())

		assert.Error(t, actual, "no luck")
		assert.Equal(t, tl, int64(0))
	})

	t.Run("BadJSON", func(t *testing.T) {
		tl, actual := Executor(func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			stdout.Write([]byte(`no luck`))
			return nil
		}).GetTimeline(context.Background())

		assert.Error(t, actual, "invalid character 'o' in literal null (expecting 'u')")
		assert.Equal(t, tl, int64(0))
	})

	t.Run("NoLeader", func(t *testing.T) {
		tl, actual := Executor(func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			stdout.Write([]byte(`[{"Cluster": "hippo-ha", "Member": "hippo-instance1-ltcf-0", "Host": "hippo-instance1-ltcf-0.hippo-pods", "Role": "Replica", "State": "running", "TL": 4, "Lag in MB": 0}]`))
			return nil
		}).GetTimeline(context.Background())

		assert.NilError(t, actual)
		assert.Equal(t, tl, int64(0))
	})

	t.Run("Success", func(t *testing.T) {
		tl, actual := Executor(func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			stdout.Write([]byte(`[{"Cluster": "hippo-ha", "Member": "hippo-instance1-67mc-0", "Host": "hippo-instance1-67mc-0.hippo-pods", "Role": "Leader", "State": "running", "TL": 4}, {"Cluster": "hippo-ha", "Member": "hippo-instance1-ltcf-0", "Host": "hippo-instance1-ltcf-0.hippo-pods", "Role": "Replica", "State": "running", "TL": 4, "Lag in MB": 0}]`))
			return nil
		}).GetTimeline(context.Background())

		assert.NilError(t, actual)
		assert.Equal(t, tl, int64(4))
	})
}
