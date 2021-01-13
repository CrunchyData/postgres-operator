---
title: "Customize a Postgres Cluster"
draft: false
weight: 130
---

The PostgreSQL Operator makes it very easy and quick to [create a cluster]({{< relref "tutorial/create-cluster.md" >}}), but there are possibly more customizations you want to make to your cluster. These include:

- Resource allocations (e.g. Memory, CPU, PVC size)
- Sidecars (e.g. [Monitoring]({{< relref "architecture/monitoring.md" >}}), [pgBouncer]({{< relref "tutorial/pgbouncer.md" >}}), [pgAdmin 4]({{< relref "architecture/pgadmin4.md" >}}))
- High Availability (e.g. adding replicas)
- Specifying specific PostgreSQL images (e.g. one with PostGIS)
- Specifying a [Pod anti-affinity and Node affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/)
- Enable and/or require TLS for all connections
- [Custom PostgreSQL configurations]({{< relref "advanced/custom-configuration.md" >}})

and more.

There are an abundance of ways to customize your PostgreSQL clusters with the PostgreSQL Operator. You can read about all of these options in the [`pgo create cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) reference.

The goal of this section is to present a few of the common actions that can be taken to help create the PostgreSQL cluster of your choice. Later sections of the tutorial will cover other topics, such as creating a cluster with TLS or tablespaces.

## Create a PostgreSQL Cluster With Monitoring

The [PostgreSQL Operator Monitoring]({{< relref "architecture/monitoring.md" >}}) stack provides a convenient way to gain insights into the availabilty and performance of your PostgreSQL clusters. In order to collect metrics from your PostgreSQL clusters, you have to enable the `crunchy-postgres-exporter` sidecar alongside your PostgreSQL cluster. You can do this with the `--metrics` flag on [`pgo create cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}):

```
pgo create cluster hippo --metrics
```

Note that the `--metrics` flag just enables a sidecar that can be scraped. You will need to install the [monitoring stack]({{< relref "installation/metrics/_index.md" >}}) separately, or tie it into your existing monitoring infrastructure.

If you have an exiting cluster that you would like to add metrics collection to, you can use the `--enable-metrics` flag on the [`pgo update cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) command:

```
pgo update cluster hippo --enable-metrics
```

## Customize PVC Size

Databases come in all different sizes, and those sizes can certainly change over time. As such, it is helpful to be able to specify what size PVC you want to store your PostgreSQL data.

### Customize PVC Size for PostgreSQL

The PostgreSQL Operator lets you choose the size of your "PostgreSQL data directory" (aka "PGDATA" directory) using the `--pvc-size` flag. The PVC size should be selected using standard [Kubernetes resource units](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-units-in-kubernetes), e.g. `20Gi`.

For example, to create a PostgreSQL cluster that has a data directory that is `20Gi` in size:

```
pgo create cluster hippo --pvc-size=20Gi
```

### Customize PVC Size for pgBackRest

You can also specify the PVC size for the [pgBackRest repository]({{< relref "architecture/disaster-recovery.md" >}}) with the `--pgbackrest-pvc-size`. [pgBackRest](https://pgbackrest.org/) is used to store all of your backups, so you want to size it so that you can meet your backup retention policy.

For example, to create a pgBackRest repository that has a PVC sized to `100Gi` in size:

```
pgo create cluster hippo --pgbackrest-pvc-size=100Gi
```

## Customize CPU / Memory

Databases have different CPU and memory requirements, often which is dictated by the amount of data in your working set (i.e. actively accessed data). Kubernetes provides several ways for Pods to [manage CPU and memory resources](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/):

- CPU & Memory Requests
- CPU & Memory Limits

A CPU or Memory Request tells Kubernetes to ensure that there is _at least_ that amount of resource available on the Node to schedule a Pod to.

A CPU Limit tells Kubernetes to not let a Pod exceed utilizing that amount of CPU. A Pod will only be allowed to use that maximum amount of CPU. Similarly, a Memory limit tells Kubernetes to not let a Pod exceed a certain amount of Memory. In this case, if Kubernetes detects that a Pod has exceed a Memory limit, it will try to terminate any processes that are causing the limit to be exceed. We mention this as, prior to cgroups v2, Memory limits can potentially affect PostgreSQL availability and we advise to use them carefully.

The below goes into how you can customize the CPU and memory resources that are made available to the core deployment Pods with your PostgreSQL cluster. Customizing CPU and memory does add more resources to your PostgreSQL cluster, but to fully take advantage of additional resources, you will need to [customize your PostgreSQL configuration](#customize-postgresql-configuration) and tune parameters such as `shared_buffers` and others.

### Customize CPU / Memory for PostgreSQL

The PostgreSQL Operator provides several flags for [`pgo create cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) to help manage resources for a PostgreSQL instance:

- `--cpu`: Specify the CPU Request for a PostgreSQL instance
- `--cpu-limit`: Specify the CPU Limit for a PostgreSQL instance
- `--memory`: Specify the Memory Request for a PostgreSQL instance
- `--memory-limit`: Specify the Memory Limit for a PostgreSQL instance

For example, to create a PostgreSQL cluster that makes a CPU Request of 2.0 with a CPU Limit of 4.0 and a Memory Request of 4Gi with a Memory Limit of 6Gi:

```
pgo create cluster hippo \
  --cpu=2.0 --cpu-limit=4.0 \
  --memory=4Gi --memory-limit=6Gi
```

### Customize CPU / Memory for Crunchy PostgreSQL Exporter Sidecar

If you deploy your [PostgreSQL cluster with monitoring](#create-a-postgresql-cluster-with-monitoring), you may want to adjust the resources of the `crunchy-postgres-exporter` sidecar that runs next to each PostgreSQL instnace. You can do this with the following flags:

- `--exporter-cpu`: Specify the CPU Request for a `crunchy-postgres-exporter` sidecar
- `--exporter-cpu-limit`: Specify the CPU Limit for a `crunchy-postgres-exporter` sidecar
- `--exporter-memory`: Specify the Memory Request for a `crunchy-postgres-exporter` sidecar
- `--exporter-memory-limit`: Specify the Memory Limit for a `crunchy-postgres-exporter` sidecar

For example, to create a PostgreSQL cluster with a metrics sidecar with custom CPU and memory requests + limits, you could do the following:

```
pgo create cluster hippo --metrics \
  --exporter-cpu=0.5 --exporter-cpu-limit=1.0 \
  --exporter-memory=256Mi --exporter-memory-limit=1Gi
```

### Customize CPU / Memory for pgBackRest

You can also customize the CPU and memory requests and limits for pgBackRest with the following flags:

- `--pgbackrest-cpu`: Specify the CPU Request for pgBackRest
- `--pgbackrest-cpu-limit`: Specify the CPU Limit for pgBackRest
- `--pgbackrest-memory`: Specify the Memory Request for pgBackRest
- `--pgbackrest-memory-limit`: Specify the Memory Limit for pgBackRest

For example, to create a PostgreSQL cluster with custom CPU and memory requests + limits for pgBackRest, you could do the following:

```
pgo create cluster hippo \
  --pgbackrest-cpu=0.5 --pgbackrest-cpu-limit=1.0 \
  --pgbackrest-memory=256Mi --pgbackrest-memory-limit=1Gi
```

## Create a High Availability PostgreSQL Cluster

[High availability]({{< relref "architecture/high-availability/_index.md" >}}) allows you to deploy PostgreSQL clusters with redundancy that allows them to be accessible by your applications even if there is a downtime event to your primary instance. The PostgreSQL clusters use the distributed consensus storage system that comes with Kubernetes so that availability is tied to that of your Kubenretes clusters. For an in-depth discussion of the topic, please read the [high availability]({{< relref "architecture/high-availability/_index.md" >}}) section of the documentation.

To create a high availability PostgreSQL cluster with one replica, you can run the following command:

```
pgo create cluster hippo --replica-count=1
```

You can scale up and down your PostgreSQL cluster with the [`pgo scale`]({{< relref "pgo-client/reference/pgo_scale.md" >}}) and [`pgo scaledown`]({{< relref "pgo-client/reference/pgo_scaledown.md" >}}) commands.

## Set Tolerations for a PostgreSQL Cluster

[Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) help with the scheduling of Pods to appropriate nodes. There are many reasons that a Kubernetes administrator may want to use tolerations, such as restricting the types of Pods that can be assigned to particular nodes.

The PostgreSQL Operator supports adding tolerations to PostgreSQL instances using the `--toleration` flag. The format for adding a toleration is as such:

```
rule:Effect
```

or

```
rule
```

where a `rule` can represent existence (e.g. `key`) or equality (`key=value`) and `Effect` is one of `NoSchedule`, `PreferNoSchedule`, or `NoExecute`. For more information on how tolerations work, please refer to the [Kubernetes documentation](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/).

You can assign multiple tolerations to a PostgreSQL cluster.

For example, to add two tolerations to a new PostgreSQL cluster, one that is an existence toleration for a key of `ssd` and the other that is an equality toleration for a key/value pair of `zone`/`east`, you can run the following command:

```
pgo create cluster hippo \
  --toleration=ssd:NoSchedule \
  --toleration=zone=east:NoSchedule
```

Tolerations can be updated on an existing cluster using the [`pgo update cluster`]({{ relref "pgo-client/reference/pgo_update_cluster.md" }}) command. For example, to add a toleration of `zone=west:NoSchedule` and remove the toleration of `zone=east:NoSchedule`, you could run the following command:

```
pgo update cluster hippo \
  --toleration=zone=west:NoSchedule \
  --toleration=zone-east:NoSchedule-
```

You can also add or edit tolerations directly on the `pgclusters.crunchydata.com` custom resource and the PostgreSQL Operator will roll out the changes to the appropriate instances.

## Customize PostgreSQL Configuration

PostgreSQL provides a lot of different knobs that can be used to fine tune the [configuration](https://www.postgresql.org/docs/current/runtime-config.html) for your workload. While you can [customize your PostgreSQL configuration]({{< relref "advanced/custom-configuration.md" >}}) after your cluster has been deployed, you may also want to load in your custom configuration during initialization.

The PostgreSQL Operator uses [Patroni](https://patroni.readthedocs.io/) to help manage cluster initialization and high availability. To understand how to build out a configuration file to be used to customize your PostgreSQL cluster, please review the [Patroni documentation](https://patroni.readthedocs.io/en/latest/SETTINGS.html).

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

Next, create a [`ConfigMap`](https://kubernetes.io/docs/tasks/configure-pod-container/configure-pod-configmap/) called `hippo-custom-config` like so:

```
kubectl -n pgo create configmap hippo-custom-config --from-file=postgres-ha.yaml
```

You can then have you new PostgreSQL cluster use `hippo-custom-config` as part of its cluster initialization by using the `--custom-config` flag of `pgo create cluster`:

```
pgo create cluster hippo --custom-config=hippo-custom-config
```

After your cluster is initialized, [connect to your cluster]({{< relref "tutorial/connect-cluster.md" >}}) and confirm that your settings have been applied:

```
SHOW shared_buffers;

 shared_buffers
----------------
 2GB
```

## Troubleshooting

### PostgreSQL Pod Can't Be Scheduled

There are many reasons why a PostgreSQL Pod may not be scheduled:

- **Resources are unavailable**. Ensure that you have a Kubernetes [Node](https://kubernetes.io/docs/concepts/architecture/nodes/) with enough resources to satisfy your memory or CPU Request.
- **PVC cannot be provisioned**. Ensure that you request a PVC size that is available, or that your PVC storage class is set up correctly.
- **Node affinity rules cannot be satisfied**. If you assigned a node label, ensure that the Nodes with that label are available for scheduling. If they are, ensure that there are enough resources available.
- **Pod anti-affinity rules cannot be satisfied**. This most likely happens when [pod anti-affinity]({{< relref "architecture/high-availability/_index.md" >}}#how-the-crunchy-postgresql-operator-uses-pod-anti-affinity) is set to `required` and there are not enough Nodes available for scheduling. Consider adding more Nodes or relaxing your anti-affinity rules.

### PostgreSQL Pod reports "Authentication Failed for `ccp_monitoring`"

This is a temporary error that occurs when a new PostgreSQL cluster is first
initialized with the `--metrics` flag. The `crunchy-postgres-exporter` container
within the PostgreSQL Pod may be ready before the container with PostgreSQL is
ready. If a message in your logs further down displays a timestamp, e.g.:

```
             now              
-------------------------------
2020-11-10 08:23:15.968196-05
```

Then the `ccp_monitoring` user is properly reconciled with the PostgreSQL
cluster.

If the error message does not go away, this could indicate a few things:

- The PostgreSQL instance has not initialized. Check to ensure that PostgreSQL
has successfully started.
- The password for the `ccp_monitoring` user has changed. In this case you will
need to update the Secret with the monitoring credentials.

### PostgreSQL Pod Not Scheduled to Nodes Matching Tolerations

While Kubernetes [Tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) allow for Pods to be scheduled to Nodes based on their taints, this does not mean that the Pod _will_ be assigned to those nodes. To provide Kubernetes scheduling guidance on where a Pod should be assigned, you must also use [Node Affinity]({{< relref "architecture/high-availability/_index.md" >}}#node-affinity).

## Next Steps

As mentioned at the beginning, there are a lot more customizations that you can make to your PostgreSQL cluster, and we will cover those as the tutorial progresses! This section was to get you familiar with some of the most common customizations, and to explore how many options `pgo create cluster` has!

Now you have your PostgreSQL cluster up and running and using the resources as you see fit. What if you want to make changes to the cluster? We'll explore some of the commands that can be used to update your PostgreSQL cluster!
