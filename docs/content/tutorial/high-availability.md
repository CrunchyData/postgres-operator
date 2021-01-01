---
title: "High Availability"
draft: false
weight: 180
---

One of the great things about PostgreSQL is its reliability: it is very stable and typically "just works." However, there are certain things that can happen in the environment that PostgreSQL is deployed in that can affect its uptime, including:

- The database storage disk fails or some other hardware failure occurs
- The network on which the database resides becomes unreachable
- The host operating system becomes unstable and crashes
- A key database file becomes corrupted
- A data center is lost

There may also be downtime events that are due to the normal case of operations, such as performing a minor upgrade, security patching of operating system, hardware upgrade, or other maintenance.

Fortunately, the Crunchy PostgreSQL Operator is prepared for this.

![PostgreSQL Operator High-Availability Overview](/images/postgresql-ha-overview.png)

The Crunchy PostgreSQL Operator supports a distributed-consensus based high-availability (HA) system that keeps its managed PostgreSQL clusters up and running, even if the PostgreSQL Operator disappears. Additionally, it leverages Kubernetes specific features such as [Pod Anti-Affinity](#how-the-crunchy-postgresql-operator-uses-pod-anti-affinity) to limit the surface area that could lead to a PostgreSQL cluster becoming unavailable. The PostgreSQL Operator also supports automatic healing of failed primaries and leverages the efficient pgBackRest "delta restore" method, which eliminates the need to fully reprovision a failed cluster!

This tutorial will cover the "howtos" of high availbility. For more information on the topic, please review the detailed [high availability architecture]({{< relref "architecture/high-availability/_index.md" >}}) section.

## Create a HA PostgreSQL Cluster

High availability is enabled in the PostgreSQL Operator by default so long as you have more than one replica. To create a high availability PostgreSQL cluster, you can execute the following command:

```
pgo create cluster hippo --replica-count=1
```

## Scale a PostgreSQL Cluster

You can scale an existing PostgreSQL cluster to add HA to it by using the [`pgo scale`]({{< relref "pgo-client/reference/pgo_scale.md">}}) command:

```
pgo scale hippo
```

## Scale Down a PostgreSQL Cluster

To scale down a PostgreSQL cluster, you will have to provide a target of which instance you want to scale down. You can do this with the [`pgo scaledown`]({{< relref "pgo-client/reference/pgo_scaledown.md">}}) command:

```
pgo scaledown hippo --query
```

which will yield something similar to:

```
Cluster: hippo
REPLICA             	STATUS    	NODE      	REPLICATION LAG     	PENDING RESTART
hippo-ojnd          	running   	node01    	           0 MB     	          false
```

Once you have determined which instance you want to scale down, you can run the following command:

```
pgo scaledown hippo --target=hippo-ojnd
```

## Manual Failover

Each PostgreSQL cluster will manage its own availability. If you wish to manually fail over, you will need to use the [`pgo failover`]({{< relref "pgo-client/reference/pgo_failover.md">}}) command.

There are two ways to issue a manual failover to your PostgreSQL cluster:

1. Allow for the PostgreSQL Operator to select the best replica candidate for failover.
2. Select your own replica candidate for failover.

Both methods are detailed below.

### Manual Failover - PostgreSQL Operator Candidate Selection

To have the PostgreSQL Operator select the best replica candidate for failover, all you need to do is execute the following command:

```
pgo failover hippo
```

The PostgreSQL Operator will determine which is the best replica candidate to fail over to, and take into account factors such as replication lag and current timeline.

### Manual Failover - Manual Selection

If you wish to have your cluster manually failover, you must first query your determine which instance you want to fail over to. You can do so with the following command:

```
pgo failover hippo --query
```

which will yield something similar to:

```
Cluster: hippo
REPLICA             	STATUS    	NODE      	REPLICATION LAG     	PENDING RESTART
hippo-ojnd          	running   	node01    	           0 MB     	          false
```

Once you have determine your failover target, you can run the following command:

```
pgo failover hippo --target==hippo-ojnd
```

## Synchronous Replication

If you have a [write sensitive workload and wish to use synchronous replication]({{< relref "architecture/high-availability/_index.md" >}}#synchronous-replication-guarding-against-transactions-loss), you can create your PostgreSQL cluster with synchronous replication turned on:

```
pgo create cluster hippo --sync-replication
```

Please understand the tradeoffs of synchronous replication before using it.

## Pod Anti-Affinity and Node Affinity

To learn how to use pod anti-affinity and node affinity, please refer to the [high availability architecture documentation]({{< relref "architecture/high-availability/_index.md" >}}).

## Tolerations

If you want to have a PostgreSQL instance use specific Kubernetes [tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/), you can use the `--toleration` flag on [`pgo scale`]({{< relref "pgo-client/reference/pgo_scale.md">}}). Any tolerations added to the new PostgreSQL instance fully replace any tolerations available to the entire cluster.

For example, to assign equality toleration for a key/value pair of `zone`/`west`, you can run the following command:

```
pgo scale hippo --toleration=zone=west:NoSchedule
```

For more information on the PostgreSQL Operator and tolerations, please review the [high availability architecture documentation]({{< relref "architecture/high-availability/_index.md" >}}).

## Troubleshooting

### No Primary Available After Both Synchronous Replication Instances Fail

Though synchronous replication is available for guarding against transaction loss for [write sensitive workloads]({{< relref "architecture/high-availability/_index.md" >}}#synchronous-replication-guarding-against-transactions-loss), by default the high availability systems prefers availability over consistency and will continue to accept writes to a primary even if a replica fails. Additionally, in most scenarios, a system using synchronous replication will be able to recover and self heal should a primary or a replica go down.

However, in the case that both a primary and its synchronous replica go down at the same time, a new primary may not be promoted. To guard against transaction loss, the high availability system will not promote any instances if it cannot determine if they had been one of the synchronous instances. As such, when it recovers, it will bring up all the instances as replicas.

To get out of this situation, inspect the replicas using `pgo failover --query` to determine the best candidate (typically the one with the least amount of replication lag). After determining the best candidate, promote one of the replicas using `pgo failover --target` command.

If you are still having issues, you may need to execute into one of the Pods and inspect the state with the `patronictl` command.

A detailed breakdown of this case be found [here](https://github.com/CrunchyData/postgres-operator/issues/2132#issuecomment-748719843).

## Next Steps

Backups, restores, point-in-time-recoveries: [disaster recovery]({{< relref "architecture/disaster-recovery.md" >}}) is a big topic! We'll learn about you can [perform disaster recovery]({{< relref "tutorial/disaster-recovery.md" >}}) and more in the PostgreSQL Operator.
