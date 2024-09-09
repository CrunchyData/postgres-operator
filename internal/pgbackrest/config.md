<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

# pgBackRest Configuration Overview

The initial pgBackRest configuration for the Postgres Clusters is designed to stand up a 
minimal configuration for use by the various pgBackRest functions needed by the Postgres
cluster. These settings are meant to be the minimally required settings, with other
settings supported through the use of custom configurations.

During initial cluster creation, four pgBackRest use cases are involved. 

These settings are configured in either the [global] or [stanza] sections of the 
pgBackRest configuration based on their designation in the pgBackRest code.
For more information on the above, and other settings, please see
https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/config/parse.auto.c

As shown, the settings with the `cfgSectionGlobal` designation are

`log-path`: The log path provides a location for pgBackRest to store log files.

`log-level-file`: Level for file logging. Set to 'off' when the repo host has no volume.

`repo-path`: Path where backups and archive are stored. 
             The repository is where pgBackRest stores backups and archives WAL segments.

`repo-host`: Repository host when operating remotely via TLS.


The settings with the `cfgSectionStanza` designation are

`pg-host`: PostgreSQL host for operating remotely via TLS.

`pg-path`: The path of the PostgreSQL data directory.
		       This should be the same as the data_directory setting in postgresql.conf.

`pg-port`: The port that PostgreSQL is running on.

`pg-socket-path`: The unix socket directory that is specified when PostgreSQL is started.

For more information on these and other configuration settings, please see
`https://pgbackrest.org/configuration.html`.

# Configuration Per Function

Below, each of the four configuration sets is outlined by use case. Please note that certain 
settings have acceptable defaults for the cluster's usage (such as for `repo1-type` which 
defaults to `posix`), so those settings are not included.


1. Primary Database Pod 

[global]
log-path
repo1-host
repo1-path

[stanza]
pg1-path
pg1-port
pg1-socket-path

2. pgBackRest Repo Pod

[global]
log-path
repo1-path
log-level-file

[stanza]
pg1-host
pg1-path
pg1-port
pg1-socket-path

3. pgBackRest Stanza Job Pod

[global]
log-path

4. pgBackRest Backup Job Pod

[global]
log-path


# Initial pgBackRest Configuration

In order to be used by the Postgres cluster, these default configurations are stored in
a configmap. This configmap is named with the following convention `<clustername>-pgbackrest-config`, 
such that a cluster named 'mycluster' would have a configuration configmap named
`mycluster-pgbackrest-config`.

As noted above, there are three distinct default configurations, each of which is referenced 
by a key value in the configmap's data section. For the primary database pod, the key is
`pgbackrest_primary.conf`. For the pgBackRest repo pod, the key is `pgbackrest_repo.conf`.
Finally, for the pgBackRest stanza job pod and the initial pgBackRest backup job pod, the
key is `pgbackrest_job.conf`.
	
For each pod, the relevant configuration file is mounted as a projected volume named 
`pgbackrest-config-vol`. The configuration file will be found in the `/etc/pgbackrest` directory
of the relevant container and is named `pgbackrest.conf`, matching the default pgBackRest location. 
For more information, please see 
`https://pgbackrest.org/configuration.html#introduction`


# Custom Configuration Support

TODO(tjmoore4): Document custom configuration solution once implemented

Custom pgBackRest configurations is supported by using the `--config-include-path`
flag with the desired pgBackRest command. This should point to the directory path
where the `*.conf` file with the custom configuration is located.

This file will be added as a projected volume and must be formatted in the standard
pgBackRest INI convention. Please note that any of the configuration settings listed 
above MUST BE CONFIGURED VIA THE POSTGRESCLUSTER SPEC so as to avoid errors.

For more information, please see
`https://pgbackrest.org/user-guide.html#quickstart/configure-stanza`.

---

There are three ways to configure pgBackRest: INI files, environment variables,
and command-line arguments. Any particular option comes from exactly one of those
places. For example, when an option is in an INI file and a command-line argument,
only the command-line argument is used. This is true even for options that can
be specified more than once. The [precedence](https://pgbackrest.org/command.html#introduction):

> Command-line options override environment options which override config file options.

From one of those places, only a handful of options may be set more than once
(see `PARSE_RULE_OPTION_MULTI` in [parse.auto.c][]). The resulting value of
these options matches the order in which they were loaded: left-to-right on the
command-line or top-to-bottom in INI files.

The remaining options must be set exactly once. `pgbackrest` exits non-zero when
the option occurs twice on the command-line or twice in a file:

```
ERROR: [031]: option 'io-timeout' cannot be set multiple times
```

A few options are only allowed in certain places. Credentials, for example,
cannot be passed as command-line arguments (see `PARSE_RULE_OPTION_SECURE` in [parse.auto.c][]).
Some others cannot be in INI files (see `cfgSectionCommandLine` in [parse.auto.c][]).
Notably, these must be environment variables or command-line arguments:

- `--repo` and `--stanza`
- restore `--target` and `--target-action`
- backup and restore `--type`

pgBackRest looks for and loads multiple INI files from multiple places according
to the `config`, `config-include-path`, and/or `config-path` options. The order
is a [little complicated][file-precedence]. When none of these options are set:

 1. One of `/etc/pgbackrest/pgbackrest.conf` or `/etc/pgbackrest.conf` is read
    in that order, [whichever exists][default-config].
 2. All `/etc/pgbackrest/conf.d/*.conf` files that exist are read in alphabetical order.

There is no "precedence" between these files; they do not "override" each other.
Options that can be set multiple times are interpreted as each file is loaded.
Options that cannot be set multiple times will error when they are in multiple files.

There *is* precedence, however, *inside* these files, organized by INI sections.

- The "global"             section applies to all repositories, stanzas, and commands.
- The "global:*command*"   section applies to all repositories and stanzas for a particular command.
- The "*stanza*"           section applies to all repositories and commands for a particular stanza.
- The "*stanza*:*command*" section applies to all repositories for a particular stanza and command.

Options in more specific sections (lower in the list) [override][file-precedence]
options in less specific sections.

[default-config]:  https://pgbackrest.org/configuration.html#introduction
[file-precedence]: https://pgbackrest.org/user-guide.html#quickstart/configure-stanza
[parse.auto.c]: https://github.com/pgbackrest/pgbackrest/blob/release/2.38/src/config/parse.auto.c

```console
$ tail -vn+0 pgbackrest.conf conf.d/*
==> pgbackrest.conf <==
[global]
exclude = main
exclude = main
io-timeout = 10
link-map = x=x1
link-map = x=x2
link-map = y=y1

[global:backup]
io-timeout = 20

[db]
io-timeout = 30
link-map = y=y2

[db:backup]
io-timeout = 40

==> conf.d/one.conf <==
[global]
exclude = one

==> conf.d/two.conf <==
[global]
exclude = two

==> conf.d/!three.conf <==
[global]
exclude = three

==> conf.d/~four.conf <==
[global]
exclude = four

$ pgbackrest --config-path="$(pwd)" help backup | grep -A1 exclude
  --exclude                        exclude paths/files from the backup
                                   [current=main, main, three, one, two, four]

$ pgbackrest --config-path="$(pwd)" help backup --exclude=five | grep -A1 exclude
  --exclude                        exclude paths/files from the backup
                                   [current=five]

$ pgbackrest --config-path="$(pwd)" help backup | grep io-timeout
  --io-timeout                     I/O timeout [current=20, default=60]

$ pgbackrest --config-path="$(pwd)" help backup --stanza=db | grep io-timeout
  --io-timeout                     I/O timeout [current=40, default=60]

$ pgbackrest --config-path="$(pwd)" help info | grep io-timeout
  --io-timeout                     I/O timeout [current=10, default=60]

$ pgbackrest --config-path="$(pwd)" help info --stanza=db | grep io-timeout
  --io-timeout                     I/O timeout [current=30, default=60]

$ pgbackrest --config-path="$(pwd)" help restore | grep -A1 link-map
  --link-map                       modify the destination of a symlink
                                   [current=x=x2, y=y1]

$ pgbackrest --config-path="$(pwd)" help restore --stanza=db | grep -A1 link-map
  --link-map                       modify the destination of a symlink
                                   [current=y=y2]
```

---

Given all the above, we configure pgBackRest using files mounted into the
`/etc/pgbackrest/conf.d` directory. They are last in the projected volume to
ensure they take precedence over other projections.

- `/etc/pgbackrest/conf.d` <br/>
  Use this directory to store pgBackRest configuration. Files ending with `.conf`
  are loaded in alphabetical order.

- `/etc/pgbackrest/conf.d/~postgres-operator/*` <br/>
  Use this subdirectory to store things like TLS certificates and keys. Files in
  subdirectories are not loaded automatically.
