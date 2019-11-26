---
title: "Failover in the PostgreSQL Operator Overview"
date:
draft: false
weight: 6
---

## Failover in the PostgreSQL Operator

There are a number of potential events that could cause a primary database instance or cluster to become unavailable during the course of normal operations, including: 

* A database storage (disk) failure or any other hardware failure 
* The network on which the database resides becomes unreachable
* The host operating system becomes unstable and crashes
* A key database file becomes corrupted
* Total loss of data center

There may also be downtime events that are due to the normal case of operations, such as performing a minor upgrade, security patching of operating system, hardware upgrade, or other maintenance.

To enable rapid recovery from the unavailability of the primary PostgreSQL instance within a PostgreSQL cluster, the PostgreSQL Operator supports both Manual and Automated failover within a single Kubernetes cluster. 

### PostgreSQL Cluster Architecture 

The failover from a primary PostgreSQL instances to a replica PostgreSQL instance within a PostgreSQL cluster. 

### Manual Failover

Manual failover is performed by PostgreSQL Operator API actions involving a *query* and then a *target* being specified to pick the fail-over replica target.

### Automatic Failover 

Automatic failover is managed and performed by Patroni, which is running within each primary and replica database pod within the cluster to ensure the PG database remains highly-available.  By monitoring the cluster, Patroni is able to detect failures in the primary database, and then automatically failover to (i.e. "promote") a healthy replica.  Automatic failover capabilities are enabled by default for any newly created clusters, but can also be disabled for a newly created cluster by setting `DisableFailover` to true in the `pgo.yaml` configuration, or by setting the `--disable-failover` flag via the PGO CLI when creating the cluster.  If disabled, failover capabiltiies can then be enabled (as well as disabled once again) at any time by utilizing the `pgo update cluster` command. 

When a failover does occur, the system automatically attempts to turn the old primary into a replica (using `pg_rewind` if needed), ensuring the cluster maintains the same amount of database pods and replicas following a failure.  Additionally, the `role` label on each pod is updated as needed to properly identify the `master` pod and any `replica` pods following a failover event, therefore ensuring the primary and replica services point to the proper database pods.  And finally, the `pgBackRest` dedicated repository host is also automatically reconfigured to point to the `PGDATA` directory of the new `primary` pod following a failover.
