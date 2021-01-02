package pgo_cli_test

/*
 Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

type pgoCmd struct {
	cmd     *exec.Cmd
	timeout <-chan time.Time
}

// pgo returns a builder of an invocation of the PostgreSQL Operator CLI. It
// inherits a copy of any variables set in TestContext.DefaultEnvironment and
// the calling environment.
func pgo(args ...string) *pgoCmd {
	c := new(pgoCmd)
	c.cmd = exec.Command("pgo", args...)
	c.cmd.Env = append(c.cmd.Env, TestContext.DefaultEnvironment...)
	c.cmd.Env = append(c.cmd.Env, os.Environ()...)
	return c
}

// WithEnvironment overrides a single environment variable of this invocation.
func (c *pgoCmd) WithEnvironment(key, value string) *pgoCmd {
	c.cmd.Env = append(c.cmd.Env, key+"="+value)
	return c
}

// Exec invokes the PostgreSQL Operator CLI, returning its standard output and
// any error encountered.
func (c *pgoCmd) Exec(t testing.TB) (string, error) {
	var stdout, stderr bytes.Buffer

	cmd := c.cmd
	cmd.Stdout, cmd.Stderr = &stdout, &stderr

	//t.Logf("Running `%s %s`", cmd.Path, strings.Join(cmd.Args[1:], " "))
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf(
			"error starting %q: %v\nstdout:\n%v\nstderr:\n%v",
			cmd.Path, err, stdout.String(), stderr.String())
	}

	chError := make(chan error, 1)
	chTimeout := c.timeout

	if chTimeout == nil {
		chTimeout = time.After(time.Minute)
	}

	go func() { chError <- cmd.Wait() }()
	select {
	case err := <-chError:
		if err != nil {
			//if ee, ok := err.(*exec.ExitError); ok {
			//	t.Logf("rc: %v", ee.ProcessState.ExitCode())
			//}
			return stdout.String(), fmt.Errorf(
				"error running %q: %v\nstdout:\n%v\nstderr:\n%v",
				cmd.Path, err, stdout.String(), stderr.String())
		}
	case <-chTimeout:
		cmd.Process.Kill()
		return stdout.String(), fmt.Errorf(
			"timed out waiting for %q:\nstdout:\n%v\nstderr:\n%v",
			cmd.Path, stdout.String(), stderr.String())
	}
	return stdout.String(), nil
}
