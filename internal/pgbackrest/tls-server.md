<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

# pgBackRest TLS Server

A handful of pgBackRest features require connectivity between `pgbackrest` processes
on different pods:

- [dedicated repository host](https://pgbackrest.org/user-guide.html#repo-host)
- [backup from standby](https://pgbackrest.org/user-guide.html#standby-backup)

When a PostgresCluster is configured to store backups on a PVC, the dedicated
repository host is used to make that PVC available to all PostgreSQL instances
in the cluster. Regardless of whether the repo host has a defined PVC, it
functions as the server for the pgBackRest clients that run on the Instances.

The repository host runs a `pgbackrest` server that is secured through TLS and
[certificates][]. When performing backups, it connects to `pgbackrest` servers
running on PostgreSQL instances (as sidecars). Restore jobs connect to the
repository host to fetch files. PostgreSQL calls `pgbackrest` which connects
to the repository host to [send and receive WAL files][archiving].

[archiving]: https://www.postgresql.org/docs/current/continuous-archiving.html
[certificates]: certificates.md


The `pgbackrest` command acts as a TLS client and connects to a pgBackRest TLS
server when `pg-host-type=tls` and/or `repo-host-type=tls`. The default for these is `ssh`:

- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/config/parse.auto.c#L3771
- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/config/parse.auto.c#L6137


The pgBackRest TLS server is configured through the `tls-server-*` [options](config.md).
In pgBackRest 2.38, changing any of these options or changing certificate contents
requires a reload of the server, as shown in the "Setup TLS Server" section of the
documentation, with the command configured as

```
ExecReload=kill -HUP $MAINPID
```

- https://pgbackrest.org/user-guide-rhel.html#repo-host/setup-tls

- `tls-server-address`, `tls-server-port` <br/>
  The network address and port on which to listen. pgBackRest 2.38 listens on
  the *first* address returned by `getaddrinfo()`. There is no way to listen on
  all interfaces.

  - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/io/socket/server.c#L172
  - https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/io/socket/common.c#L87

- `tls-server-cert-file`, `tls-server-key-file` <br/>
  The [certificate chain][certificates] and private key pair used to encrypt connections.

- `tls-server-ca-file` <br/>
  The certificate used to verify client [certificates][].
  [Required](https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/config/parse.auto.c#L8767).

- `tls-server-auth` <br/>
  A map/hash/dictionary of certificate common names and the stanzas they are authorized
  to interact with.
  [Required](https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/config/parse.auto.c#L8751).


In pgBackRest 2.38, as mentioned above, sending SIGHUP causes a configuration reload.

- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/command/server/server.c#L178

```
P00 DETAIL: configuration reload begin
P00   INFO: server command begin 2.38...
P00 DETAIL: configuration reload end
```

Sending SIGINT to the TLS server causes it to exit with code 63, TermError.

- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/exit.c#L73-L75
- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/exit.c#L62
- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/common/error.auto.c#L48


```
P00   INFO: server command end: terminated on signal [SIGINT]
```

Sending SIGTERM exits the signal loop and lead to the command termination.

- https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/command/server/server.c#L194


```
P00   INFO: server command end: completed successfully
```
