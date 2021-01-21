---
title: "Custom Configuration"
date:
draft: false
weight: 4
---

## Custom PostgreSQL Configuration

Users and administrators can specify a
custom set of PostgreSQL configuration files to be used when creating
a new PostgreSQL cluster.  The configuration files you can
change include -

 * postgres-ha.yaml
 * setup.sql

Different configurations for PostgreSQL might be defined for
the following -

 * OLTP types of databases
 * OLAP types of databases
 * High Memory
 * Minimal Configuration for Development
 * Project Specific configurations
 * Special Security Requirements

#### Global ConfigMap

If you create a *configMap* called *pgo-custom-pg-config* with any
of the above files within it, new clusters will use those configuration
files when setting up a new database instance.  You do *NOT* have to
specify all of the configuration files. It is entirely up to your use case
to determine which to use.

An example set of configuration files and a script to create the
global configMap is found at
```
$PGOROOT/examples/custom-config
```

If you run the *create.sh* script there, it will create the configMap
that will include the PostgreSQL configuration files within that directory.

#### Config Files Purpose

The *postgres-ha.yaml* file is the main configuration file that allows for the
configuration of a wide variety of tuning parameters for you PostgreSQL cluster.
This includes various PostgreSQL settings, e.g. those that should be applied to
files such as `postgresql.conf`, `pg_hba.conf` and `pg_ident.conf`, as well as
tuning parameters for the High Availability features included in each cluster.
The various configuration settings available can be
[found here](https://access.crunchydata.com/documentation/patroni/latest/settings/index.html#settings)

The *setup.sql* file is a SQL file that is executed following the initialization
of a new PostgreSQL cluster, specifically after *initdb* is run when the database
is first created. Changes would be made to this if you wanted to define which
database objects are created by default.

#### Granular Config Maps

Granular config maps can be defined if it is necessary to use
a different set of configuration files for different clusters
rather than having a single configuration (e.g. Global Config Map).
A specific set of ConfigMaps with their own set of PostgreSQL
configuration files can be created. When creating new clusters, a
`--custom-config` flag can be passed along with the name of the
ConfigMap which will be used for that specific cluster or set of
clusters.

#### Defaults

If there is no reason to change the default PostgreSQL configuration
files that ship with the Crunchy Postgres container, there is no
requirement to. In this event, continue using the Operator as usual
and avoid defining a global configMap.

## Create a PostgreSQL Cluster With Custom Configuration

The PostgreSQL Operator allows for a PostgreSQL cluster to be created with a customized configuration. To do this, one must create a ConfigMap with an entry called `postgres-ha.yaml` that contains the custom configuration. The custom configuration follows the [Patorni YAML format](https://access.crunchydata.com/documentation/patroni/latest/settings/). Note that parameters that are placed in the `bootstrap` section are applied once during cluster initialization. Editing these values in a working cluster require following the [Modifying PostgreSQL Cluster Configuration](#modifying-postgresql-cluster-configuration) section.

For example, let's say we want to create a PostgreSQL cluster with `shared_buffers` set to `2GB`, `max_connections` set to `30` and `password_encryption` set to `scram-sha-256`. We would create a configuration file that looks similar to:

```
---
bootstrap:
  dcs:
    postgresql:
      parameters:
        max_connections: 30
        shared_buffers: 2GB
        password_encryption: scram-sha-256
```

Save this configuration in a file called `postgres-ha.yaml`.

Create a [`ConfigMap`](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) like so:

```
kubectl -n pgo create configmap hippo-custom-config --from-file=postgres-ha.yaml
```

You can then have you new PostgreSQL cluster use `hippo-custom-config` as part of its cluster initialization by using the `--custom-config` flag of `pgo create cluster`:

```
pgo create cluster hippo -n pgo --custom-config=hippo-custom-config
```

After your cluster is initialized, [connect to your cluster]({{< relref "tutorial/connect-cluster.md" >}}) and confirm that your settings have been applied:

```
SHOW shared_buffers;

 shared_buffers
----------------
 2GB
```

## Modifying PostgreSQL Cluster Configuration

Once a PostgreSQL cluster has been initialized, its configuration settings
can be updated and modified as needed.  This done by modifying the
`<clusterName>-pgha-config` ConfigMap that is created for each individual
PostgreSQL cluster.

The `<clusterName>-pgha-config` ConfigMap is populated following cluster
initializtion, specifically using the baseline configuration settings used to
bootstrap the cluster.  Therefore, any customiztions applied using a custom
`postgres-ha.yaml` file as described in the **Custom PostgreSQL Configuration**
section above will also be included when the ConfigMap is populated.

The various configuration settings available for modifying and updating
and cluster's configuration can be
[found here](https://access.crunchydata.com/documentation/patroni/latest/settings/index.html#settings).
Please proceed with caution when modiying configuration, especially those settings
applied by default by Operator.  Certain settings are required for normal operation
of the Operator and the PostgreSQL clusters it creates, and altering these
settings could result in expected behavior.

### Types of Configuration

Within the `<clusterName>-pgha-config` ConfigMap are two forms of configuration:

- **Distributed Configuration Store (DCS):** Cluster-wide
configuration settings that are applied to all database servers in the PostgreSQL
cluster
- **Local Database:** Configuration settings that are applied
individually to each database server (i.e. the primary and each replica) within
the cluster.

The DCS configuration settings are stored within the `<clusterName>-pgha-config`
ConfigMap in a configuration named `<clusterName>-dcs-config`, while the local
database configurations are stored in one or more configurations named
`<serverName>-local-config` (with one local configuration for the primary and each
replica within the cluster).  Please note that
[as described here](https://access.crunchydata.com/documentation/patroni/latest/dynamic_configuration/),
certain settings can only be applied via the DCS to ensure they are uniform among
the primary and all replicas within the cluster.

The following is an example of the both the DCS and primary configuration settings
as stored in the `<clusterName>-pgha-config` ConfigMap for a cluster named `mycluster`.
Please note the `mycluster-dcs-config` configuration defining the DCS configuration
for `mycluster`, along with the `mycluster-local-config` configuration defining the
local configuration for the database server named `mycluster`, which is the current
primary within the PostgreSQL cluster.

```bash
$ kubectl describe cm mycluster-pgha-config   
Name:         mycluster-pgha-config
Namespace:    pgouser1
Labels:       pg-cluster=mycluster
              pgha-config=true
              vendor=crunchydata
Annotations:  <none>

Data
====
mycluster-dcs-config:
----
postgresql:
  parameters:
    archive_command: source /opt/cpm/bin/pgbackrest/pgbackrest-set-env.sh && pgbackrest
      archive-push "%p"
    archive_mode: true
    archive_timeout: 60
    log_directory: pg_log
    log_min_duration_statement: 60000
    log_statement: none
    max_wal_senders: 6
    shared_buffers: 128MB
    shared_preload_libraries: pgaudit.so,pg_stat_statements.so
    temp_buffers: 8MB
    unix_socket_directories: /tmp
    wal_level: logical
    work_mem: 4MB
  recovery_conf:
    restore_command: source /opt/cpm/bin/pgbackrest/pgbackrest-set-env.sh && pgbackrest
      archive-get %f "%p"
  use_pg_rewind: true

mycluster-local-config:
----
postgresql:
  callbacks:
    on_role_change: /opt/cpm/bin/callbacks/pgha-on-role-change.sh
  create_replica_methods:
  - pgbackrest
  - basebackup
  pg_hba:
  - local all postgres peer
  - host replication primaryuser 0.0.0.0/0 md5
  - host all primaryuser 0.0.0.0/0 reject
  - host all all 0.0.0.0/0 md5
  pgbackrest:
    command: /opt/cpm/bin/pgbackrest/pgbackrest-create-replica.sh
    keep_data: true
    no_params: true
  pgbackrest_standby:
    command: /opt/cpm/bin/pgbackrest/pgbackrest-create-replica.sh
    keep_data: true
    no_master: 1
    no_params: true
  pgpass: /tmp/.pgpass
  remove_data_directory_on_rewind_failure: true
  use_unix_socket: true
```

### Updating Configuration Settings

In order to update a cluster's configuration settings and then apply
those settings (e.g. to the DCS and/or any individual database servers), the
DCS and local configuration settings within the `<clusterName>-pgha-config`
ConfigMap can be modified.  This can be done using the various commands
available using the `kubectl` client (or the `oc` client if using OpenShift)
for modifying Kubernetes resources. For instance, the following command can be
utilized to open the ConfigMap in a local text editor, and then update the
various cluster configurations as needed:

```bash
kubectl edit configmap mycluster-pgha-config
```

Once the `<clusterName>-pgha-config` ConfigMap has been updated, any
changes made will be detected by the Operator, and then applied to the
DCS and/or any individual database servers within the cluster.

#### PostgreSQL Configuration

In order to update the `postgresql.conf` file for a one of more database servers, the
`parameters` section of either the DCS and/or a local database configuration can be
updated, e.g.:

```yaml
----
postgresql:
  parameters:
    max_wal_senders: 10
```

The various key/value pairs provided within the `parameters` section result in the
configuration of the same settings within the `postgresql.conf` file.  Please note that
settings applied locally to a database server take precedence over those set via the DCS (with the
exception being those that must be set via the DCS, as
[described here](https://access.crunchydata.com/documentation/patroni/latest/dynamic_configuration/)).

Also, please note that `pg_hba` and `pg_ident` sections exist to update both the `pg_hba.conf` and
`pg_ident.conf` PostgreSQL configuration files as needed.

#### A Note on Customizing `authentication`

One of the blocks that can be modified in a `local` database setting is the
`authentication` block. This can be useful for setting customizations such as
TLS connection requirements (`sslmode`). However, one should take care when
modifying this block, as modifying certain parameters can interfere with the
management features that the PostgreSQL Operator provides.

In particular, one should **not** customize the `username` or `password`
attributes within this section as that will interface with the PostgreSQL
Operator. Additionally, is using the built-in support for certificate-based
authentication for replication users, you should not modify the `sslcert`,
`sslkey`, `sslrootcert`, and `sslcrl` entries in the `replication` block of the
`authentication` block.

### Restarting Database Servers

Changes to certain settings may require one or more PostgreSQL databases within the cluster to be
restarted.  This can be accomplished using the `pgo restart` command included with the `pgo` client.
To detect if a restart is needed for a instance within a cluster called `mycluster` after making a
configuration change, the `query` flag can be utilized with the `pgo restart` command as follows:

```bash
$ pgo restart mycluster2 --query

Cluster: mycluster2
INSTANCE                ROLE            STATUS          NODE            REPLICATION LAG         PENDING RESTART
mycluster               primary         running         node01                     0 MB                   false
mycluster-ambq          replica         running         node01                     0 MB                    true
```

Here we can see that the `mycluster-ambq` instance (i.e. the sole replica in cluster `mycluster`)
is pending a restart, as shown by the `PENDING RESTART` column.  A restart can then be requested
as follows:

```bash
$ pgo restart mycluster --target mycluster-ambq
WARNING: Are you sure? (yes/no): yes
Successfully restarted instance mycluster
```

It is also possible to target multiple instances at the same time:

```bash
$ pgo restart mycluster --target mycluster --target mycluster-ambq
WARNING: Are you sure? (yes/no): yes
Successfully restarted instance mycluster
Successfully restarted instance mycluster-ambq
```

Or if no target is specified, the all instances within the cluster will be restarted:

```bash
$ pgo restart mycluster
WARNING: Are you sure? (yes/no): yes
Successfully restarted instance mycluster
Successfully restarted instance mycluster-ambq
```

### Refreshing Configuration Settings

If necessary, it is possible to refresh the configuration stored within the
`<clusterName>-pgha-config` ConfigMap with a fresh copy of either the DCS
configuration and/or the configuration for one or more local database servers.
This is specifically done by fully deleting a configuration from the
`<clusterName>-pgha-config` ConfigMap.  Once a configuration has been deleted,
the Operator will detect this and refresh the ConfigMap with a fresh copy of
that specific configuration.

For instance, the following `kubectl patch` command can be utilized to
remove the `mycluster-dcs-config` configuration from the example above,
causing that specific configuration to be refreshed with a fresh copy of
the DCS configuration settings for `mycluster`:

```bash
kubectl patch configmap mycluster-pgha-config \
  --type='json' -p='[{"op": "remove", "path": "/data/mycluster-dcs-config"}]'
```


## Custom pgBackRest Configuration

Users can configure pgBackRest by passing the name of an existing ConfigMap to
the `--pgbackrest-custom-config` flag when creating a PostgreSQL cluster. The
entire contents of that ConfigMap appear as files in pgBackRest's
[`config-include-path` directory](https://pgbackrest.org/user-guide.html).

Regardless of the flags passed at creation, every PostgreSQL cluster is
automatically configured to read from a ConfigMap named
`<clusterName>-config-backrest` and a Secret named
`<clusterName>-config-backrest`. These objects can be populated either before
or _after_ a PostgreSQL cluster is created. The entire contents of each appear
as files in pgBackRest's `config-include-path` directory.

Though the above is very flexible, not all pgBackRest settings can be managed
this way. There are a few that are always overridden by the PostgreSQL Operator
(the path to the PostgreSQL data directory, for example).
