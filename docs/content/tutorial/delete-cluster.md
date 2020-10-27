---
title: "Delete a Postgres Cluster"
draft: false
weight: 150
---

There are many reasons you may want to delete a PostgreSQL cluster, and a few different questions to consider, such as do you want to permanently delete the data or save it for later use?

The PostgreSQL Operator offers several different workflows for deleting a cluster, from wiping all assets, to keeping PVCs of your data directory, your backup repository, or both.

## Delete Everything

Deleting everything in a PostgreSQL cluster is a simple as using the [`pgo delete cluster`]({{< relref "pgo-client/reference/pgo_delete_cluster.md" >}}) command. For example, to delete the `hippo` cluster:

```
pgo delete cluster hippo
```

This command launches a [Job](https://kubernetes.io/docs/concepts/workloads/controllers/job/) that uses the `pgo-rmdata` container to delete all of the Kubernetes objects associated with this PostgreSQL cluster. Once the `pgo-rmdata` Job finishes executing, all of your data, configurations, etc. will be removed.

## Keep Backups

If you want to keep your backups, which can be used to [restore your PostgreSQL cluster at a later time]({{< relref "architecture/disaster-recovery.md">}}#restore-to-a-new-cluster) (a popular method for cloning and having sample data for your development team to use!), use the `--keep-backups` flag! For example, to delete the `hippo` PostgreSQL cluster but keep all of its backups:

```
pgo delete cluster hippo --keep-backups
```

This keeps the pgBackRest PVC which follows the pattern `<clusterName>-hippo-pgbr-repo` (e.g. `hippo-pgbr-repo`) and any PVCs that were created using the `pgdump` method of [`pgo backup`]({{< relref "pgo-client/reference/pgo_backup.md">}}).

## Keep the PostgreSQL Data Directory

You may also want to delete your PostgreSQL cluster data directory, which is the core of your database, but remove any actively running Pods. This can be accomplished with the `--keep-data` flag. For example, to keep the data directory of the `hippo` cluster:

```
pgo delete cluster hippo --keep-data
```

Once the `pgo-rmdata` Job completes, your data PVC for `hippo` will still remain, but you will be unable to access it unless you attach it to a new PostgreSQL instance. The easiest way to access your data again is to create a PostgreSQL cluster with the same name:

```
pgo create cluster hippo
```

and the PostgreSQL Operator will re-attach your PVC to the newly running cluster.

## Next Steps

We've covered the fundamental lifecycle elements of the PostgreSQL Operator, but there is much more to learn! If you're curious about how things work in the PostgreSQL Operator and how to perform daily tasks, we suggest you continue with the following sections:

- [Architecture]({{< relref "architecture/_index.md" >}})
- [Common `pgo` Client Tasks]({{< relref "pgo-client/common-tasks.md" >}})

The tutorial will now go into some more advanced topics. Up next, learn how to [secure connections to your PostgreSQL clusters with TLS]({{< relref "tutorial/tls.md" >}}).
