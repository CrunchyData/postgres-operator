// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgadmin

import (
	"context"
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"gotest.tools/v3/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crunchydata/postgres-operator/internal/initialize"
	"github.com/crunchydata/postgres-operator/internal/testing/require"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

func TestWriteUsersInPGAdmin(t *testing.T) {
	ctx := context.Background()
	cluster := &v1beta1.PostgresCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "testcluster",
			Namespace: "testnamespace",
		},
		Spec: v1beta1.PostgresClusterSpec{
			Port: initialize.Int32(5432),
		},
	}

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
import sys
import types

cluster = types.SimpleNamespace()
(cluster.name, cluster.hostname, cluster.port) = sys.argv[1:]

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

import copy
import json
import sys

from pgadmin import create_app
from pgadmin.model import db, Role, User, Server, ServerGroup
from pgadmin.utils.constants import INTERNAL
from pgadmin.utils.crypto import encrypt

with create_app().app_context():

    admin = db.session.query(User).filter_by(id=1).first()
    admin.active = False
    admin.email = ''
    admin.password = ''
    admin.username = ''

    db.session.add(admin)
    db.session.commit()

    for line in sys.stdin:
        if not line.strip():
            continue

        data = json.loads(line)
        address = data['username'] + '@pgo'
        user = (
            db.session.query(User).filter_by(username=address).first() or
            User()
        )
        user.auth_source = INTERNAL
        user.email = user.username = address
        user.password = data['password']
        user.active = bool(user.password)
        user.roles = db.session.query(Role).filter_by(name='User').all()

        if user.password:
            user.masterpass_check = 'any'
            user.verify_and_update_password(user.password)

        db.session.add(user)
        db.session.commit()

        group = (
            db.session.query(ServerGroup).filter_by(
                user_id=user.id,
            ).order_by("id").first() or
            ServerGroup()
        )
        group.name = "Crunchy PostgreSQL Operator"
        group.user_id = user.id
        db.session.add(group)
        db.session.commit()

        server = (
            db.session.query(Server).filter_by(
                servergroup_id=group.id,
                user_id=user.id,
                name=cluster.name,
            ).first() or
            Server()
        )

        server.name = cluster.name
        server.host = cluster.hostname
        server.port = cluster.port
        server.servergroup_id = group.id
        server.user_id = user.id
        server.maintenance_db = "postgres"
        server.ssl_mode = "prefer"

        server.username = data['username']
        server.password = encrypt(data['password'], data['password'])
        server.save_password = int(bool(data['password']))

        if server.id and db.session.is_modified(server):
            old = copy.deepcopy(server)
            db.make_transient(server)
            server.id = None
            db.session.delete(old)

        db.session.add(server)
        db.session.commit()
`,
				"testcluster",
				"testcluster-primary.testnamespace.svc",
				"5432",
			})
			return expected
		}

		assert.Equal(t, expected, WriteUsersInPGAdmin(ctx, cluster, exec, nil, nil))
	})

	t.Run("Flake8", func(t *testing.T) {
		flake8 := require.Flake8(t)

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

		_ = WriteUsersInPGAdmin(ctx, cluster, exec, nil, nil)
		assert.Assert(t, called)
	})

	t.Run("Empty", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, _ ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.Assert(t, len(b) == 0, "expected no stdin, got %q", string(b))
			return nil
		}

		assert.NilError(t, WriteUsersInPGAdmin(ctx, cluster, exec, nil, nil))
		assert.Equal(t, calls, 1)

		assert.NilError(t, WriteUsersInPGAdmin(ctx, cluster, exec, []v1beta1.PostgresUserSpec{}, nil))
		assert.Equal(t, calls, 2)

		assert.NilError(t, WriteUsersInPGAdmin(ctx, cluster, exec, nil, map[string]string{}))
		assert.Equal(t, calls, 3)
	})

	t.Run("Passwords", func(t *testing.T) {
		calls := 0
		exec := func(
			_ context.Context, stdin io.Reader, _, _ io.Writer, _ ...string,
		) error {
			calls++

			b, err := io.ReadAll(stdin)
			assert.NilError(t, err)
			assert.DeepEqual(t, string(b), strings.TrimLeft(`
{"password":"","username":"user-no-options"}
{"password":"","username":"user-no-databases"}
{"password":"some$pass!word","username":"user-with-password"}
`, "\n"))
			return nil
		}

		assert.NilError(t, WriteUsersInPGAdmin(ctx, cluster, exec,
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
