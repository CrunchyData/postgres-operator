---
title: "Administrative Tasks"
date:
draft: false
weight: 105
---

## Manually Restarting PostgreSQL

There are times when you might need to manually restart PostgreSQL. This can be done by adding or updating a custom annotation to the cluster's `spec.metadata.annotations` section. PGO will notice the change and perform a [rolling restart]({{< relref "/architecture/high-availability.md" >}}#rolling-update).

For example, if you have a cluster named `hippo` in the namespace `postgres-operator`, all you need to do is patch the hippo PostgresCluster with the following:

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

PGO automatically notifies PgBouncer when there are changes to the contents of
PgBouncer certificate Secrets. Recent PgBouncer versions load those changes
without downtime, but versions prior to 1.16.0 need to be restarted manually.
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

## Changing the Primary

There may be times when you want to change the primary in your HA cluster. This can be done
using the `patroni.switchover` section of the PostgresCluster spec. It allows
you to enable switchovers in your PostgresClusters, target a specific instance as the new
primary, and run a failover if your PostgresCluster has entered a bad state.

Let's go through the process of performing a switchover!

First you need to update your spec to prepare your cluster to change the primary. Edit your spec
to have the following fields:

```yaml
spec:
  patroni:
    switchover:
      enabled: true
```

After you apply this change, PGO will be looking for the trigger to perform a switchover in your
cluster. You will trigger the switchover by adding the `postgres-operator.crunchydata.com/trigger-switchover`
annotation to your custom resource. The best way to set this annotation is
with a timestamp, so you know when you initiated the change.

For example, for our `hippo` cluster, we can run the following command to trigger the switchover:

```shell
kubectl annotate -n postgres-operator postgrescluster hippo \
  postgres-operator.crunchydata.com/trigger-switchover="$(date)"
```

{{% notice tip %}}
If you want to perform another switchover you can re-run the annotation command and add the `--overwrite` flag:

```shell
kubectl annotate -n postgres-operator postgrescluster hippo --overwrite \
  postgres-operator.crunchydata.com/trigger-switchover="$(date)"
```
{{% /notice %}}

PGO will detect this annotation and use the Patroni API to request a change to the current primary!

The roles on your database instance Pods will start changing as Patroni works. The new primary
will have the `master` role label, and the old primary will be updated to `replica`.

The status of the switch will be tracked using the `status.patroni.switchover` field. This will be set
to the value defined in your trigger annotation. If you use a timestamp as the annotation this is
another way to determine when the switchover was requested.

After the instance Pod labels have been updated and `status.patroni.switchover` has been set, the
primary has been changed on your cluster!

{{% notice info %}}
After changing the primary, we recommend that you disable switchovers by setting `spec.patroni.switchover.enabled`
to false or remove the field from your spec entirely. If the field is removed the corresponding
status will also be removed from the PostgresCluster.
{{% /notice %}}


#### Targeting an instance

Another option you have when switching the primary is providing a target instance as the new
primary. This target instance will be used as the candidate when performing the switchover.
The `spec.patroni.switchover.targetInstance` field takes the name of the instance that you are switching to.

This name can be found in a couple different places; one is as the name of the StatefulSet and
another is on the database Pod as the `postgres-operator.crunchydata.com/instance` label. The
following commands can help you determine who is the current primary and what name to use as the
`targetInstance`:

```shell-session
$ kubectl get pods -l postgres-operator.crunchydata.com/cluster=hippo \
    -L postgres-operator.crunchydata.com/instance \
    -L postgres-operator.crunchydata.com/role

NAME                      READY   STATUS      RESTARTS   AGE     INSTANCE               ROLE
hippo-instance1-jdb5-0    3/3     Running     0          2m47s   hippo-instance1-jdb5   master
hippo-instance1-wm5p-0    3/3     Running     0          2m47s   hippo-instance1-wm5p   replica
```

In our example cluster `hippo-instance1-jdb5` is currently the primary meaning we want to target
`hippo-instance1-wm5p` in the switchover. Now that you know which instance is currently the
primary and how to find your `targetInstance`, let's update your cluster spec:

```yaml
spec:
  patroni:
    switchover:
      enabled: true
      targetInstance: hippo-instance1-wm5p
```

After applying this change you will once again need to trigger the switchover by annotating the
PostgresCluster (see above commands). You can verify the switchover has completed by checking the
Pod role labels and `status.patroni.switchover`.

#### Failover

Finally, we have the option to failover when your cluster has entered an unhealthy state. The
only spec change necessary to accomplish this is updating the `spec.patroni.switchover.type`
field to the `Failover` type. One note with this is that a `targetInstance` is required when
performing a failover. Based on the example cluster above, assuming `hippo-instance1-wm5p` is still
a replica, we can update the spec:

```yaml
spec:
  patroni:
    switchover:
      enabled: true
      targetInstance: hippo-instance1-wm5p
      type: Failover
```

Apply this spec change and your PostgresCluster will be prepared to perform the failover. Again
you will need to trigger the switchover by annotating the PostgresCluster (see above commands)
and verify that the Pod role labels and `status.patroni.switchover` are updated accordingly.

{{% notice warning %}}
Errors encountered in the switchover process can leave your cluster in a bad
state. If you encounter issues, found in the operator logs, you can update the spec to fix the
issues and apply the change. Once the change has been applied, PGO will attempt to perform the
switchover again.
{{% /notice %}}

## Next Steps

We've covered a lot in terms of building, maintaining, scaling, customizing, restarting, and expanding our Postgres cluster. However, there may come a time where we need to [delete our Postgres cluster]({{< relref "delete-cluster.md" >}}). How do we do that?
