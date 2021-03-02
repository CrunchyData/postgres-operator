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
		// #nosec G204 Executor only calls `patronictl`.
		cmd := exec.CommandContext(ctx, command[0], command[1:]...)
		cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, stderr
		return cmd.Run()
	})
}

func TestExecutorChangePrimary(t *testing.T) {
	expected := errors.New("bang")
	exec := func(
		_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
	) error {
		assert.DeepEqual(t, command, strings.Fields(
			`patronictl switchover --scheduled=now --force --master=old --candidate=new`,
		))
		assert.Assert(t, stdin == nil, "expected no stderr, got %T", stdin)
		assert.Assert(t, stderr != nil, "should capture stderr")
		assert.Assert(t, stdout != nil, "should capture stdout")
		return expected
	}

	actual := Executor(exec).ChangePrimary(context.Background(), "old", "new")

	assert.Equal(t, expected, actual, "should call exec")
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
