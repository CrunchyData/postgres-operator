<!--
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
https://github.com/pgbackrest/pgbackrest/blob/master/src/config/parse.auto.c

As shown, the settings with the `cfgSectionGlobal` designation are

`log-path`: The log path provides a location for pgBackRest to store log files.

`repo-path`: Path where backups and archive are stored. 
             The repository is where pgBackRest stores backups and archives WAL segments.

`repo-host`: Repository host when operating remotely via SSH.


The settings with the `cfgSectionStanza` designation are

`pg-host`: PostgreSQL host for operating remotely via SSH.

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
