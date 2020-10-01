---
title: "Update a Postgres Cluster"
draft: false
weight: 140
---

You've done it: your application is a huge success! It's so successful that you database needs more resources to keep up with the demand. How do you add more resources to your PostgreSQL cluster?

The PostgreSQL Operator provides several options to [update a cluster's]({{< relref "pgo-client/reference/pgo_update_cluster.md" >}}) resource utilization, including:

- Resource allocations (e.g. Memory, CPU, PVC size)
- Tablespaces
- Annotations
- Availability options
- [Configuration]({{< relref "advanced/custom-configuration.md" >}})

and more. There are additional actions that can be taken as well outside of the update process, including [scaling a cluster]({{< relref "architecture/high-availability/_index.md" >}}), adding a pgBouncer or [pgAdmin 4]({{< relref "architecture/pgadmin4.md" >}}) Deployment, and more.

The goal of this section is to present a few of the common actions that can be taken to update your PostgreSQL cluster so it has the resources and configuration that you require.

## Update CPU / Memory

You can update the CPU and memory resources available to the Pods in your PostgreSQL cluster by using the [`pgo update cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) command. By using this method, the PostgreSQL instances are safely shut down and the new resources are applied in a rolling fashion (though we caution that a brief downtime may still occur).

Customizing CPU and memory does add more resources to your PostgreSQL cluster, but to fully take advantage of additional resources, you will need to [customize your PostgreSQL configuration]({{< relref "advanced/custom-configuration.md" >}}) and tune parameters such as `shared_buffers` and others.

### Customize CPU / Memory for PostgreSQL

The PostgreSQL Operator provides several flags for [`pgo update cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) to help manage resources for a PostgreSQL instance:

- `--cpu`: Specify the CPU Request for a PostgreSQL instance
- `--cpu-limit`: Specify the CPU Limit for a PostgreSQL instance
- `--memory`: Specify the Memory Request for a PostgreSQL instance
- `--memory-limit`: Specify the Memory Limit for a PostgreSQL instance

For example, to update a PostgreSQL cluster that makes a CPU Request of 2.0 with a CPU Limit of 4.0 and a Memory Request of 4Gi with a Memory Limit of 6Gi:

```
pgo update cluster hippo \
  --cpu=2.0 --cpu-limit=4.0 \
  --memory=4Gi --memory-limit=6Gi
```

### Customize CPU / Memory for Crunchy PostgreSQL Exporter Sidecar

If your [PostgreSQL cluster has monitoring](#create-a-postgresql-cluster-with-monitoring), you may want to adjust the resources of the `crunchy-postgres-exporter` sidecar that runs next to each PostgreSQL instnace. You can do this with the following flags:

- `--exporter-cpu`: Specify the CPU Request for a `crunchy-postgres-exporter` sidecar
- `--exporter-cpu-limit`: Specify the CPU Limit for a `crunchy-postgres-exporter` sidecar
- `--exporter-memory`: Specify the Memory Request for a `crunchy-postgres-exporter` sidecar
- `--exporter-memory-limit`: Specify the Memory Limit for a `crunchy-postgres-exporter` sidecar

For example, to update a PostgreSQL cluster with a metrics sidecar with custom CPU and memory requests + limits, you could do the following:

```
pgo update cluster hippo \
  --exporter-cpu=0.5 --exporter-cpu-limit=1.0 \
  --exporter-memory=256Mi --exporter-memory-limit=1Gi
```

### Customize CPU / Memory for pgBackRest

You can also customize the CPU and memory requests and limits for pgBackRest with the following flags:

- `--pgbackrest-cpu`: Specify the CPU Request for pgBackRest
- `--pgbackrest-cpu-limit`: Specify the CPU Limit for pgBackRest
- `--pgbackrest-memory`: Specify the Memory Request for pgBackRest
- `--pgbackrest-memory-limit`: Specify the Memory Limit for pgBackRest

For example, to update a PostgreSQL cluster with custom CPU and memory requests + limits for pgBackRest, you could do the following:

```
pgo update cluster hippo \
  --pgbackrest-cpu=0.5 --pgbackrest-cpu-limit=1.0 \
  --pgbackrest-memory=256Mi --pgbackrest-memory-limit=1Gi
```

## Customize PostgreSQL Configuration

PostgreSQL provides a lot of different knobs that can be used to fine tune the [configuration](https://www.postgresql.org/docs/current/runtime-config.html) for your workload. While you can [customize your PostgreSQL configuration]({{< relref "advanced/custom-configuration.md" >}}) after your cluster has been deployed, you may also want to load in your custom configuration during initialization.

The configuration can be customized by editing the `<clusterName>-pgha-config` ConfigMap. For example, with the `hippo` cluster:

```
kubectl -n pgo edit configmap hippo-pgha-config
```

We recommend that you read the section on how to [customize your PostgreSQL configuration]({{< relref "advanced/custom-configuration.md" >}}) to find out how to customize your configuration.

## Troubleshooting

### Configuration Did Not Update

Any updates to a ConfigMap may take a few moments to propagate to all of your Pods. Once it is propagated, the PostgreSQL Operator will attempt to reload the new configuration on each Pod.

If the information has propagated but the Pods have not been reloaded, you can force an explicit reload with the [`pgo reload`]({{< relref "pgo-client/reference/pgo_reload.md" >}}) command:

```
pgo reload hippo
```

Some customized configuration settings can only be applied to your PostgreSQL cluster after it is restarted. For example, to restart the `hippo` cluster, you can use the [`pgo restart`]({{< relref "pgo-client/reference/pgo_restart.md" >}}) command:

```
pgo restart hippo
```

## Next Steps

We've seen how to create, customize, and update a PostgreSQL cluster with the PostgreSQL Operator. What about [deleting a PostgreSQL cluster]({{< relref "tutorial/delete-cluster.md" >}})?
