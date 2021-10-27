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
	"bytes"
	"context"
	"encoding/json"
	"io"

	"github.com/crunchydata/postgres-operator/internal/logging"
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
	ctx context.Context, exec Executor,
	users []v1beta1.PostgresUserSpec, passwords map[string]string,
) error {
	// The location of pgAdmin files can vary by container image. Look for
	// typical names in the module search path: the PyPI package is named
	// "pgadmin4" while custom builds might use "pgadmin4-web". The pgAdmin
	// packages expect to find themselves on the search path, so prepend that
	// directory there (like pgAdmin does in its WSGI entrypoint).
	// - https://pypi.org/project/pgadmin4/
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/pgAdmin4.wsgi;hb=REL-4_20#l13
	const search = `
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
`

	// The user with id=1 is automatically created by pgAdmin when it creates
	// its configuration database. Clear that username so it cannot conflict
	// with the users we create, and deactivate the user so it cannot log in.
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/migrations/versions/fdc58d9bd449_.py;hb=REL-4_20#l129
	//
	// The "verify_and_update_password" method hashes the plaintext password
	// according to pgAdmin security settings. It is part of the User model
	// since pgAdmin v4.19 and Flask-Security-Too v3.20.
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=requirements.txt;hb=REL-4_20#l40
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/pgadmin/model/__init__.py;hb=REL-4_20#l65
	// - https://flask-security-too.readthedocs.io/en/stable/api.html#flask_security.UserMixin.verify_and_update_password
	//
	// TODO(cbandy): pgAdmin v4.21 adds "auth_source" and "username" as required attributes.
	// - https://git.postgresql.org/gitweb/?p=pgadmin4.git;f=web/pgadmin/model/__init__.py;hb=REL-4_21#l65
	const script = `
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
`

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
		err = exec(ctx, &stdin, &stdout, &stderr, "python", "-c", search+script)

		log := logging.FromContext(ctx)
		log.V(1).Info("wrote pgAdmin users",
			"stdout", stdout.String(),
			"stderr", stderr.String())
	}

	return err
}
