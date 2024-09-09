<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

Patroni configuration is complicated. The daemon `patroni` and the client
`patronictl` are configured slightly differently. Some settings are the same for
the whole cluster, some are different on each instance.

Some things are stored in Kubernetes (our "DCS") and automatically applied by
Patroni every HA reconciliation. Everything else requires a restart or reload
of Patroni to be applied.

Configuration files take precedence over DCS contents.
Environment variables take precedence over configuration files.

`patronictl` uses both the DCS and the Patroni API, so it must be configured for both.
`patroni` takes one required argument, the path to its configuration file(s).

When the configuration path is a directory, the YAML files it contains are
loaded in alphabetical order. Mappings are merged recursively such that later
files (and deeper mappings) take precedence. A key with an undefined or `null`
value removes that key/value from the merged result. (Don't accidentally
generate `null` sections!) Sequences are not merged; a later value overwrites
an earlier one.

---

Given the above, we provide to the user two ways to configure Patroni and thus
PostgreSQL: YAML files and DCS.

Configuration that applies to the whole cluster is in PostgresCluster.Spec and
we copy it into DCS. This allows us to add rules to `pg_hba` and `pg_ident`
for replication and other service accounts. These settings are automatically
applied by Patroni.

Configuration that applies to an individual instance will be in ConfigMaps and
Secrets that get mounted as files into `/etc/patroni`. The user can effectively
use these cluster-wide by referencing the same objects for every instance.
These settings take effect after a Patroni reload.

We will also configure Patroni using YAML files mounted into `/etc/patroni`. To
give these high precedence, they are last in the projected volume and named to
sort last alphabetically.

```
$ ls -1dF /etc/patroni/* /etc/patroni/*/*
/etc/patroni/~~jailbreak.yaml
/etc/patroni/other.yaml
/etc/patroni/~postgres-operator/
/etc/patroni/~postgres-operator/stuff.txt
/etc/patroni/~postgres-operator_x.yaml
/etc/patroni/some.yaml

$ python3
>>> import os
>>> sorted(os.listdir('/etc/patroni'))
['other.yaml', 'some.yaml', '~postgres-operator', '~postgres-operator_x.yaml', '~~jailbreak.yaml']
```

- `/etc/patroni` <br/>
  Use this directory to store Patroni configuration. Files with YAML extensions
  are loaded in alphabetical order.

- `/etc/patroni/~postgres-operator/*` <br/>
  Use this subdirectory to store things like TLS certificates and keys. Files in
  subdirectories are not loaded automatically, but avoid YAML extensions just in
  case.

- ConfigMap `{cluster}-config`, Key `patroni.yaml` →
  `/etc/patroni/~postgres-operator_cluster.yaml`

- ConfigMap `{instance}-config`, Key `patroni.yaml` →
  `/etc/patroni/~postgres-operator_instance.yaml`




- https://github.com/zalando/patroni/blob/v2.0.1/docs/dynamic_configuration.rst
- https://github.com/zalando/patroni/blob/v2.0.1/docs/SETTINGS.rst
- https://github.com/zalando/patroni/blob/v2.0.1/docs/ENVIRONMENT.rst

TODO: document PostgreSQL parameters separately...

# Client and Daemon configuration

| Environment | YAML | DCS | Mutable | C/I | C/D | . |
|-------------|------|-----|---------|-----|-----|---|
| PATRONI_CONFIGURATION | - | - | - | - | both | All configuration as a single YAML document. No files nor other environment are considered.
| PATRONI_SCOPE | scope | No | immutable | cluster  | both | Cluster identifier.
| PATRONI_NAME  | name  | No | immutable | instance | patroni | Instance identifier. Must be Pod.Name for Kubernetes DCS.
||
| PATRONI_LOG_LEVEL           | -                   | -  | -       | -      | patronictl | Logging level. (default: WARNING)
| PATRONI_LOG_LEVEL           | log.level           | No | mutable | either | patroni    | Logging level. (default: INFO)
| PATRONI_LOG_TRACEBACK_LEVEL | log.traceback_level | No | mutable | either | patroni    | Logging level that includes tracebacks. (default: ERROR)
| PATRONI_LOG_FORMAT          | log.format          | No | mutable | either | patroni    | Format of log entries.
| PATRONI_LOG_DATEFORMAT      | log.dateformat      | No | mutable | either | patroni    | Format of log entry timestamps.
| PATRONI_LOG_MAX_QUEUE_SIZE  | log.max_queue_size  | No | mutable | either | patroni    |
| PATRONI_LOG_DIR             | log.dir             | No | mutable | either | patroni    | Directory for log files.
| PATRONI_LOG_FILE_SIZE       | log.file_size       | No | mutable | either | patroni    | Size of log file (in bytes) that triggers rotation. (default: 25MB)
| PATRONI_LOG_FILE_NUM        | log.file_num        | No | mutable | either | patroni    | Number of rotated log files to retain. (default: 4)
| PATRONI_LOG_LOGGERS         | log.loggers         | No | mutable | either | patroni    | Mapping of log levels per Python module. (Environment is YAML.)
||
| PATRONI_RESTAPI_LISTEN          | restapi.listen           | No | mutable | either   | patroni | Address and port on which to bind.
| PATRONI_RESTAPI_CONNECT_ADDRESS | restapi.connect_address  | No | mutable | instance | patroni | How to connect to Patroni from outside the Pod.
| PATRONI_RESTAPI_CERTFILE        | restapi.certfile         | No | mutable | either   | both    | Path to the server certificate. Set this to enable TLS.
| PATRONI_RESTAPI_KEYFILE         | restapi.keyfile          | No | mutable | either   | both    | Path to the server certificate key.
| PATRONI_RESTAPI_CAFILE          | restapi.cafile           | No | mutable | either   | both    | Path to the client certificate authority.
| PATRONI_RESTAPI_VERIFY_CLIENT   | restapi.verify_client    | No | mutable | either   | patroni | Whether or not to verify client certificates. (default: none)
| PATRONI_RESTAPI_USERNAME | restapi.authentication.username | No | mutable | either   | both    | HTTP Basic Authentication for "unsafe" endpoints: DELETE, PATCH, POST, PUT
| PATRONI_RESTAPI_PASSWORD | restapi.authentication.password | No | mutable | either   | both    | HTTP Basic Authentication for "unsafe" endpoints: DELETE, PATCH, POST, PUT
| PATRONI_RESTAPI_HTTP_EXTRA_HEADERS  | restapi.http_extra_headers  | No | mutable | either | patroni | Additional headers for HTTP responses.
| PATRONI_RESTAPI_HTTPS_EXTRA_HEADERS | restapi.https_extra_headers | No | mutable | either | patroni | Additional headers for HTTP responses over TLS.
||
| PATRONICTL_CONFIG_FILE | -            | -  | -       | -      | patronictl | Path to the config file. (default: ~/.config/patroni/patronictl.yaml)
| PATRONI_CTL_INSECURE   | ctl.insecure | No | mutable | either | patronictl | Whether or not to verify the server certificate.
| PATRONI_CTL_CACERT     | ctl.cacert   | No | mutable | either | patronictl | Path to the server certificate authority. (default: restapi.cafile)
| PATRONI_CTL_CERTFILE   | ctl.certfile | No | mutable | either | patronictl | Path to the client certificate. (default: restapi.certfile)
| PATRONI_CTL_KEYFILE    | ctl.keyfile  | No | mutable | either | patronictl | Path to the client certificate key. (default: restapi.keyfile)
||
| PATRONI_KUBERNETES_BYPASS_API_SERVICE | kubernetes.bypass_api_service | No | restart   | either   | both | Resolve the IPs behind the service periodically and use them directly.
| PATRONI_KUBERNETES_USE_ENDPOINTS      | kubernetes.use_endpoints      | No | immutable | cluster  | both | Elect and store state using Endpoints (instead of ConfigMap).
| PATRONI_KUBERNETES_PORTS              | kubernetes.ports              | No | restart   | either   | both | When using Endpoints, port details need to match the leader Service.
| PATRONI_KUBERNETES_LABELS             | kubernetes.labels             | No | immutable | cluster  | both | Used to find objects of the cluster. Patroni writes them on things it creates.
| PATRONI_KUBERNETES_ROLE_LABEL         | kubernetes.role_label         | No | immutable | cluster  | both | Name of the label containing "master", "replica", etc.
| PATRONI_KUBERNETES_SCOPE_LABEL        | kubernetes.scope_label        | No | immutable | cluster  | both | Name of the label containing cluster identifier.
| PATRONI_KUBERNETES_NAMESPACE          | kubernetes.namespace          | No | immutable | cluster  | both |
| PATRONI_KUBERNETES_POD_IP             | kubernetes.pod_ip             | No | immutable | instance | both |
||
| - | watchdog.mode          | Yes¹ | mutable | either | patroni | (default: automatic)
| - | watchdog.device        | Yes¹ | mutable | either | patroni | Path to watchdog device. (default: /dev/watchdog)
| - | watchdog.safety_margin | Yes¹ | mutable | either | patroni | (default: 5)

¹ This section must be entirely in DCS or entirely in YAML.


# PostgreSQL and Failover configuration

Used only by `patroni`, not `patronictl`.

| Environment | YAML | DCS | Mutable | C/I | . |
|-------------|------|-----|---------|-----|---|
| - | ttl           | Only | mutable | cluster | TTL of the leader lock in seconds. (default: 30)
| - | loop_wait     | Only | mutable | cluster | Seconds between HA reconciliations. (default: 10)
| - | retry_timeout | Only | mutable | cluster | Timeout for DCS and PostgreSQL operations in seconds. (default: 10)

There is an implicit relationship between `ttl`, `loop_wait`, and `retry_timeout`.
According to https://github.com/zalando/patroni/issues/1579#issuecomment-641830296,
`ttl` should be greater than the maximum time it may take for a single
synchronization which is `loop_wait` plus two `retry_timeout`. That is,
`ttl > loop_wait + retry_timeout + retry_timeout` because immediately after
acquiring the leader lock, the Patroni leader:

  1. Sleeps until the next scheduled sync (at most `loop_wait`)
  2. Wakes and tries to read from DCS (at most `retry_timeout`)
  3. Decides to release or retain the lock
  4. Then tries to write that to DCS (at most `retry_timeout`)

| Environment | YAML | DCS | Mutable | C/I | . |
|-------------|------|-----|---------|-----|---|
|||||| https://github.com/zalando/patroni/blob/v2.0.1/docs/replication_modes.rst
| - | maximum_lag_on_failover | Only | mutable | cluster | Bytes behind which a replica may not become leader. (default: 1MB)
| - | check_timeline          | Only | mutable | cluster | Whether or not a replica on an older timeline may become leader. (default: false)
| - | max_timelines_history   | Only | mutable | cluster | (default: 0)
| - | synchronous_mode        | Only | mutable | cluster | (default: false)
| - | synchronous_mode_strict | Only | mutable | cluster | (default: false)
| - | synchronous_node_count  | Only | mutable | cluster | (default: 1)
| - | master_stop_timeout     | Yes  | mutable | cluster | (default: 0)
| - | master_start_timeout    | Yes  | mutable | cluster | (default: 300)
||
|||||| Setting `host`, `port`, or `restore_command` enables standby behavior.
| - | standby_cluster.create_replica_methods   | Only | immutable | cluster | List of methods to use when creating a standby leader. See `postgresql.create_replica_methods`. (default: basebackup)
| - | standby_cluster.host                     | Only | immutable | cluster | Address to dial for streaming replication.
| - | standby_cluster.port                     | Only | immutable | cluster |
| - | standby_cluster.primary_slot_name        | Only | immutable | cluster |
| - | standby_cluster.restore_command          | Only | immutable | cluster | Override "postgresql.parameters.restore_command" on leader and replicas.
| - | standby_cluster.archive_cleanup_command  | Only | immutable | cluster |
| - | standby_cluster.recovery_min_apply_delay | Only | immutable | cluster |
||
| - | tags.nofailover    | No | mutable | instance | Whether or not this instance can be leader. (default: false)
| - | tags.nosync        | No | mutable | instance | Whether or not this instance can be synchronous replica. (default: false)
| - | tags.clonefrom     | No | mutable | instance | Whether or not this instance is preferred source for pg_basebackup. (default: false)
| - | tags.noloadbalance | No | mutable | instance | Whether or not `/replica` endpoint ever returns success. (default: false)
| - | tags.replicatefrom | No | mutable | instance | The address of another replica for cascading replication.
||
| PATRONI_POSTGRESQL_LISTEN          | postgresql.listen          | No   | ?       | either   | Addresses and port on which to bind. Patroni uses the first address for local connections.
| PATRONI_POSTGRESQL_CONNECT_ADDRESS | postgresql.connect_address | No   | ?       | instance | How to connect to PostgreSQL from outside the Pod.
| -                                  | postgresql.use_unix_socket | Yes  | mutable | either   | Prefer to use sockets. (default: false)
| PATRONI_POSTGRESQL_DATA_DIR        | postgresql.data_dir        | No   | ?       | either   | Location of the PostgreSQL data directory.
| PATRONI_POSTGRESQL_CONFIG_DIR      | postgresql.config_dir      | Yes  | ?       | either   | Location of the writable PostgreSQL config directory, defaults to data directory.
| PATRONI_POSTGRESQL_BIN_DIR         | postgresql.bin_dir         | Yes  | ?       | either   | Location of the PostgreSQL binaries. Empty means use PATH.
| PATRONI_POSTGRESQL_PGPASS          | postgresql.pgpass          | No   | mutable | either   | Location of the writable password file.
| -                                  | postgresql.recovery_conf   | Yes  | mutable | either   | Mapping of additional settings written to recovery.conf of follower. (replica?)
| -                                  | postgresql.custom_conf     | Yes  | mutable | cluster  | Path to a custom configuration file instead of `postgresql.base.conf`.
| -                                  | postgresql.parameters      | Yes  | mutable | either   | PostgreSQL parameters.
| -                                  | postgresql.pg_hba          | Yes  | mutable | either   | The entirety of pg_hba.conf as lines.
| -                                  | postgresql.pg_ident        | Yes  | mutable | either   | The entirety of pg_ident.conf as lines.
| -                                  | postgresql.pg_ctl_timeout  | Yes  | mutable | either   | Timeout when performing start, stop, or restart. (default: 60s)
| -                                  | postgresql.use_pg_rewind   | Yes  | mutable | either   | Whether or not to use pg_rewind when a former leader rejoins the cluster. (default: false)
| -                                  | postgresql.use_slots       | Only | mutable | either   | Whether or not to use replication slots. (default: true)
||
| - | postgresql.remove_data_directory_on_rewind_failure     | Yes | mutable | either |
| - | postgresql.remove_data_directory_on_diverged_timelines | Yes | mutable | either |
||
| - | postgresql.callbacks.on_reload      | Yes¹ | mutable | either | Command to execute when (before? after?) (Patroni? PostgreSQL?) configuration reloads.
| - | postgresql.callbacks.on_restart     | Yes¹ | mutable | either | Command to execute when (before? after?) PostgreSQL restarts.
| - | postgresql.callbacks.on_role_change | Yes¹ | mutable | either | Command to execute when (before? after?) the instance is promoted or demoted.
| - | postgresql.callbacks.on_start       | Yes¹ | mutable | either | Command to execute when (before? after?) PostgreSQL starts.
| - | postgresql.callbacks.on_stop        | Yes¹ | mutable | either | Command to execute when (before? after?) PostgreSQL stops.
||
|||||| https://github.com/zalando/patroni/blob/v2.0.1/docs/replica_bootstrap.rst#building-replicas
| - | postgresql.create_replica_methods | Yes  | mutable | either | List of methods to use when creating a replica. (default: basebackup)
| - | postgresql.basebackup             | Yes  | mutable | either | List of arguments to pass to pg_basebackup when using the `basebackup` replica method.
| - | postgresql.{method}.command       | Yes¹ | mutable | either | Command to execute for this replica method.
| - | postgresql.{method}.keep_data     | Yes¹ | mutable | either | Whether or not Patroni should empty the data directory before. (default: false)
| - | postgresql.{method}.no_master     | Yes¹ | mutable | either | Whether or not Patroni can call this method when no instances are running. (default: false)
| - | postgresql.{method}.no_params     | Yes¹ | mutable | either | Whether or not Patroni should pass extra arguments to the command. (default: false)
||
|||||| https://github.com/zalando/patroni/blob/v2.0.1/docs/replica_bootstrap.rst#bootstrap
| - | bootstrap.method                 | No | immutable | cluster | Method to use when initializing a new cluster. (default: initdb)
| - | bootstrap.initdb                 | No | immutable | cluster | List of arguments to pass to initdb when using the `initdb` bootstrap method.
| - | bootstrap.{method}.command       | No | immutable | cluster | Command to execute for this bootstrap method.
| - | bootstrap.{method}.no_params     | No | immutable | cluster | Whether or not Patroni should pass extra arguments to the command. (default: false)
| - | bootstrap.{method}.recovery_conf | No | immutable | cluster | Mapping of recovery settings. Before PostgreSQL 12, these go into a special file.
| - | bootstrap.{method}.keep_existing_recovery_conf | No | immutable | cluster | Whether or not Patroni should remove signal files.
||
| - | bootstrap.dcs                       | No | immutable | cluster | Mapping to load into DCS when initializing a new cluster.
| - | bootstrap.pg_hba                    | No | immutable | cluster | Lines of HBA to use when no `postgresql.pg_hba` nor `postgresql.parameters.hba_file`.
| - | bootstrap.post_bootstrap            | No | immutable | cluster | Command to execute after PostgreSQL is initialized and running but before "users" below. (string)
| - | bootstrap.users.{username}.options  | No | immutable | cluster | List of options for `CREATE ROLE` SQL.
| - | bootstrap.users.{username}.password | No | immutable | cluster | Password for the role. (optional)
||
| PATRONI_SUPERUSER_USERNAME        | postgresql.authentication.superuser.username        | No | immutable | cluster | Used during initdb and later to connect. (optional)
| PATRONI_SUPERUSER_PASSWORD        | postgresql.authentication.superuser.password        | No | immutable | cluster | Used during initdb and later to connect. (optional)
| PATRONI_SUPERUSER_SSLMODE         | postgresql.authentication.superuser.sslmode         | No | mutable   | either  |
| PATRONI_SUPERUSER_SSLCERT         | postgresql.authentication.superuser.sslcert         | No | mutable   | either  | Path to the client certificate.
| PATRONI_SUPERUSER_SSLKEY          | postgresql.authentication.superuser.sslkey          | No | mutable   | either  | Path to the client certificate key.
| PATRONI_SUPERUSER_SSLPASSWORD     | postgresql.authentication.superuser.sslpassword     | No | mutable   | either  | Password for the client certificate key.
| PATRONI_SUPERUSER_SSLROOTCERT     | postgresql.authentication.superuser.sslrootcert     | No | mutable   | either  | Path to the server certificate authority.
| PATRONI_SUPERUSER_SSLCRL          | postgresql.authentication.superuser.sslcrl          | No | mutable   | either  | Path to the server CRL.
| PATRONI_SUPERUSER_CHANNEL_BINDING | postgresql.authentication.superuser.channel_binding | No | mutable   | either  | Applicable when using SCRAM auth over SSL.
||
| PATRONI_REPLICATION_USERNAME        | postgresql.authentication.replication.username        | No | immutable | cluster | Used during bootstrap and later to connect.
| PATRONI_REPLICATION_PASSWORD        | postgresql.authentication.replication.password        | No | immutable | cluster | Used during bootstrap and later to connect. (optional)
| PATRONI_REPLICATION_SSLMODE         | postgresql.authentication.replication.sslmode         | No | mutable   | either  |
| PATRONI_REPLICATION_SSLCERT         | postgresql.authentication.replication.sslcert         | No | mutable   | either  | Path to the client certificate.
| PATRONI_REPLICATION_SSLKEY          | postgresql.authentication.replication.sslkey          | No | mutable   | either  | Path to the client certificate key.
| PATRONI_REPLICATION_SSLPASSWORD     | postgresql.authentication.replication.sslpassword     | No | mutable   | either  | Password for the client certificate key.
| PATRONI_REPLICATION_SSLROOTCERT     | postgresql.authentication.replication.sslrootcert     | No | mutable   | either  | Path to the server certificate authority.
| PATRONI_REPLICATION_SSLCRL          | postgresql.authentication.replication.sslcrl          | No | mutable   | either  | Path to the server CRL.
| PATRONI_REPLICATION_CHANNEL_BINDING | postgresql.authentication.replication.channel_binding | No | mutable   | either  | Applicable when using SCRAM auth over SSL.
||
| PATRONI_REWIND_* | postgresql.authentication.rewind.* | No | " | " | Same as above. (Patroni uses superuser when this is not set.)

¹ This section must be entirely in DCS or entirely in YAML.
