---
title: "Disaster Recovery"
date:
draft: false
weight: 200
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
backpus and perform full and point-in-time restores

![PostgreSQL Operator pgBackRest Integration](/images/postgresql-cluster-dr-base.png)

The PostgreSQL Operator leverages a pgBackRest repository to facilitate the
usage of the pgBackRest features in a PostgreSQL cluster. When a new PostgreSQL
cluster is created, it simultaneously creates a pgBackRest repository as
described in the [Provisioning](/architecture/provisioning/) section.

At PostgreSQL cluster creation time, you can specify a specific Storage Class
for the pgBackRest repository. Additionally, you can also specify the type of
pgBackRest repository that can be used, including:

- `local`: Uses the storage that is provided by the Kubernetes cluster's Storage
Class that you select
- `s3`: Use Amazon S3 or an object storage system that uses the S3 protocol
- `local,s3`: Use both the storage that is provided by the Kubernetes cluster's
Storage Class that you select AND Amazon S3 (or equivalent object storage system
that uses the S3 protocol)

The pgBackRest repository consists of the following Kubernetes objects:

- A Deployment
- A Secret that contains information that is specific to the PostgreSQL cluster
that it is deployed with (e.g. SSH keys, AWS S3 keys, etc.)
- A Service

The PostgreSQL primary is automatically configured to use the
`pgbackrest archive-push` and push the write-ahead log (WAL) archives to the
correct repository.

## Backups

Backups can be taken with the `pgo backup` command

The PostgreSQL Operator supports three types of pgBackRest backups:

- Full (`full`): A full backup of all the contents of the PostgreSQL cluster
- Differential (`diff`): A backup of only the files that have changed since the
last full backup
- Incremental (`incr`):  A backup of only the files that have changed since the
last full or differential backup

By default, `pgo backup` will attempt to take an **incremental (`incr`)** backup
unless otherwise specified.

For example, to specify a full backup:

```shell
pgo backup hacluster --backup-opts="--type=full"
```

The PostgreSQL Operator also supports setting pgBackRest retention policies as
well for backups. For example, to take a full backup and to specify to only keep
the last 7 backups:

```shell
pgo backup hacluster --backup-opts="--type=full --repo1-retention-full=7"
```

## Restores

The PostgreSQL Operator supports the ability to perform a full restore on a
PostgreSQL cluster as well as a point-in-time-recovery using the `pgo restore`
command. Note that both of these options are **destructive** to the existing
PostgreSQL cluster; to "restore" the PostgreSQL cluster to a new deployment,
please see the [Clone](/pgo-client/common-tasks/#clone-a-postgresql-cluster) section.

The `pgo restore` command lets you specify the point at which you want to
restore your database using the `--pitr-target` flag with the `pgo restore`
command.

**NOTE**: Ensure you are backing up your PostgreSQL cluster regularly, as this
will help expedite your restore times. The next section will cover scheduling
regular backups.

When the PostgreSQL Operator issues a restore, the following actions are taken
on the cluster:

- The PostgreSQL Operator disables the "autofail" mechanism so that no failovers
will occur during the restore.
- Any replicas that may be associated with the PostgreSQL cluster are destroyed
- A new Persistent Volume Claim (PVC) is allocated using the specifications
provided for the primary instance. This may have been set with the
`--storage-class` flag when the cluster was originally created
- A Kubernetes Job is created that will perform a pgBackRest restore operation
to the newly allocated PVC. This is facilitated by the `pgo-backrest-restore`
container image.

![PostgreSQL Operator Restore Step 1](/images/postgresql-cluster-restore-step-1.png)

- When restore Job successfully completes, a new Deployment for the PostgreSQL
cluster primary instance is created. A recovery is then issued to the specified
point-in-time, or if it is a full recovery, up to the point of the latest WAL
archive in the repository.
- Once the PostgreSQL primary instance is available, the PostgreSQL Operator
will take a new, full backup of the cluster.

![PostgreSQL Operator Restore Step 2](/images/postgresql-cluster-restore-step-2.png)

At this point, the PostgreSQL cluster has been restored. However, you will need
to re-enable autofail if you would like your PostgreSQL cluster to be
highly-available. You can re-enable autofail with this command:

```shell
pgo update cluster hacluster --autofail=true
```

## Scheduling Backups

Any effective disaster recovery strategy includes having regularly scheduled
backups. The PostgreSQL Operator enables this through its scheduling sidecar
that is deployed alongside the Operator.

The PostgreSQL Operator Scheduler is essentially a [cron](https://en.wikipedia.org/wiki/Cron)
server that will run jobs that it is specified. Schedule commands use the cron
syntax to set up scheduled tasks.

![PostgreSQL Operator Schedule Backups](/images/postgresql-cluster-dr-schedule.png)

For example, to schedule a full backup once a day at 1am, the following command
can be used:

```shell
pgo create schedule hacluster --schedule="0 1 * * *" \
  --schedule-type=pgbackrest  --pgbackrest-backup-type=full
```

To schedule an incremental backup once every 3 hours:

```shell
pgo create schedule hacluster --schedule="0 */3 * * *" \
  --schedule-type=pgbackrest  --pgbackrest-backup-type=incr
```

### Setting Backup Retention Policies

Unless specified, pgBackRest will keep an unlimited number of backups. As part
of your regularly scheduled backups, it is encouraged for you to set a retention
policy. This can be accomplished using the `--repo1-retention-full` for full
backups and `--repo1-retention-diff` for differential backups via the
`--schedule-opts` parameter.

For example, using the above example of taking a nightly full backup, you can
specify a policy of retaining 21 backups using the following command:

```shell
pgo create schedule hacluster --schedule="0 1 * * *" \
  --schedule-type=pgbackrest  --pgbackrest-backup-type=full \
  --schedule-opts="--repo1-retention-full=21"
```

### Schedule Expression Format

Schedules are expressed using the following rules, which should be familiar to
users of cron:

```
Field name   | Mandatory? | Allowed values  | Allowed special characters
----------   | ---------- | --------------  | --------------------------
Seconds      | Yes        | 0-59            | * / , -
Minutes      | Yes        | 0-59            | * / , -
Hours        | Yes        | 0-23            | * / , -
Day of month | Yes        | 1-31            | * / , - ?
Month        | Yes        | 1-12 or JAN-DEC | * / , -
Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?
```

## Using S3

The PostgreSQL Operator integration with pgBackRest allows it to use the AWS S3
object storage system, as well as other object storage systems that implement
the S3 protocol.

In order to enable S3 storage, it is helpful to provide some of the S3
information prior to deploying the PostgreSQL Operator, or updating the
`pgo-config` ConfigMap and restarting the PostgreSQL Operator pod.

First, you will need to add the proper S3 bucket name, AWS S3 endpoint and
the AWS S3 region to the `Cluster` section of the `pgo.yaml`
[configuration file](/configuration/pgo-yaml-configuration/):

```yaml
Cluster:
  BackrestS3Bucket: my-postgresql-backups-example
  BackrestS3Endpoint: s3.amazonaws.com
  BackrestS3Region: us-east-1
```

These values can also be set on a per-cluster basis with the
`pgo create cluster` command, i.e.:


- `--pgbackrest-s3-bucket` - specifics the AWS S3 bucket that should be utilized
- `--pgbackrest-s3-endpoint` specifies the S3 endpoint that should be utilized
- `--pgbackrest-s3-key` - specifies the AWS S3 key that should be utilized
- `--pgbackrest-s3-key-secret`- specifies the AWS S3 key secret that should be
utilized
- `--pgbackrest-s3-region` - specifies the AWS S3 region that should be utilized

Sensitive information, such as the values of the AWS S3 keys and secrets, are
stored in Kubernetes Secrets and are securely mounted to the PostgreSQL
clusters.

To enable a PostgreSQL cluster to use S3, the `--pgbackrest-storage-type` on the
`pgo create cluster` command needs to be set to `s3` or `local,s3`.

Once configured, the `pgo backup` and `pgo restore` commands will work with S3
similarly to the above!
