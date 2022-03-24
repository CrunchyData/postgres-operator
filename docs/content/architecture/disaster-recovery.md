---
title: "Disaster Recovery"
date:
draft: false
weight: 140
---

![PostgreSQL Operator High-Availability Overview](/images/postgresql-ha-multi-data-center.png)

Advanced [high-availability]({{< relref "architecture/high-availability.md" >}})
and [backup management]({{< relref "architecture/backups.md" >}})
strategies involve spreading your database clusters across multiple data centers
to help maximize uptime. In Kubernetes, this technique is known as "[federation](https://en.wikipedia.org/wiki/Federation_(information_technology))".
Federated Kubernetes clusters are able to communicate with each other,
coordinate changes, and provide resiliency for applications that have high
uptime requirements.

As of this writing, federation in Kubernetes is still in ongoing development
and is something we monitor with intense interest. As Kubernetes federation
continues to mature, we wanted to provide a way to deploy PostgreSQL clusters
managed by the [PostgreSQL Operator](https://www.crunchydata.com/developers/download-postgres/containers/postgres-operator)
that can span multiple Kubernetes clusters. This can be accomplished with a
few environmental setups:

- Two Kubernetes clusters
- An external storage system, using one of the following:
  - S3, or an external storage system that uses the S3 protocol
  - GCS
  - Azure Blob Storage
  - A Kubernetes storage system that can span multiple clusters

At a high-level, the PostgreSQL Operator follows the "active-standby" data
center deployment model for managing the PostgreSQL clusters across Kubernetes
clusters. In one Kubernetes cluster, the PostgreSQL Operator deploy PostgreSQL as an
"active" PostgreSQL cluster, which means it has one primary and one-or-more
replicas. In another Kubernetes cluster, the PostgreSQL cluster is deployed as
a "standby" cluster: every PostgreSQL instance is a replica.

A side-effect of this is that in each of the Kubernetes clusters, the PostgreSQL
Operator can be used to deploy both active and standby PostgreSQL clusters,
allowing you to mix and match! While the mixing and matching may not ideal for
how you deploy your PostgreSQL clusters, it does allow you to perform online
moves of your PostgreSQL data to different Kubernetes clusters as well as manual
online upgrades.

Lastly, while this feature does extend high-availability, promoting a standby
cluster to an active cluster is **not** automatic. While the PostgreSQL clusters
within a Kubernetes cluster do support self-managed high-availability, a
cross-cluster deployment requires someone to specifically promote the cluster
from standby to active.

## Standby Cluster Overview

Standby PostgreSQL clusters are managed just like any other PostgreSQL cluster
that is managed by the PostgreSQL Operator. For example, adding replicas to a
standby cluster is identical as adding them to a primary cluster.

As the architecture diagram above shows, the main difference is that there is
no primary instance: one PostgreSQL instance is reading in the database changes
from the backup repository, while the other replicas are replicas of that instance.
This is known as [cascading replication](https://www.postgresql.org/docs/current/warm-standby.html#CASCADING-REPLICATION).
 replicas are cascading replicas, i.e. replicas replicating from a database server that itself is replicating from another database server.

Because standby clusters are effectively read-only, certain functionality
that involves making changes to a database, e.g. PostgreSQL user changes, is
blocked while a cluster is in standby mode.  Additionally, backups and restores
are blocked as well. While [pgBackRest](https://pgbackrest.org/) does support
backups from standbys, this requires direct access to the primary database,
which cannot be done until the PostgreSQL Operator supports Kubernetes
federation.

## Creating a Standby PostgreSQL Cluster

For creating a standby Postgres cluster with PGO, please see the [disaster recovery tutorial]({{< relref "tutorial/disaster-recovery.md" >}}#standby-cluster)

## Promoting a Standby Cluster

There comes a time where a standby cluster needs to be promoted to an active
cluster. Promoting a standby cluster means that a PostgreSQL instance within
it will become a primary and start accepting both reads and writes. This has the
net effect of pushing WAL (transaction archives) to the pgBackRest repository,
so we need to take a few steps first to ensure we don't accidentally create a
split-brain scenario.

First, if this is not a disaster scenario, you will want to "shutdown" the
active PostgreSQL cluster. This can be done by setting:

```
spec:
  shutdown: true
```

The effect of this is that all the Kubernetes Statefulsets and Deployments for this cluster are
scaled to 0.

We can then promote the standby cluster using the following:

```
spec:
  standby:
    enabled: false
```

This command essentially removes the standby configuration from the Kubernetes
cluster’s DCS, which triggers the promotion of the current standby leader to a
primary PostgreSQL instance. You can view this promotion in the PostgreSQL
standby leader's (soon to be active leader's) logs:

With the standby cluster now promoted, the cluster with the original active
PostgreSQL cluster can now be turned into a standby PostgreSQL cluster.  This is
done by deleting and recreating all PVCs for the cluster and re-initializing it
as a standby using the backup repository.  Being that this is a destructive action
(i.e. data will only be retained if any Storage Classes and/or Persistent
Volumes have the appropriate reclaim policy configured) a warning is shown
when attempting to enable standby.

The cluster will reinitialize from scratch as a standby, just
like the original standby that was created above.  Therefore any transactions
written to the original standby, should now replicate back to this cluster.
