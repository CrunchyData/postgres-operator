/*
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

package pgbackrest

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
)

const (
	// errMsgConfigHashMismatch is the error message displayed when a configuration hash mismatch
	// is detected while attempting stanza creation
	errMsgConfigHashMismatch = "postgres operator error: pgBackRest config hash mismatch"

	// errMsgBackupDbMismatch is the error message returned from pgBackRest when PG versions
	// or PG system identifiers do not match between the PG instance and the existing stanza
	errMsgBackupDbMismatch = "backup and archive info files exist but do not match the database"
)

// Executor calls "pgbackrest" commands
type Executor func(
	ctx context.Context, stdin io.Reader, stdout, stderr io.Writer, command ...string,
) error

// StanzaCreateOrUpgrade runs either the pgBackRest "stanza-create" or "stanza-upgrade" command
// depending on the boolean value of the "upgrade" function parameter. This function is invoked
// by the "reconcileStanzaCreate" function with "upgrade" set to false; if the stanza already
// exists but the PG version has changed, pgBackRest will error with the "errMsgBackupDbMismatch"
// error. If that occurs, we then rerun the command with "upgrade" set to true.
// - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/command/check/common.c#L154-L156
// If the bool returned from this function is true, this indicates that a pgBackRest config hash
// mismatch was identified that prevented the pgBackRest stanza-create or stanza-upgrade command
// from running (with a config mismatch indicating that the pgBackRest configuration as stored in
// the cluster's pgBackRest ConfigMap has not yet propagated to the Pod).
func (exec Executor) StanzaCreateOrUpgrade(ctx context.Context, configHash string,
	upgrade bool) (bool, error) {

	var stdout, stderr bytes.Buffer

	stanzaCmd := "create"
	if upgrade {
		stanzaCmd = "upgrade"
	}

	// this is the script that is run to create a stanza.  First it checks the
	// "config-hash" file to ensure all configuration changes (e.g. from ConfigMaps) have
	// propagated to the container, and if not, it prints an error and returns with exit code 1).
	// Otherwise, it runs the pgbackrest command, which will either be "stanza-create" or
	// "stanza-upgrade", depending on the value of the boolean "upgrade" parameter.
	const script = `
declare -r hash="$1" stanza="$2" message="$3" cmd="$4"
if [[ "$(< /etc/pgbackrest/conf.d/config-hash)" != "${hash}" ]]; then
    printf >&2 "%s" "${message}"; exit 1;
else
    pgbackrest "${cmd}" --stanza="${stanza}"
fi
`
	if err := exec(ctx, nil, &stdout, &stderr, "bash", "-ceu", "--",
		script, "-", configHash, DefaultStanzaName, errMsgConfigHashMismatch,
		fmt.Sprintf("stanza-%s", stanzaCmd)); err != nil {

		errReturn := stderr.String()

		// if the config hashes didn't match, return true and don't return an error since this is
		// expected while waiting for config changes in ConfigMaps and Secrets to make it to the
		// container
		if errReturn == errMsgConfigHashMismatch {
			return true, nil
		}

		// if the err returned from pgbackrest command is about a version mismatch
		// then we should run upgrade rather than create
		if strings.Contains(errReturn, errMsgBackupDbMismatch) {
			return Executor(exec).StanzaCreateOrUpgrade(ctx, configHash, true)
		}

		// if none of the above errors, return the err
		return false, errors.WithStack(fmt.Errorf("%w: %v", err, errReturn))
	}

	return false, nil
}
