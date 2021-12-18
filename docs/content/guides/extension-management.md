---
title: "Extension Management"
date:
draft: false
weight: 175
---

[Extensions](https://www.postgresql.org/docs/current/external-extensions.html) combine functions, data types, casts, etc. -- everything you need
to add some new feature to PostgreSQL in an easy to install package. How easy to install?
For many extensions, like the `fuzzystrmatch` extension, it's as easy as connecting to the database and running a command like this:

```
CREATE EXTENSION fuzzystrmatch;
```

However, in other cases, an extension might require additional configuration management.
PGO lets you add those configurations to the `PostgresCluster` spec easily.


PGO also allows you to add a custom databse initialization script in case you would like to
automate how and where the extension is installed.


This guide will walk through adding custom configuration for an extension and
automating installation, using the example of Crunchy Data's own `pgnodemx` extension.

- [pgnodemx](#pgnodemx)

## `pgnodemx`

[`pgnodemx`](https://github.com/CrunchyData/pgnodemx) is a PostgreSQL extension
that is able to pull container-specific metrics (e.g. CPU utilization, memory
consumption) from the container itself via SQL queries.

In order to do this, `pgnodemx` requires information from the Kubernetes [DownwardAPI](https://kubernetes.io/docs/tasks/inject-data-application/downward-api-volume-expose-pod-information/)
to be mounted on the PostgreSQL pods. Please see the `pgnodemx and the DownwardAPI`
section of the [backup architecture]({{< relref "architecture/backups.md" >}}) page for more information on
where and how the DownwardAPI is mounted.

### `pgnodemx` Configuration

To enable the `pdnodemx` extension, we need to set certain configurations. Luckily,
this can all be done directly through the spec:

```yaml
spec:
  patroni:
    dynamicConfiguration:
      postgresql:
        parameters:
          shared_preload_libraries: pgnodemx
          pgnodemx.kdapi_enabled: on
          pgnodemx.kdapi_path: /etc/database-containerinfo
```

Those three settings will

* load `pgnodemx` at start;
* enable the `kdapi` functions (which are specific to the capture of Kubernetes DownwardAPI information);
* tell `pgnodemx` where those DownwardAPI files are mounted (at the `/etc/dabatase-containerinfo` path).

If you create a `PostgresCluster` with those configurations, you will be able to connect,
create the extension in a database, and run the functions installed by that extension:

```
CREATE EXTENSION pgnodemx;
SELECT * FROM proc_diskstats();
```

### Automating `pgnodemx` Creation

Now that you know how to configure `pgnodemx`, let's say you want to automate the creation of
the extension in a particular database, or in all databases. We can do that through
a custom database initialization.

First, we have to create a ConfigMap with the initialization SQL. Let's start with the
case where we want `pgnodemx` created for us in the `hippo` database. Our initialization
SQL file might be named `init.sql` and look like this:

```
\c hippo\\
CREATE EXTENSION pgnodemx;
```

Now we create the ConfigMap from that file in the same namespace as our PostgresCluster will be created:

```shell
kubectl create configmap hippo-init-sql -n postgres-operator --from-file=init.sql=path/to/init.sql
```

You can check that the ConfigMap was created and has the right information:

```shell
kubectl get configmap -n postgres-operator hippo-init-sql -o yaml

apiVersion: v1
data:
  init.sql: |-
    \c hippo\\
    CREATE EXTENSION pgnodemx;
kind: ConfigMap
metadata:
  name: hippo-init-sql
  namespace: postgres-operator
```

Now, in addition to the spec changes we made above to allow `pgnodemx` to run,
we add that ConfigMap's information to the PostgresCluster spec: the name of the
ConfigMap (`hippo-init-sql`) and the key for the data (`init.sql`):

```yaml
spec:
  databaseInitSQL:
    key: init.sql
    name: hippo-init-sql
```

Apply that spec to a new or existing PostgresCluster, and the pods should spin up with
`pgnodemx` already installed in the `hippo` database.

