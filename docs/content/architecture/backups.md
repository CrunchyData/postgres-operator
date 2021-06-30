---
title: "Backup Management"
date:
draft: false
weight: 120
---

When using the PostgreSQL Operator, the answer to the question "do you take
backups of your database" is automatically "yes!"

The PostgreSQL Operator uses the open source
[pgBackRest](https://pgbackrest.org) backup and restore utility that is designed
for working with databases that are many terabytes in size. As described in the
[Provisioning](/architecture/provisioning/) section, pgBackRest is enabled by
default as it permits the PostgreSQL Operator to automate some advanced as well
as convenient behaviors, including:

- Efficient provisioning of new replicas that are added to the PostgreSQL
cluster
- Preventing replicas from falling out of sync from the PostgreSQL primary by
allowing them to replay old WAL logs
- Allowing failed primaries to automatically and efficiently heal using the
"delta restore" feature
- Serving as the basis for the cluster cloning feature
- ...and of course, allowing for one to take full, differential, and incremental
backups and perform full and point-in-time restores

Below is one example of how PGO manages backups with both a local storage and a Amazon S3 configuration.

![PostgreSQL Operator pgBackRest Integration](/images/postgresql-cluster-dr-base.png)

The PostgreSQL Operator leverages a pgBackRest repository to facilitate the
usage of the pgBackRest features in a PostgreSQL cluster. When a new PostgreSQL
cluster is created, it simultaneously creates a pgBackRest repository.

You can store your pgBackRest backups in up to four different locations and using four different storage types:

- Any Kubernetes supported storage class
- Amazon S3 (or S3 equivalents like MinIO)
- Google Cloud Storage (GCS)
- Azure Blob Storage

The pgBackRest repository consists of the following Kubernetes objects:

- A Deployment
- A Secret that contains information that is specific to the PostgreSQL cluster
that it is deployed with (e.g. SSH keys, AWS S3 keys, etc.)
- A Service

The PostgreSQL primary is automatically configured to use the
`pgbackrest archive-push` and push the write-ahead log (WAL) archives to the
correct repository.

## Backups

PGO supports three types of pgBackRest backups:

- Full (`full`): A full backup of all the contents of the PostgreSQL cluster
- Differential (`diff`): A backup of only the files that have changed since the
last full backup
- Incremental (`incr`):  A backup of only the files that have changed since the
last full or differential backup

## Scheduling Backups

Any effective disaster recovery strategy includes having regularly scheduled
backups. PGO enables this by managing a series of Kubernetes CronJobs to ensure that backups are executed at scheduled times.

Note that pgBackRest presently only supports taking one backup at a time. This may change in a future release, but for the time being we suggest that you stagger your backup times.

Please see the [backup configuration tutorial]({{< relref "tutorial/backups.md" >}}) for how to set up backup schedules.

## Restores

The PostgreSQL Operator supports the ability to perform a full restore on a
PostgreSQL cluster as well as a point-in-time-recovery. There are two types of
ways to restore a cluster:

- Restore to a new cluster
- Restore in-place

For examples for this, please see the [disaster recovery tutorial]({{< relref "tutorial/disaster-recovery.md" >}})

### Setting Backup Retention Policies

Unless specified, pgBackRest will keep an unlimited number of backups. You can specify backup retention using the `repoN-retention-` options. Please see the [backup configuration tutorial]({{< relref "tutorial/backups.md" >}}) for examples.

## Deleting a Backup

{{% notice warning %}}
If you delete a backup that is *not* set to expire, you may be unable to meet
your retention requirements. If you are deleting backups to free space, it is
recommended to delete your oldest backups first.
{{% /notice %}}

A backup can be deleted by running the [`pgbackrest expire`](https://pgbackrest.org/command.html#command-expire) command directly on the pgBackRest repository Pod or a Postgres instance.
