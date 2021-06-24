---
title: "Disaster Recovery and Cloning"
date:
draft: false
weight: 85
---

Perhaps someone accidentally dropped the `users` table. Perhaps you want to clone your production database to a step-down environment. Perhaps you want to exercise your disaster recovery system (and it is important that you do!).

Regardless of scenario, it's important to know how you can perform a "restore" operation with PGO to be able to recovery your data from a particular point in time, or clone a database for other purposes.

Let's look at how we can perform different types of restore operations. First, let's understand the core restore properties on the custom resource.

## Restore Properties

There are several attributes on the custom resource that are important to understand as part of the restore process. All of these attributes are grouped together in the `spec.dataSource.postgresCluster` section of the custom resource.

Please review the table below to understand how each of these attributes work in the context of setting up a restore operation.

- `spec.dataSource.postgresCluster.clusterName`: The name of the cluster that you are restoring from. This corresponds to the `metadata.name` attribute on a different `postgrescluster` custom resource.
- `spec.dataSource.postgresCluster.repoName`: The name of the pgBackRest repository from the `spec.dataSource.postgresCluster.clusterName` to use for the restore. Can be one of `repo1`, `repo2`, `repo3`, or `repo4`. The repository must exist in the other cluster.
- `spec.dataSource.postgresCluster.options`: Any additional [pgBackRest restore options](https://pgbackrest.org/command.html#command-restore) or general options you would like to pass in. For example, you may want to set `--process-max` to help improve performance on larger databases.

Let's walk through some examples for how we can clone and restore our databases.

## Clone a Postgres Cluster

Let's create a clone of our [`hippo`]({{< relref "./create-cluster.md" >}}) cluster that we created previously. We know that our cluster is named `hippo` (baesd on its `metadata.name`) and that we only have a single backup repository called `repo1`.

Let's call our new cluster `elephant`. We can create a clone of the `hippo` cluster using a manifest like this:

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: elephant
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
  image: registry.developers.crunchydata.com/crunchydata/crunchy-postgres-ha:centos8-13.3-0
  postgresVersion: 13
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  archive:
    pgbackrest:
      image: registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:centos8-2.33-0
      repoHost:
        dedicated: {}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
```

Note this section of the spec:

```
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
```

This is the part that tells PGO to create the `elephant` cluster as an independent copy of the `hippo` cluster.

The above is all you need to do to clone a Postgres cluster! PGO will work on creating a copy of your data on a new persistent volume claim (PVC) and work on initializing your cluster to spec. Easy!

## Perform a Point-in-time-Recovery (PITR)

Did someone drop the user table? You may want to perform a point-in-time-recovery (PITR) to revert your database back to a state before a change occurred. Fortunately, PGO can help you do that.

You can set up a PITR using the [restore](https://pgbackrest.org/command.html#command-restore) command of [pgBackRest](https://www.pgbackrest.org), the backup management tool that powers the disaster recovery capabilities of PGO. You will need to set a few options on `spec.dataSource.postgresCluster.options` to perform a PITR. These options include:

- `--type=time`: This tells pgBackRest to perform a PITR.
- `--target`: Where to perform the PITR to. Any example recovery target is `2021-06-09 14:15:11 EDT`.
- `--set` (optional): Choose which backup to start the PITR from.

Let's use the `elephant` example above. Let's say we want to perform a point-in-time-recovery (PITR) to `2021-06-09 14:15:11 EDT`, we can use the following manifest:

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: elephant
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
      options:
      - --type=time
      - --target="2021-06-09 14:15:11 EDT"
  image: registry.developers.crunchydata.com/crunchydata/crunchy-postgres-ha:centos8-13.3-0
  postgresVersion: 13
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  archive:
    pgbackrest:
      image: registry.developers.crunchydata.com/crunchydata/crunchy-pgbackrest:centos8-2.33-0
      repoHost:
        dedicated: {}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
```

The section to pay attention to is this:

```
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
      options: --type=time --target="2021-06-09 14:15:11 EDT"
```

Notice how we put in the options to specify where to make the PITR.

Using the above manfiest, PGO will go ahead and create a new Postgres cluster that recovers its data up until `2021-06-09 14:15:11 EDT`. At that point, the cluster is promoted and you can start accessing your database from that specific point in time!

Note that to perform a PITR, you must have a backup that is at least as old as your PITR time. In other words, you can't perform a PITR back to a time where you do not have a backup!

## Next Steps

Now we've seen how to clone a cluster and perform a point-in-time-recovery, let's see how we can [monitor]({{< relref "./monitoring.md" >}}) our Postgres cluster to detect and prevent issues from occurring.
