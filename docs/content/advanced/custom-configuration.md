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
tuning paramters for the High Availability features inlcuded in each cluster.
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
    unix_socket_directories: /tmp,/crunchyadm
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
  - local all crunchyadm peer
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

The various key/value pairs provided within the `paramters` section result in the
configuration of the same settings within the `postgresql.conf` file.  Please note that
settings applied locally to a database server take precendence over those set via the DCS (with the
exception being those that must be set via the DCS, as
[described here](https://access.crunchydata.com/documentation/patroni/latest/dynamic_configuration/)).

Also, please note that `pg_hba` and `pg_ident` sections exist to update both the `pg_hba.conf` and
`pg_ident.conf` PostgreSQL configuration files as needed.

#### Restarting Database Servers

Changes to certain settings may require a restart of a PostgreSQL database.
This can be accomplished using the `patronictl` utility included wihtin
each PostgreSQL database container in the cluster, specifically
using the `patronictl restart` command.  For example, to detect if a
restart is needed for a server in a cluster called `mycluster`, the
`kubectl exec` command can be utilized to access the database container for
the primary PostgreSQL database server, and run the `patronictl list`
command:

```bash
$ kubectl exec -it mycluster-6f89d8bb85-pnlwz -- patronictl list
+ Cluster: mycluster (6821144425371877525) -------+---------+----+-----------+-----------------+
|            Member           |    Host   |  Role  |  State  | TL | Lag in MB | Pending restart |
+-----------------------------+-----------+--------+---------+----+-----------+-----------------+
| mycluster-6f89d8bb85-pnlwz | 10.44.0.6 | Leader | running |  1 |           |        *        |
+-----------------------------+-----------+--------+---------+----+-----------+-----------------+
```

Here we can see that the ` mycluster-6f89d8bb85-pnlwz` server is pending a restart,
which can then be accomplished as follows:

```bash
$ kubectl exec -it mycluster-6f89d8bb85-pnlwz -- patronictl restart mycluster mycluster-6f89d8bb85-pnlwz
+ Cluster: mycluster (6821144425371877525) -------+---------+----+-----------+
|            Member           |    Host   |  Role  |  State  | TL | Lag in MB |
+-----------------------------+-----------+--------+---------+----+-----------+
| mycluster-6f89d8bb85-pnlwz | 10.44.0.6 | Leader | running |  1 |           |
+-----------------------------+-----------+--------+---------+----+-----------+
When should the restart take place (e.g. 2020-04-29T17:23)  [now]: now
Are you sure you want to restart members mycluster-6f89d8bb85-pnlwz? [y/N]: y
Restart if the PostgreSQL version is less than provided (e.g. 9.5.2)  []:
Success: restart on member mycluster-6f89d8bb85-pnlwz
```

Please note that these commands can be run from the primary or any replica
database container within the PostgreSQL cluster being updated.

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
