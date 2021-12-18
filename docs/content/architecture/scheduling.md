---
title: "Scheduling"
date:
draft: false
weight: 120
---

Deploying to your Kubernetes cluster may allow for greater reliability than other
environments, but that's only the case when it's configured correctly. Fortunately,
PGO, the Postgres Operator from Crunchy Data, is ready to help with helpful
default settings to ensure you make the most out of your Kubernetes environment!

## High Availability By Default

As shown in the [high availability tutorial]({{< relref "tutorial/high-availability.md" >}}#pod-topology-spread-constraints),
PGO supports the use of [Pod Topology Spread Constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/)
to customize your Pod deployment strategy, but useful defaults are already in place
for you without any additional configuration required!

PGO's default scheduling constraints for HA is implemented for the various Pods
 comprising a PostgreSQL cluster, specifically to ensure the Operator always
 deploys a High-Availability cluster architecture by default.

 Using Pod Topology Spread Constraints, the general scheduling guidelines are as
 follows:

- Pods are only considered from the same cluster.
- PgBouncer pods are only considered amongst other PgBouncer pods.
- Postgres pods are considered amongst all Postgres pods and pgBackRest repo host Pods.
- pgBackRest repo host Pods are considered amongst all Postgres pods and pgBackRest repo hosts Pods.
- Pods are scheduled across the different `kubernetes.io/hostname` and `topology.kubernetes.io/zone` failure domains.
- Pods are scheduled when there are fewer nodes than pods, e.g. single node.

With the above configuration, your data is distributed as widely as possible
throughout your Kubernetes cluster to maximize safety.

## Customization

While the default scheduling settings are designed to meet the widest variety of
environments, they can be customized or removed as needed. Assuming a PostgresCluster
named 'hippo', the default Pod Topology Spread Constraints applied on Postgres
Instance and pgBackRest Repo Host Pods are as follows:

```
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        postgres-operator.crunchydata.com/cluster: hippo
      matchExpressions:
      - key: postgres-operator.crunchydata.com/data
        operator: In
        values:
        - postgres
        - pgbackrest
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        postgres-operator.crunchydata.com/cluster: hippo
      matchExpressions:
      - key: postgres-operator.crunchydata.com/data
        operator: In
        values:
        - postgres
        - pgbackrest
```

Similarly, for PgBouncer Pods they will be:

```
topologySpreadConstraints:
  - maxSkew: 1
    topologyKey: kubernetes.io/hostname
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        postgres-operator.crunchydata.com/cluster: hippo
        postgres-operator.crunchydata.com/role: pgbouncer
  - maxSkew: 1
    topologyKey: topology.kubernetes.io/zone
    whenUnsatisfiable: ScheduleAnyway
    labelSelector:
      matchLabels:
        postgres-operator.crunchydata.com/cluster: hippo
        postgres-operator.crunchydata.com/role: pgbouncer
```

Which, as described in the [API documentation](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/#spread-constraints-for-pods),
means that there should be a maximum of one Pod difference within the
`kubernetes.io/hostname` and `topology.kubernetes.io/zone` failure domains when
considering either `data` Pods, i.e. Postgres Instance or pgBackRest repo host Pods
from a single PostgresCluster or when considering PgBouncer Pods from a single
PostgresCluster.

Any other scheduling configuration settings, such as [Affinity, Anti-affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#affinity-and-anti-affinity),
[Taints, Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/),
or other [Pod Topology Spread Constraints](https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/)
will be added in addition to these defaults. Care should be taken to ensure the
combined effect of these settings are appropriate for your Kubernetes cluster.

In cases where these defaults are not desired, PGO does provide a method to disable
the default Pod scheduling by setting the `spec.disableDefaultPodScheduling` to
'true'.
