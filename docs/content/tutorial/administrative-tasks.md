---
title: "Administrative Tasks"
date:
draft: false
weight: 105
---

## Manually Restarting PostgreSQL

There are times when you might need to manually restart PostgreSQL. This can be done by adding or updating a custom annotation to the cluster's `spec.metadata.annotations` section. PGO will notice the change and perform a [rolling restart]({{< relref "/architecture/high-availability.md" >}}#rolling-update).

For example, if you have a cluster named `hippo` in the namespace `postgres-operator`, all you need to do is patch the hippo postgrescluster with the following:

```shell
kubectl patch postgrescluster/hippo -n postgres-operator --type merge \
  --patch '{"spec":{"metadata":{"annotations":{"restarted":"'"$(date)"'"}}}}'
```

Watch your hippo cluster: you will see the rolling update has been triggered and the restart has begun.

## Shutdown

You can shut down a Postgres cluster by setting the `spec.shutdown` attribute to `true`. You can do this by editing the manifest, or, in the case of the `hippo` cluster, executing a comand like the below:

```
kubectl patch postgrescluster/hippo -n postgres-operator --type merge \
  --patch '{"spec":{"shutdown": true}}'
```

Shutting down a cluster will terminate all of the active Pods. Any Statefulsets or Deployments are scaled to `0`.

To turn a Postgres cluster that is shut down back on, you can set `spec.shutdown` to `false`.

## Rotating TLS Certificates

Credentials should be invalidated and replaced (rotated) as often as possible
to minimize the risk of their misuse. Unlike passwords, every TLS certificate
has an expiration, so replacing them is inevitable. When you use your own TLS
certificates with PGO, you are responsible for replacing them appropriately.
Here's how.


PGO automatically detects and loads changes to the contents of PostgreSQL server
and replication Secrets without downtime. You or your certificate manager need
only replace the values in the Secret referenced by `spec.customTLSSecret`.

If instead you change `spec.customTLSSecret` to refer to a new Secret or new fields,
PGO will perform a [rolling restart]({{< relref "/architecture/high-availability.md" >}}#rolling-update).

{{% notice info %}}
When changing the PostgreSQL certificate authority, make sure to update
[`customReplicationTLSSecret`]({{< relref "/tutorial/customize-cluster.md" >}}#customize-tls) as well.
{{% /notice %}}

PgBouncer needs to be restarted after its certificates change.
There are a few ways to do it:

1. Store the new certificates in a new Secret. Edit the PostgresCluster object
   to refer to the new Secret, and PGO will perform a rolling restart of PgBouncer.
   ```yaml
   spec:
     proxy:
       pgBouncer:
         customTLSSecret:
           name: hippo.pgbouncer.new.tls
   ```

   _or_

2. Replace the old certificates in the current Secret. PGO doesn't notice when
   the contents of your Secret change, so you need to trigger a rolling restart
   of PgBouncer. Edit the PostgresCluster object to add a unique annotation.
   The name and value are up to you, so long as the value differs from the
   previous value.
   ```yaml
   spec:
     proxy:
       pgBouncer:
         metadata:
           annotations:
             restarted: Q1-certs
   ```

   This `kubectl patch` command uses your local date and time:

   ```shell
   kubectl patch postgrescluster/hippo --type merge \
     --patch '{"spec":{"proxy":{"pgBouncer":{"metadata":{"annotations":{"restarted":"'"$(date)"'"}}}}}}'
   ```

## Next Steps

We've covered a lot in terms of building, maintaining, scaling, customizing, restarting, and expanding our Postgres cluster. However, there may come a time where we need to [delete our Postgres cluster]({{< relref "delete-cluster.md" >}}). How do we do that?
