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

package pgadmin

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestWriteUsersInPGAdmin(t *testing.T) {
	ctx := context.Background()

	t.Run("Arguments", func(t *testing.T) {
		expected := errors.New("pass-through")
		exec := func(
			_ context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
		) error {
			assert.Assert(t, stdin != nil, "should send stdin")
			assert.Assert(t, stdout != nil, "should capture stdout")
			assert.Assert(t, stderr != nil, "should capture stderr")

			assert.Check(t, !strings.ContainsRune(strings.Join(command, ""), '\t'),
				"Python should not be indented with tabs")

			assert.DeepEqual(t, command, []string{"python", "-c", `
import importlib.util
import os
import sys

spec = importlib.util.find_spec('.pgadmin', (
    importlib.util.find_spec('pgadmin4') or
    importlib.util.find_spec('pgadmin4-web')
).name)
root = os.path.dirname(spec.submodule_search_locations[0])
if sys.path[0] != root:
    sys.path.insert(0, root)

import json
import sys
from pgadmin import create_app
from pgadmin.model import db, Role, User

with create_app().app_context():
    admin = db.session.query(User).filter_by(id=1).first()
    admin.active = False
    admin.email = ''
    admin.password = ''

    db.session.add(admin)
    db.session.commit()

    for line in sys.stdin:
        if not line.strip():
            continue

        data = json.loads(line)
        user = (
            db.session.query(User).filter_by(email=data['username']).first() or
            User(email=data['username'])
        )
        user.password = data['password']
        user.active = bool(user.password)
        user.roles = db.session.query(Role).filter_by(name='User').all()

        if user.password:
            user.verify_and_update_password(user.password)

        db.session.add(user)
        db.session.commit()
`})
			return expected
		}

		assert.Equal(t, expected, WriteUsersInPGAdmin(ctx, exec, nil, nil))
	})

	t.Run("Flake8", func(t *testing.T) {
		flake8, err := exec.LookPath("flake8")
		if err != nil {
			t.Skip(`requires "flake8" executable`)
		} else {
			output, err := exec.Command(flake8, "--version").CombinedOutput()
			assert.NilError(t, err)
			t.Logf("using %q:\n%s", flake8, output)
		}

		called := false
		exec := func(
			_ context.Context, _ io.Reader, _, _ io.Writer, command ...string,
		) error {
			called = true

			// Expect a python command with an inline script.
			assert.DeepEqual(t, command[:2], []string{"python", "-c"})
			assert.Assert(t, len(command) > 2)
			script := command[2]

			// Write out that inline script.
			dir := t.TempDir()
			file := filepath.Join(dir, "script.py")
			assert.NilError(t, os.WriteFile(file, []byte(script), 0o600))

			// Expect flake8 to be happy. Ignore "E402 module level import not
			// at top of file" in addition to the defaults.
			cmd := exec.Command(flake8, "--extend-ignore=E402", file)
			output, err := cmd.CombinedOutput()
			assert.NilError(t, err, "%q\n%s", cmd.Args, output)

			return nil
		}

		_ = WriteUsersInPGAdmin(ctx, exec, nil, nil)
		assert.Assert(t, called)
	})

	t.Run("Empty", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, _ ...string,
		) error {
			calls++

			b, err := ioutil.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Assert(t, len(b) == 0, "expected no stdin, got %q", string(b))
			return nil
		}

		assert.NilError(t, WriteUsersInPGAdmin(ctx, exec, nil, nil))
		assert.Equal(t, calls, 1)

		assert.NilError(t, WriteUsersInPGAdmin(ctx, exec, []v1beta1.PostgresUserSpec{}, nil))
		assert.Equal(t, calls, 2)

		assert.NilError(t, WriteUsersInPGAdmin(ctx, exec, nil, map[string]string{}))
		assert.Equal(t, calls, 3)
	})

	t.Run("Passwords", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, _ ...string,
		) error {
			calls++

			b, err := ioutil.ReadAll(stdin)
			assert.NilError(t, err)
			assert.DeepEqual(t, string(b), strings.TrimLeft(`
{"password":"","username":"user-no-options"}
{"password":"","username":"user-no-databases"}
{"password":"some$pass!word","username":"user-with-password"}
`, "\n"))
			return nil
		}

		assert.NilError(t, WriteUsersInPGAdmin(ctx, exec,
			[]v1beta1.PostgresUserSpec{
				{
					Name:      "user-no-options",
					Databases: []v1beta1.PostgresIdentifier{"db1"},
				},
				{
					Name:    "user-no-databases",
					Options: "some options here",
				},
				{
					Name: "user-with-password",
				},
			},
			map[string]string{
				"no-user":            "ignored",
				"user-with-password": "some$pass!word",
			},
		))
		assert.Equal(t, calls, 1)
	})
}
