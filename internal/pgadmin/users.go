// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgadmin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/crunchydata/postgres-operator/internal/logging"
	"github.com/crunchydata/postgres-operator/internal/naming"
	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

type Executor func(
	ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// WriteUsersInPGAdmin uses exec and "python" to create users in pgAdmin and
// update their passwords when they already exist. A blank password for a user
// blocks that user from logging in to pgAdmin. The pgAdmin configuration
// database must exist before calling this.
func WriteUsersInPGAdmin(
	ctx context.Context, cluster *v1beta1.PostgresCluster, exec Executor,
	users []v1beta1.PostgresUserSpec, passwords map[string]string,
) error {
	primary := naming.ClusterPrimaryService(cluster)

	args := []string{
		cluster.Name,
		primary.Name + "." + primary.Namespace + ".svc",
		fmt.Sprint(*cluster.Spec.Port),
	}
	script := strings.Join([]string{
		// Unpack arguments into an object.
		// - https://docs.python.org/3/library/types.html#types.SimpleNamespace
		`
import sys
import types

cluster = types.SimpleNamespace()
(cluster.name, cluster.hostname, cluster.port) = sys.argv[1:]`,

		// The location of pgAdmin files can vary by container image. Look for
		// typical names in the module search path: the PyPI package is named
		// "pgadmin4" while custom builds might use "pgadmin4-web". The pgAdmin
		// packages expect to find themselves on the search path, so prepend
		// that directory there (like pgAdmin does in its WSGI entrypoint).
		// - https://pypi.org/project/pgadmin4/
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgAdmin4.wsgi#L18
		`
import importlib.util
import os
import sys

spec = importlib.util.find_spec('.pgadmin', (
    importlib.util.find_spec('pgadmin4') or
    importlib.util.find_spec('pgadmin4-web')
).name)
root = os.path.dirname(spec.submodule_search_locations[0])
if sys.path[0] != root:
    sys.path.insert(0, root)`,

		// Import pgAdmin modules now that they are on the search path.
		// NOTE: When testing with the REPL, use the `__enter__` method to
		// avoid one level of indentation.
		//
		//     create_app().app_context().__enter__()
		//
		`
import copy
import json
import sys

from pgadmin import create_app
from pgadmin.model import db, Role, User, Server, ServerGroup
from pgadmin.utils.constants import INTERNAL
from pgadmin.utils.crypto import encrypt

with create_app().app_context():`,

		// The user with id=1 is automatically created by pgAdmin when it
		// creates its configuration database. Clear that email and username
		// so they cannot conflict with users we create, and deactivate the user
		// so it cannot log in.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/migrations/versions/fdc58d9bd449_.py#L129
		`
    admin = db.session.query(User).filter_by(id=1).first()
    admin.active = False
    admin.email = ''
    admin.password = ''
    admin.username = ''

    db.session.add(admin)
    db.session.commit()`,

		// Process each line of input as a single user definition. Those with
		// a non-blank password are allowed to login.
		//
		// The "internal" authentication source requires that username and email
		// be the same and be an email address. Append "@pgo" to the username
		// to pass login validation.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/authenticate/internal.py#L88
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/utils/validation_utils.py#L13
		//
		// The "auth_source" and "username" attributes are part of the User
		// model since pgAdmin v4.21.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/model/__init__.py#L66
		`
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
        user.roles = db.session.query(Role).filter_by(name='User').all()`,

		// After a user logs in, pgAdmin checks that the "master password" is
		// set. It does not seem to use the value nor check that it is valid.
		// We set it to "any" to satisfy the check.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/browser/__init__.py#L963
		//
		// The "verify_and_update_password" method hashes the plaintext password
		// according to pgAdmin security settings. It is part of the User model
		// since pgAdmin v4.19 and Flask-Security-Too v3.20.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/requirements.txt#L40
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/model/__init__.py#L66
		// - https://flask-security-too.readthedocs.io/en/stable/api.html#flask_security.UserMixin.verify_and_update_password
		`
        if user.password:
            user.masterpass_check = 'any'
            user.verify_and_update_password(user.password)`,

		// Write the user to get its generated identity.
		`
        db.session.add(user)
        db.session.commit()`,

		// One server group and connection are configured for each user, similar
		// to the way they are made using their respective dialog windows.
		// - https://www.pgadmin.org/docs/pgadmin4/latest/server_group_dialog.html
		// - https://www.pgadmin.org/docs/pgadmin4/latest/server_dialog.html
		//
		// We use a similar method to the import method when creating server connections
		// - https://www.pgadmin.org/docs/pgadmin4/latest/import_export_servers.html
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/setup.py#L294
		`
        group = (
            db.session.query(ServerGroup).filter_by(
                user_id=user.id,
            ).order_by("id").first() or
            ServerGroup()
        )
        group.name = "Crunchy PostgreSQL Operator"
        group.user_id = user.id
        db.session.add(group)
        db.session.commit()`,

		// The name of the server connection is the same as the cluster name.
		// Note that the server connections are created when the users are created or
		// modified. Changes to a server connection will generally persist until a
		// change is made to the corresponding user. For custom server connections,
		// a new server should be created with a unique name.
		`
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
        server.ssl_mode = "prefer"`,

		// Encrypt the Server password with the User's plaintext password.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/__init__.py#L601
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/utils/master_password.py#L21
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/browser/server_groups/servers/__init__.py#L1091
		//
		// The "save_password" attribute is part of the Server model since
		// pgAdmin v4.21.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/model/__init__.py#L108
		`
        server.username = data['username']
        server.password = encrypt(data['password'], data['password'])
        server.save_password = int(bool(data['password']))`,

		// Due to limitations on the types of updates that can be made to active
		// server connections, when the current server connection is updated, we
		// need to delete it and add a new server connection in its place. This
		// will require a refresh if pgAdmin web GUI is being used when the
		// update takes place.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-4_30/web/pgadmin/browser/server_groups/servers/__init__.py#L772
		//
		// TODO(cbandy): We could possibly get the same effect by invalidating
		// the user's sessions in pgAdmin v5.4 with Flask-Security-Too v4.
		// - https://github.com/pgadmin-org/pgadmin4/blob/REL-5_4/web/pgadmin/model/__init__.py#L67
		// - https://flask-security-too.readthedocs.io/en/stable/api.html#flask_security.UserDatastore.set_uniquifier
		`
        if server.id and db.session.is_modified(server):
            old = copy.deepcopy(server)
            db.make_transient(server)
            server.id = None
            db.session.delete(old)

        db.session.add(server)
        db.session.commit()`,
	}, "\n") + "\n"

	var err error
	var stdin, stdout, stderr bytes.Buffer

	encoder := json.NewEncoder(&stdin)
	encoder.SetEscapeHTML(false)

	for i := range users {
		spec := users[i]

		if err == nil {
			err = encoder.Encode(map[string]interface{}{
				"username": spec.Name,
				"password": passwords[string(spec.Name)],
			})
		}
	}

	if err == nil {
		err = exec(ctx, &stdin, &stdout, &stderr,
			append([]string{"python", "-c", script}, args...)...)

		log := logging.FromContext(ctx)
		log.V(1).Info("wrote pgAdmin users",
			"stdout", stdout.String(),
			"stderr", stderr.String())
	}

	return err
}
