// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

package pgbackrest

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/crunchydata/postgres-operator/pkg/apis/postgres-operator.crunchydata.com/v1beta1"
)

const (
	// errMsgConfigHashMismatch is the error message displayed when a configuration hash mismatch
	// is detected while attempting stanza creation
	errMsgConfigHashMismatch = "postgres operator error: pgBackRest config hash mismatch"

	// errMsgStaleReposWithVolumesConfig is the error message displayed when a volume-backed repo has been
	// configured, but the configuration has not yet propagated into the container.
	errMsgStaleReposWithVolumesConfig = "postgres operator error: pgBackRest stale volume-backed repo configuration"
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
	postgresCluster *v1beta1.PostgresCluster) (bool, error) {

	var stdout, stderr bytes.Buffer

	var reposWithVolumes []v1beta1.PGBackRestRepo
	for _, repo := range postgresCluster.Spec.Backups.PGBackRest.Repos {
		if repo.Volume != nil {
			reposWithVolumes = append(reposWithVolumes, repo)
		}
	}

	grep := "grep %s-path /etc/pgbackrest/conf.d/pgbackrest_instance.conf"

	var checkRepoCmd string
	if len(reposWithVolumes) > 0 {
		repo := reposWithVolumes[0]
		checkRepoCmd = checkRepoCmd + fmt.Sprintf(grep, repo.Name)

		reposWithVolumes = reposWithVolumes[1:]
		for _, repo := range reposWithVolumes {
			checkRepoCmd = checkRepoCmd + fmt.Sprintf(" && "+grep, repo.Name)
		}
	}

	// this is the script that is run to create a stanza.  First it checks the
	// "config-hash" file to ensure all configuration changes (e.g. from ConfigMaps) have
	// propagated to the container, and if not, it prints an error and returns with exit code 1).
	// Next, it checks that any volume-backed repo added to the config has propagated into
	// the container, and if not, prints an error and exits with code 1.
	// Otherwise, it runs the pgbackrest command, which will either be "stanza-create" or
	// "stanza-upgrade", depending on the value of the boolean "upgrade" parameter.
	const script = `
declare -r hash="$1" stanza="$2" hash_msg="$3" vol_msg="$4" check_repo_cmd="$5"
if [[ "$(< /etc/pgbackrest/conf.d/config-hash)" != "${hash}" ]]; then
    printf >&2 "%s" "${hash_msg}"; exit 1;
elif ! bash -c "${check_repo_cmd}"; then
 	 printf >&2 "%s" "${vol_msg}"; exit 1;
else
    pgbackrest stanza-create --stanza="${stanza}" || pgbackrest stanza-upgrade --stanza="${stanza}"
fi
`
	if err := exec(ctx, nil, &stdout, &stderr, "bash", "-ceu", "--",
		script, "-", configHash, DefaultStanzaName, errMsgConfigHashMismatch, errMsgStaleReposWithVolumesConfig,
		checkRepoCmd); err != nil {

		errReturn := stderr.String()

		// if the config hashes didn't match, return true and don't return an error since this is
		// expected while waiting for config changes in ConfigMaps and Secrets to make it to the
		// container
		if errReturn == errMsgConfigHashMismatch {
			return true, nil
		}

		// if the configuration for volume-backed repositories is stale, return true and don't return an error since this
		// is expected while waiting for config changes in ConfigMaps to make it to the container
		if errReturn == errMsgStaleReposWithVolumesConfig {
			return true, nil
		}

		// if none of the above errors, return the err
		return false, errors.WithStack(fmt.Errorf("%w: %v", err, errReturn))
	}

	return false, nil
}
