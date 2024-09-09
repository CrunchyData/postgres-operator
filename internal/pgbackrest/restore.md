<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

## Target Action

The `--target-action` option of `pgbackrest restore` almost translates to the
PostgreSQL `recovery_target_action` parameter but not exactly. The behavior of
that parameter also depends on the PostgreSQL version and on other parameters.

For PostgreSQL 9.5 through 15,

 - The PostgreSQL documentation states that for `recovery_target_action`
   "the default is `pause`," but that is only the case when `hot_standby=on`.

 - The PostgreSQL documentation states that when `hot_standby=off` "a setting
   of `pause` will act the same as `shutdown`," but that cannot be configured
   through pgBackRest.

The default value of `hot_standby` is `off` prior to PostgreSQL 10 and `on` since.

### PostgreSQL 15, 14, 13, 12

[12]: https://www.postgresql.org/docs/12/runtime-config-wal.html
[commit]: https://git.postgresql.org/gitweb/?p=postgresql.git;h=2dedf4d9a899b36d1a8ed29be5efbd1b31a8fe85

| --target-action  | recovery_target_action | hot_standby=off | hot_standby=on (default) |
|------------------|------------------------|-----------------|--------------------------|
| _not configured_ | _not configured_       | shutdown        | pause                    |
| `pause`          | _not configured_       | shutdown        | pause                    |
| _not possible_   | `pause`                | shutdown        | pause                    |
| `promote`        | `promote`              | promote         | promote                  |
| `shutdown`       | `shutdown`             | shutdown        | shutdown                 |


### PostgreSQL 11, 10

[11]: https://www.postgresql.org/docs/11/recovery-target-settings.html
[10]: https://www.postgresql.org/docs/10/runtime-config-replication.html

| --target-action  | recovery_target_action | hot_standby=off | hot_standby=on (default) |
|------------------|------------------------|-----------------|--------------------------|
| _not configured_ | _not configured_       | promote         | pause                    |
| `pause`          | _not configured_       | promote         | pause                    |
| _not possible_   | `pause`                | shutdown        | pause                    |
| `promote`        | `promote`              | promote         | promote                  |
| `shutdown`       | `shutdown`             | shutdown        | shutdown                 |


### PostgreSQL 9.6, 9.5

[9.6]: https://www.postgresql.org/docs/9.6/recovery-target-settings.html

| --target-action  | recovery_target_action | hot_standby=off (default) | hot_standby=on |
|------------------|------------------------|---------------------------|----------------|
| _not configured_ | _not configured_       | promote                   | pause          |
| `pause`          | _not configured_       | promote                   | pause          |
| _not possible_   | `pause`                | shutdown                  | pause          |
| `promote`        | `promote`              | promote                   | promote        |
| `shutdown`       | `shutdown`             | shutdown                  | shutdown       |


### PostgreSQL 9.4, 9.3, 9.2, 9.1

[9.4]: https://www.postgresql.org/docs/9.4/recovery-target-settings.html
[9.4]: https://www.postgresql.org/docs/9.4/runtime-config-replication.html

| --target-action  | pause_at_recovery_target | hot_standby=off (default) | hot_standby=on |
|------------------|--------------------------|---------------------------|----------------|
| _not configured_ | _not configured_         | promote                   | pause          |
| `pause`          | _not configured_         | promote                   | pause          |
| _not possible_   | `true`                   | promote                   | pause          |
| `promote`        | `false`                  | promote                   | promote        |


<!--

### Setup

# Change to a directory with enough space to restore and choose a data directory.

cd /pgdata
export PGDATA="$(pwd)/test"

# Do a full restore then start PostgreSQL. It will run in the foreground, replay,
# and promote. Notice the LSN in the "consistent recovery state reached" message.
# The "selected new timeline" message indicates that it promoted.
# Use ^C to shutdown.

pgbackrest restore --pg1-path="$PGDATA" --stanza=db
(cd "$PGDATA"; postgres -c archive_command=false -c logging_collector=off -c port=9999)


### Test

# Delete the data directory and perform a PITR.
# Start PostgreSQL with hot_standby=on, and it will replay then pause.
# Notice the "pausing at the end of recovery" message. Use ^C to shutdown.

rm -rf "$PGDATA"
pgbackrest restore --pg1-path="$PGDATA" --stanza=db --type=lsn --target="$LSN"
(cd "$PGDATA"; postgres -c archive_command=false -c logging_collector=off -c port=9999 -c hot_standby=on)

# Repeat the test with any pgBackRest and PostgreSQL settings you like. Look
# in PostgreSQL conf files to see what is or is not configured.

grep recovery_target_action "$PGDATA"/*.conf

-->
