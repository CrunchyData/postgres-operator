Crunchy Data PostgreSQL Operator
=======

[PostgreSQL](https://postgresql.org) is a powerful, open source object-relational database system. It has more than 15 years of active development and a proven architecture that has earned it a strong reputation for reliability, data integrity, and correctness.


TL;DR;
------

```console
$ helm install --name postgres-operator --namespace=postgres-operator postgres-operator
```

Introduction
------------

This chart bootstraps a PostgreSQL Operator.

The PostgreSQL Operator provides a Kubernetes operator capability for managing PostgreSQL Clusters deployed within a Kubernetes.
The PostgreSQL Operator leverages Kubernetes Custom Resources to define resource types such as:

 * *pgcluster*
 * *pgbackups*
 * *pgreplicas*
 * *pgupgrades*
 * *pgpolicies*
 * *pgpolicylogs*


Once those custom objects are defined, Kubernetes provides the ability to create and manage those objects similar to any other native Kubernetes object.

The PostgreSQL Operator runs within Kubernetes detecting these new custom object types being created, updated, or removed.

Once the objects are detected, the PostgreSQL Operator enables users to perform operations across the Kubernetes environment, including:

* Create a PostgreSQL Cluster
* Destroy a PostgreSQL Cluster
* Backup a PostgreSQL Cluster
* Scale a a PostgreSQL Cluster
* Restore a PostgreSQL Cluster
* Upgrade a PostgreSQL Cluster
* View PVC
* Test Connections to a PostgreSQL Cluster
* Create a SQL-based Policy
* Apply a SQL-based Policy to a PostgreSQL Cluster
* Perform User Management
* Apply User Defined Labels to PostgreSQL Clusters
* Perform Password Management

What actually gets created on the Kube cluster for a
*pgcluster* resource is defined as a *deployment strategy*.  Strategies
are documented in detail in [Deployment Strategies](https://github.com/CrunchyData/postgres-operator/blob/master/docs/design.asciidoc#postgresql-operator-deployment-strategies).


Installing the Chart
--------------------

The chart can be installed as follows:

```console
$ helm install postgres-operator --name postgres-operator --namespace=postgres-operator
```

The command deploys postgres-operator on the Kubernetes cluster in the default configuration.

> **Tip**: List all releases using `helm list`

Uninstalling the Chart
----------------------

To uninstall/delete the `postgres-operator` deployment:

```console
$ helm delete postgres-operator
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

Configuration
-------------

See `values.yaml` for configuration notes. Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example,

```console
$ helm install postgres-operator --name postgres-operator --namespace=postgres-operator \
  --set env.debug=false
```

The above command disables the debugging.

Alternatively, a YAML file that specifies the values for the above parameters can be provided while installing the chart. For example,

```console
$ helm install postgres-operator --name postgres-operator --namespace=postgres-operator  \
  -f values.yaml
```

> **Tip**: You can use the default [values.yaml](values.yaml)
