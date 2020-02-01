---
title: "Tablespaces"
date:
draft: false
weight: 850
---

A [Tablespace](https://www.postgresql.org/docs/current/manage-ag-tablespaces.html)
is a PostgreSQL feature that is used to store data on a volume that is different
from the primary data directory. While most workloads do not require them,
tablespaces can be particularly helpful for larger data sets or utilizing
particular hardware to optimize performance on a particular PostgreSQL object
(a table, index, etc.). Some examples of use cases for tablespaces include:

- Partitioning larger data sets across different volumes
- Putting data onto archival systems
- Utilizing hardware (or a storage class) for a particular database
- Storing sensitive data on a volume that supports transparent data-encryption
(TDE)

and others.

In order to use PostgreSQL tablespaces properly in a highly-available,
distributed system, there are several considerations that need to be accounted
for to ensure proper operations:

- Each tablespace must have its own volume; this means that every tablespace for
every replica in a system must have its own volume.
- The filesystem map must be consistent across the cluster
- The backup & disaster recovery management system must be able to safely backup
and restore data to tablespaces

Additionally, a tablespace is a critical piece of a PostgreSQL instance: if
PostgreSQL expects a tablespace to exist and it is unavailable, this could
trigger a downtime scenario.

While there are certain challenges with creating a PostgreSQL cluster with
high-availability along with tablespaces in a Kubernetes-based environment, the
PostgreSQL Operator adds many conveniences to make it easier to use
tablespaces in applications.

## How Tablespaces Work in the PostgreSQL Operator

As stated above, it is important to ensure that every tablespace created has its
own volume (i.e. its own [persistent volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)).
This is especially true for any replicas in a cluster: you don't want multiple
PostgreSQL instances writing to the same volume, as this is a recipe for
disaster!

One of the keys to working with tablespaces in a high-availability cluster is to
ensure the filesystem that the tablespaces map to is consistent. Specifically,
it is imperative to have the `LOCATION` parameter that is used by PostgreSQL to
indicate where a tablespace resides to match in each instance in a cluster.

The PostgreSQL Operator achieves this by mounting all of its tablespaces to a
directory called `/tablespaces` in the container. While each tablespace will
exist in a unique PVC across all PostgreSQL instances in a cluster, each
instance's tablespaces will mount in a predictable way in `/tablespaces`.

The PostgreSQL Operator takes this one step further and abstracts this away from
you. When your PostgreSQL cluster initialized, the tablespace definition is
automatically created in PostgreSQL; you can start using it immediately! An
example of this is demonstrated in the next section.

The PostgreSQL Operator ensures the availability of the tablespaces across the
different lifecycle events that occur on a PostgreSQL cluster, including:

- High-Availability: Data in the tablespaces is replicated across the cluster,
and is available after a downtime event
- Disaster Recovery: Tablespaces are backed up and are properly restored during
a recovery
- Clone: Tablespaces are created in any cloned cluster
- Deprovisioining: Tablespaces are deleted when a PostgreSQL instance or cluster
is deleted

## Creating Tablespaces

Tablespaces can be used in a cluster with the [`pgo create cluster`](/pgo-client/reference/pgo_create_cluster/)
command. The command follows this general format:

```shell
pgo create cluster hacluster --tablespaces=tablespace1=storageclass,tablespace2=storageclass
```

For example, to create tablespaces name `faststorage1` and `faststorage2` on
PVCs that use the `nfsstorage` storage type, you would execute the following
command:

```shell
pgo create cluster hacluster --tablespaces=faststorage1=nfsstorage,faststorage2=nfsstorage
```

Once the cluster is initialized, you can immediately interface with the
tablespaces! For example, if you wanted to create a table called `sensor_data`
on the `faststorage1` tablespace, you could execute the following SQL:

```sql
CREATE TABLE sensor_data (
  sensor_id int,
  sensor_value numeric,
  created_at timestamptz DEFAULT CURRENT_TIMESTAMP
)
TABLESPACE faststorage1;
```

For more information on how tablespaces work in PostgreSQL please refer to the
[PostgreSQL manual](https://www.postgresql.org/docs/current/manage-ag-tablespaces.html).
