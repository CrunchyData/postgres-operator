---
title: "Common pgo Client Tasks"
date:
draft: false
weight: 20
---

While the full [`pgo` client reference](/pgo-client/reference/) will tell you
everything you need to know about how to use `pgo`, it may be helpful to see
several examples on how to conduct "day-in-the-life" tasks for administrating
PostgreSQL cluster with the PostgreSQL Operator.

The below guide covers many of the common operations that are required when
managing PostgreSQL clusters. The guide is broken up by different administrative
topics, such as provisioning, high-availability, etc.

## Setup Before Running the Examples

Many of the `pgo` client commands require you to specify a namespace via the
`-n` or `--namespace` flag. While this is a very helpful tool when managing
PostgreSQL deployxments across many Kubernetes namespaces, this can become
onerous for the intents of this guide.

If you install the PostgreSQL Operator using the [quickstart](/quickstart/)
guide, you will install the PostgreSQL Operator to a namespace called `pgo`. We
can choose to always use one of these namespaces by setting the `PGO_NAMESPACE`
environmental variable, which is detailed in the global [`pgo` Client](/pgo-client/)
reference,

For convenience, we will use the `pgo` namespace in the examples below.
For even more convenience, we recommend setting `pgo` to be the value of
the `PGO_NAMESPACE` variable. In the shell that you will be executing the `pgo`
commands in, run the following command:

```shell
export PGO_NAMESPACE=pgo
```

If you do not wish to set this environmental variable, or are in an environment
where you are unable to use environmental variables, you will have to use the
`--namespace` (or `-n`) flag for most commands, e.g.

`pgo version -n pgo`

### JSON Output

The default for the `pgo` client commands is to output their results in a
readable format. However, there are times where it may be helpful to you to have
the format output in a machine parseable format like JSON.

Several commands support the `-o`/`--output` flags that delivers the results of
the command in the specified output. Presently, the only output that is
supported is `json`.

As an example of using this feature, if you wanted to get the results of the
`pgo test` command in JSON, you could run the following:

```shell
pgo test hacluster -o json
```

## PostgreSQL Operator System Basics

To get started, it's first important to understand the basics of working with
the PostgreSQL Operator itself. You should know how to test if the PostgreSQL
Operator is working, check the overall status of the PostgreSQL Operator, view
the current configuration that the PostgreSQL Operator us using, and seeing
which Kubernetes Namespaces the PostgreSQL Operator has access to.

While this may not be as fun as creating high-availability PostgreSQL clusters,
these commands will help you to perform basic troubleshooting tasks in your
environment.

### Checking Connectivity to the PostgreSQL Operator

A common task when working with the PostgreSQL Operator is to check connectivity
to the PostgreSQL Operator. This can be accomplish with the [`pgo version`](/pgo-client/reference/pgo_version/)
command:

```shell
pgo version
```

which, if working, will yield results similar to:

```
pgo client version {{< param operatorVersion >}}
pgo-apiserver version {{< param operatorVersion >}}
```

### Inspecting the PostgreSQL Operator Configuration

The [`pgo show config`](/pgo-client/reference/pgo_status/) command allows you to
view the current configuration that the PostgreSQL Operator is using. This can
be helpful for troubleshooting issues such as which PostgreSQL images are being
deployed by default, which storage classes are being used, etc.

You can run the `pgo show config` command by running:

```shell
pgo show config
```

which yields output similar to:

```yaml
BasicAuth: ""
Cluster:
  CCPImagePrefix: crunchydata
  CCPImageTag: {{< param centosBase >}}-{{< param postgresVersion >}}-{{< param operatorVersion >}}
  Policies: ""
  Metrics: false
  Badger: false
  Port: "5432"
  PGBadgerPort: "10000"
  ExporterPort: "9187"
  User: testuser
  Database: userdb
  PasswordAgeDays: "60"
  PasswordLength: "8"
  Replicas: "0"
  ServiceType: ClusterIP
  BackrestPort: 2022
  Backrest: true
  BackrestS3Bucket: ""
  BackrestS3Endpoint: ""
  BackrestS3Region: ""
  BackrestS3URIStyle: ""
  BackrestS3VerifyTLS: true
  DisableAutofail: false
  DisableReplicaStartFailReinit: false
  PodAntiAffinity: preferred
  SyncReplication: false
Pgo:
  Audit: false
  PGOImagePrefix: crunchydata
  PGOImageTag: {{< param centosBase >}}-{{< param operatorVersion >}}
PrimaryStorage: nfsstorage
BackupStorage: nfsstorage
ReplicaStorage: nfsstorage
BackrestStorage: nfsstorage
Storage:
  nfsstorage:
    AccessMode: ReadWriteMany
    Size: 1G
    StorageType: create
    StorageClass: ""
    SupplementalGroups: "65534"
    MatchLabels: ""
```

### Viewing PostgreSQL Operator Managed Namespaces

The PostgreSQL Operator has the ability to manage PostgreSQL clusters across
Kubernetes Namespaces. During the course of Operations, it can be helpful to
know which namespaces the PostgreSQL Operator can use for deploying PostgreSQL
clusters.

You can view which namespaces the PostgreSQL Operator can utilize by using
the [`pgo show namespace`](/pgo-client/reference/pgo_show_namespace/) command. To
list out the namespaces that the PostgreSQL Operator has access to, you can run
the following command:

```shell
pgo show namespace --all
```

which yields output similar to:

```
pgo username: admin
namespace                useraccess          installaccess       
default                  accessible          no access           
kube-node-lease          accessible          no access           
kube-public              accessible          no access           
kube-system              accessible          no access           
pgo                      accessible          no access           
pgouser1                 accessible          accessible          
pgouser2                 accessible          accessible          
somethingelse            no access           no access   
```

**NOTE**: Based on your deployment, your Kubernetes administrator may restrict
access to the multi-namespace feature of the PostgreSQL Operator. In this case,
you do not need to worry about managing your namespaces and as such do not need
to use this command, but we recommend setting the `PGO_NAMESPACE` variable as
described in the [general notes](#general-notes) on this page.

## Provisioning: Create, View, Destroy

### Creating a PostgreSQL Cluster

You can create a cluster using the [`pgo create cluster`](/pgo-client/reference/pgo_create_cluster/)
command:

```shell
pgo create cluster hacluster
```

which if successfully, will yield output similar to this:

```
created Pgcluster hacluster
workflow id ae714d12-f5d0-4fa9-910f-21944b41dec8
```

#### Create a PostgreSQL Cluster with Different PVC Sizes

You can also create a PostgreSQL cluster with an arbitrary PVC size using the
[`pgo create cluster`](/pgo-client/reference/pgo_create_cluster/) command. For
example, if you want to create a PostgreSQL cluster with with a 128GB PVC, you
can use the following command:

```shell
pgo create cluster hacluster --pvc-size=128Gi
```

The above command sets the PVC size for all PostgreSQL instances in the cluster,
i.e. the primary and replicas.

This also extends to the size of the pgBackRest repository as well, if you are
using the local Kubernetes cluster storage for your backup repository. To
create a PostgreSQL cluster with a pgBackRest repository that uses a 1TB PVC,
you can use the following command:

```shell
pgo create cluster hacluster --pgbackrest-pvc-size=1Ti
```

#### Specify CPU / Memory for a PostgreSQL Cluster

To specify the amount of CPU and memory to request for a PostgreSQL cluster, you
can use the `--cpu` and `--memory` flags of the
[`pgo create cluster`](/pgo-client/reference/pgo_create_cluster/) command. Both
of these values utilize the [Kubernetes quantity format](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/)
for specifying how to allocate resources.

For example, to create a PostgreSQL cluster that requests 4 CPU cores and has 16
gibibytes of memory, you can use the following command:

```shell
pgo create cluster hacluster --cpu=4 --memory=16Gi
```

#### Create a PostgreSQL Cluster with PostGIS

To create a PostgreSQL cluster that uses the geospatial extension PostGIS, you
can execute the following command, updated with your desired image tag. In the
example below, the cluster will use PostgreSQL {{< param postgresVersion >}} and PostGIS {{< param postgisVersion >}}:

```shell
pgo create cluster hagiscluster \
  --ccp-image=crunchy-postgres-gis-ha \
  --ccp-image-tag={{< param centosBase >}}-{{< param postgresVersion >}}-{{< param postgisVersion >}}-{{< param operatorVersion >}}
```

#### Create a PostgreSQL Cluster with a Tablespace

Tablespaces are a PostgreSQL feature that allows a user to select specific
volumes to store data to, which is helpful in [several types of scenarios](/architecture/tablespaces/).
Often your workload does not require a tablespace, but the PostgreSQL Operator
provides support for tablespaces throughout the lifecycle of a PostgreSQL
cluster.

To create a PostgreSQL cluster that uses the [tablespace](/architecture/tablespaces/)
feature with NFS storage, you can execute the following command:

```shell
pgo create cluster hactsluster --tablespace=name=ts1:storageconfig=nfsstorage
```

You can use your preferred storage engine instead of `nfsstorage`. For example,
to create multiple tablespaces on GKE, you can execute the following command:

```shell
pgo create cluster hactsluster \
    --tablespace=name=ts1:storageconfig=gce \
    --tablespace=name=ts2:storageconfig=gce
```

Tablespaces are immediately available once the PostgreSQL cluster is
provisioned. For example, to create a table using the tablespace `ts1`, you can
run the following SQL on your PostgreSQL cluster:

```sql
CREATE TABLE sensor_data (
  id int GENERATED BY DEFAULT AS IDENTITY PRIMARY KEY,
  sensor1 numeric,
  sensor2 numeric,
  sensor3 numeric,
  sensor4 numeric
)
TABLESPACE ts1;
```

You can also create tablespaces that have different sized PVCs from the ones
defined in the storage specification. For instance, to create two tablespaces,
one that uses a 10GiB PVC and one that uses a 20GiB PVC, you can execute the
following command:

```shell
pgo create cluster hactsluster \
    --tablespace=name=ts1:storageconfig=gce:pvcsize=10Gi \
    --tablespace=name=ts2:storageconfig=gce:pvcsize=20Gi
```

#### Create a PostgreSQL Cluster Using a Backup from Another PostgreSQL Cluster

It is also possible to create a new PostgreSQL Cluster using a backup from another
PostgreSQL cluster.  To do so, simply specify the cluster containing the backup
that you would like to utilize using the `restore-from` option:


```shell
pgo create cluster hacluster2 --restore-from=hacluster1
```

When using this approach, a `pgbackrest restore` will be performed using the pgBackRest
repository for the `restore-from` cluster specified in order to populate the initial
`PGDATA` directory for the new PostgreSQL cluster.  By default, pgBackRest will restore
to the latest backup available and replay all WAL.  However, a `restore-opts` option
is also available that allows the `restore` command to be further customized, e.g. to
perform a point-in-time restore and/or restore from an S3 storage bucket:

```shell
pgo create cluster hacluster2 \
  --restore-from=hacluster1 \
  --restore-opts="--repo-type=s3 --type=time --target='2020-07-02 20:19:36.13557+00'"
```

#### Tracking a Newly Provisioned Cluster

A new PostgreSQL cluster can take a few moments to provision. You may have
noticed that the `pgo create cluster` command returns something called a
"workflow id". This workflow ID allows you to track the progress of your new
PostgreSQL cluster while it is being provisioned using the [`pgo show workflow`](/pgo-client/reference/pgo_show_workflow/)
command:

```shell
pgo show workflow ae714d12-f5d0-4fa9-910f-21944b41dec8
```

which can yield output similar to:

```
parameter           value
---------           -----
pg-cluster          hacluster
task completed      2019-12-27T02:10:14Z
task submitted      2019-12-27T02:09:46Z
workflowid          ae714d12-f5d0-4fa9-910f-21944b41dec8
```

### View PostgreSQL Cluster Details

To see details about your PostgreSQL cluster, you can use the [`pgo show cluster`](/pgo-client/reference/pgo_show_cluster/)
command. These details include elements such as:

- The version of PostgreSQL that the cluster is using
- The PostgreSQL instances that comprise the cluster
- The Pods assigned to the cluster for all of the associated components,
including the nodes that the pods are assigned to
- The Persistent Volume Claims (PVC) that are being consumed by the cluster
- The Kubernetes Deployments associated with the cluster
- The Kubernetes Services associated with the cluster
- The Kubernetes Labels that are assigned to the PostgreSQL instances

and more.

You can view the details of the cluster by executing the following command:

```shell
pgo show cluster hacluster
```

which will yield output similar to:

```
cluster : hacluster (crunchy-postgres-ha:{{< param centosBase >}}-{{< param postgresVersion >}}-{{< param operatorVersion >}})
	pod : hacluster-6dc6cfcfb9-f9knq (Running) on node01 (1/1) (primary)
	pvc : hacluster
	resources : CPU Limit= Memory Limit=, CPU Request= Memory Request=
	storage : Primary=200M Replica=200M
	deployment : hacluster
	deployment : hacluster-backrest-shared-repo
	service : hacluster - ClusterIP (10.102.20.42)
	labels : archive-timeout=60 deployment-name=hacluster pg-cluster=hacluster crunchy-pgha-scope=hacluster pgo-version={{< param operatorVersion >}} current-primary=hacluster name=hacluster pgouser=admin workflowid=ae714d12-f5d0-4fa9-910f-21944b41dec8
```

### Deleting a Cluster

You can delete a PostgreSQL cluster that is managed by the PostgreSQL Operator
by executing the following command:

```shell
pgo delete cluster hacluster
```

This will remove the cluster from being managed by the PostgreSQL Operator, as
well as delete the root data Persistent Volume Claim (PVC) and backup PVCs
associated with the cluster.

If you wish to keep your PostgreSQL data PVC, you can delete the cluster with
the following command:

```shell
pgo delete cluster hacluster --keep-data
```

You can then recreate the PostgreSQL cluster with the same data by using the
`pgo create cluster` command with a cluster of the same name:

```shell
pgo create cluster hacluster
```

This technique is used when performing tasks such as upgrading the PostgreSQL
Operator.

You can also keep the pgBackRest repository associated with the PostgreSQL
cluster by using the `--keep-backups` flag with the `pgo delete cluster`
command:

```shell
pgo delete cluster hacluster --keep-backups
```

## Testing PostgreSQL Cluster Availability

You can test the availability of your cluster by using the [`pgo test`](/pgo-client/reference/pgo_test/)
command. The `pgo test` command checks to see if the Kubernetes Services and
the Pods that comprise the PostgreSQL cluster are available to receive
connections. This includes:

- Testing that the Kubernetes Endpoints are available and able to route requests
to healthy Pods
- Testing that each PostgreSQL instance is available and ready to accept client
connections by performing a connectivity check similar to the one performed by
`pg_isready`

To test the availability of a PostgreSQL cluster, you can run the following
command:

```shell
pgo test hacluster
```

which will yield output similar to:

```
cluster : hacluster
	Services
		primary (10.102.20.42:5432): UP
	Instances
		primary (hacluster-6dc6cfcfb9-f9knq): UP
```

## Disaster Recovery: Backups & Restores

The PostgreSQL Operator supports sophisticated functionality for managing your
backups and restores. For more information for how this works, please see the
[disaster recovery](/architecture/disaster-recovery/) guide.

### Creating a Backup

The PostgreSQL Operator uses the open source [pgBackRest](https://www.pgbackrest.org)
backup and recovery utility for managing backups and PostgreSQL archives. These
backups are also used as part of managing the overall health and
high-availability of PostgreSQL clusters managed by the PostgreSQL Operator and
used as part of the cloning process as well.

When a new PostgreSQL cluster is provisioned by the PostgreSQL Operator, a full
pgBackRest backup is taken by default. This is required in order to create new
replicas (via `pgo scale`) for the PostgreSQL cluster as well as healing during
a [failover scenario](/architecture/high-availability/).

To create a backup, you can run the following command:

```shell
pgo backup hacluster
```

which by default, will create an incremental pgBackRest backup. The reason for
this is that the PostgreSQL Operator initially creates a pgBackRest full backup
when the cluster is initial provisioned, and pgBackRest will take incremental
backups for each subsequent backup until a different backup type is specified.

Most pgBackRest options are supported and can be passed in by the PostgreSQL
Operator via the `--backup-opts` flag. What follows are some examples for how
to utilize pgBackRest with the PostgreSQL Operator to help you create your
optimal disaster recovery setup.

#### Creating a Full Backup

You can create a full backup using the following command:

```shell
pgo backup hacluster --backup-opts="--type=full"
```

#### Creating a Differential Backup

You can create a differential backup using the following command:

```shell
pgo backup hacluster --backup-opts="--type=diff"
```

#### Creating an Incremental Backup

You can create a differential backup using the following command:

```shell
pgo backup hacluster --backup-opts="--type=incr"
```

An incremental backup is created without specifying any options after a full or
differential backup is taken.

### Creating Backups in S3

The PostgreSQL Operator supports creating backups in S3 or any object storage
system that uses the S3 protocol. For more information, please read the section
on [PostgreSQL Operator Backups with S3](/architecture/disaster-recovery/#using-s3)
in the architecture section.

### Displaying Backup Information

You can see information about the current state of backups in a PostgreSQL
cluster managed by the PostgreSQL Operator by executing the following command:

```shell
pgo show backup hacluster
```

### Setting Backup Retention

By default, pgBackRest will allow you to keep on creating backups until you run
out of disk space. As such, it may be helpful to manage how many backups are
retained.

pgBackRest comes with several flags for managing how backups can be retained:

- `--repo1-retention-full`: how many full backups to retain
- `--repo1-retention-diff`: how many differential backups to retain
- `--repo1-retention-archive`: how many sets of WAL archives to retain alongside
the full and differential backups that are retained

For example, to create a full backup and retain the previous 7 full backups, you
would execute the following command:

```shell
pgo backup hacluster --backup-opts="--type=full --repo1-retention-full=7"
```

### Scheduling Backups

Any effective disaster recovery strategy includes having regularly scheduled
backups. The PostgreSQL Operator enables this through its scheduling sidecar
that is deployed alongside the Operator.

#### Creating a Scheduled Backup

For example, to schedule a full backup once a day at midnight, you can execute
the following command:

```shell
pgo create schedule hacluster --schedule="0 1 * * *" \
  --schedule-type=pgbackrest  --pgbackrest-backup-type=full
```

To schedule an incremental backup once every 3 hours, you can execute the
following command:

```shell
pgo create schedule hacluster --schedule="0 */3 * * *" \
  --schedule-type=pgbackrest  --pgbackrest-backup-type=incr
```

You can also create regularly scheduled backups and combine it with a retention
policy. For example, using the above example of taking a nightly full backup,
you can specify a policy of retaining 21 backups by executing the following
command:

```shell
pgo create schedule hacluster --schedule="0 0 * * *" \
  --schedule-type=pgbackrest  --pgbackrest-backup-type=full \
  --schedule-opts="--repo1-retention-full=21"
```

### Restore a Cluster

The PostgreSQL Operator supports the ability to perform a full restore on a
PostgreSQL cluster (i.e. a "clone" or "copy") as well as a
point-in-time-recovery. There are two types of ways to restore a cluster:

- Restore to a new cluster using the `--restore-from` flag in the
[`pgo create cluster`]({{< relref "/pgo-client/reference/pgo_create_cluster.md" >}})
command. This is effectively a [clone](#clone-a-postgresql-cluster) or a copy.
- Restore in-place using the [`pgo restore`]({{< relref "/pgo-client/reference/pgo_restore.md" >}})
command. Note that this is **destructive**.

It is typically better to perform a restore to a new cluster, particularly when
performing a point-in-time-recovery, as it can allow you to more effectively
manage your downtime and avoid making undesired changes to your production data.

Additionally, the "restore to a new cluster" technique works so long as you have
a pgBackRest repository available: the pgBackRest repository does not need to be
attached to an active cluster! For example, if a cluster named `hippo` was
deleted as such:

```
pgo delete cluster hippo --keep-backups
```

you can create a new cluster from the backups like so:

```
pgo create cluster datalake --restore-from=hippo
```

Below provides guidance on how to perform a restore to a new PostgreSQL cluster
both as a full copy and to a specific point in time. Additionally, it also
shows how to restore in place to a specific point in time.

#### Restore to a New Cluster (aka "copy" or "clone")

Restoring to a new PostgreSQL cluster allows one to take a backup and create a
new PostgreSQL cluster that can run alongside an existing PostgreSQL cluster.
There are several scenarios where using this technique is helpful:

- Creating a copy of a PostgreSQL cluster that can be used for other purposes.
Another way of putting this is "creating a clone."
- Restore to a point-in-time and inspect the state of the data without affecting
the current cluster

and more.

##### Full Restore

To create a new PostgreSQL cluster from a backup and restore it fully, you can
execute the following command:

```
pgo create cluster newcluster --restore-from=oldcluster
```

##### Full Restore Across Namespaces

To create a new PostgreSQL cluster from a backup in another namespace and restore it
fully, you can execute the following command:

```
pgo create cluster newcluster --restore-from=oldcluster --restore-from-namespace=oldnamespace
```

##### Point-in-time-Recovery (PITR)

To create a new PostgreSQL cluster and restore it to specific point-in-time
(e.g. before a key table was dropped), you can use the following command,
substituting the time that you wish to restore to:

```
pgo create cluster newcluster \
  --restore-from oldcluster \
  --restore-opts "--type=time --target='2019-12-31 11:59:59.999999+00'"
```

When the restore is complete, the cluster is immediately available for reads and
writes. To inspect the data before allowing connections, add pgBackRest's
`--target-action=pause` option to the `--restore-opts` parameter.

The PostgreSQL Operator supports the full set of pgBackRest restore options,
which can be passed into the `--backup-opts` parameter. For more information,
please review the [pgBackRest restore options](https://pgbackrest.org/command.html#command-restore)

#### Restore in-place

Restoring a PostgreSQL cluster in-place is a **destructive** action that will
perform a recovery on your existing data directory. This is accomplished using
the [`pgo restore`]({{< relref "/pgo-client/reference/pgo_restore.md" >}})
command. The most common scenario is to restore the database to a specific point
in time.

##### Point-in-time-Recovery (PITR)

The more likely scenario when performing a PostgreSQL cluster restore is to
recover to a particular point-in-time (e.g. before a key table was dropped). For
example, to restore a cluster to December 31, 2019 at 11:59pm:

```
pgo restore hacluster --pitr-target="2019-12-31 11:59:59.999999+00" \
  --backup-opts="--type=time"
```

When the restore is complete, the cluster is immediately available for reads and
writes. To inspect the data before allowing connections, add pgBackRest's
`--target-action=pause` option to the `--backup-opts` parameter.

The PostgreSQL Operator supports the full set of pgBackRest restore options,
which can be passed into the `--backup-opts` parameter. For more information,
please review the [pgBackRest restore options](https://pgbackrest.org/command.html#command-restore)

Using this technique, after a restore is complete, you will need to re-enable
high availability on the PostgreSQL cluster manually. You can re-enable high
availability by executing the following command:

```
pgo update cluster hacluster --enable-autofail
```

### Logical Backups (`pg_dump` / `pg_dumpall`)

The PostgreSQL Operator supports taking logical backups with `pg_dump` and
`pg_dumpall`. While they do not provide the same performance and storage
optimizations as the physical backups provided by pgBackRest, logical backups
are helpful when one wants to upgrade between major PostgreSQL versions, or
provide only a subset of a database, such as a table.

#### Create a Logical Backup

To create a logical backup of the 'postgres' database, you can run the following
command:

```shell
pgo backup hacluster --backup-type=pgdump
```

To create a logical backup of a specific database, you can use the `--database` flag,
as in the following command:

```shell
pgo backup hacluster --backup-type=pgdump --database=mydb
```

You can pass in specific options to `--backup-opts`, which can accept most of
the options that the [`pg_dump`](https://www.postgresql.org/docs/current/app-pgdump.html)
command accepts. For example, to only dump the data from a specific table called
`users`:

```shell
pgo backup hacluster --backup-type=pgdump --backup-opts="-t users"
```

To use `pg_dumpall` to create a logical backup of all the data in a PostgreSQL
cluster, you must pass the `--dump-all` flag in `--backup-opts`, i.e.:

```shell
pgo backup hacluster --backup-type=pgdump --backup-opts="--dump-all"
```

#### Viewing Logical Backups

To view an available list of logical backups, you can use the `pgo show backup`
command:

```shell
pgo show backup --backup-type=pgdump
```

This provides information about the PVC that the logical backups are stored on
as well as the timestamps required to perform a restore from a logical backup.

#### Restore from a Logical Backup

To restore from a logical backup, you need to reference the PVC that the logical
backup is stored to, as well as the timestamp that was created by the logical
backup.

You can restore a logical backup using the following command:

```shell
pgo restore hacluster --backup-type=pgdump --backup-pvc=hacluster-pgdump-pvc \
  --pitr-target="2019-01-15-00-03-25" -n pgouser1
```

To restore to a specific database, add the `--pgdump-database` flag to the
command from above:

```shell
pgo restore hacluster --backup-type=pgdump --backup-pvc=hacluster-pgdump-pvc \
  --pgdump-database=mydb --pitr-target="2019-01-15-00-03-25" -n pgouser1
```

## High-Availability: Scaling Up & Down

The PostgreSQL Operator supports a robust [high-availability](/architecture/high-availability)
set up to ensure that your PostgreSQL clusters can stay up and running. For
detailed information on how it works, please see the
[high-availability architecture]((/architecture/high-availability)) section.

### Creating a New Replica

To create a new replica, also known as "scaling up", you can execute the
following command:

```shell
pgo scale hacluster --replica-count=1
```

If you wanted to add two new replicas at the same time, you could execute the
following command:

```shell
pgo scale hacluster --replica-count=2
```

### Viewing Available Replicas

You can view the available replicas in a few ways. First, you can use `pgo show cluster`
to see the overall information about the PostgreSQL cluster:

```shell
pgo show cluster hacluster
```

You can also find specific replica names by using the `--query` flag on the
`pgo failover` and `pgo scaledown` commands, e.g.:

```shell
pgo failover --query hacluster
```

### Manual Failover

The PostgreSQL Operator is set up with an automated failover system based on
distributed consensus, but there may be times where you wish to have your
cluster manually failover. There are two ways to issue a manual failover to
your PostgreSQL cluster:

1. Allow for the PostgreSQL Operator to select the best replica candidate to
failover to
2. Select your own replica candidate to failover to.

To have the PostgreSQL Operator select the best replica candidate for failover,
all you need to do is execute the following command:

```
pgo failover hacluster
```

If you wish to have your cluster manually failover, you must first query your
cluster to determine which failover targets are available. The query command
also provides information that may help your decision, such as replication lag:

```shell
pgo failover hacluster --query
```

Once you have selected the replica that is best for your to failover to, you can
perform a failover with the following command:

```shell
pgo failover hacluster --target=hacluster-abcd
```

where `hacluster-abcd` is the name of the PostgreSQL instance that you want to
promote to become the new primary.

Both methods perform the failover immediately upon execution.

#### Destroying a Replica

To destroy a replica, first query the available replicas by using the `--query`
flag on the `pgo scaledown` command, i.e.:

```shell
pgo scaledown hacluster --query
```

Once you have picked the replica you want to remove, you can remove it by
executing the following command:

```shell
pgo scaledown hacluster --target=hacluster-abcd
```

where `hacluster-abcd` is the name of the PostgreSQL replica that you want to
destroy.

## Monitoring

### PostgreSQL Metrics via pgMonitor

You can view metrics about your PostgreSQL cluster using [PostgreSQL Operator Monitoring]({{< relref "/installation/metrics" >}}),
which uses open source [pgMonitor](https://github.com/CrunchyData/pgmonitor).
First, you need to install the [PostgreSQL Operator Monitoring]({{< relref "/installation/metrics" >}})
stack for your PostgreSQL Operator environment.

After that, you need to ensure that you deploy the `crunchy-postgres-exporter`
with each PostgreSQL cluster that you deploy:

```
pgo create cluster hippo --metrics
```

For more information on how monitoring with the PostgreSQL Operator works,
please see the [Monitoring]({{< relref "/architecture/monitoring.md" >}})
section of the documentation.

### View Disk Utilization

You can see a comparison of Postgres data size versus the Persistent
volume claim size by entering the following:

```shell
pgo df hacluster -n pgouser1
```

## Cluster Maintenance & Resource Management

There are several operations that you can perform to modify a PostgreSQL cluster
over its lifetime.

#### Modify CPU / Memory for a PostgreSQL Cluster

As database workloads change, it may be necessary to modify the CPU and memory
allocation for your PostgreSQL cluster. The PostgreSQL Operator allows for this
via the `--cpu` and `--memory` flags on the [`pgo update cluster`](/pgo-client/reference/pgo_update_cluster/)
command. Similar to the create command, both flags accept values that follow the
[Kubernetes quantity format](https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/).

For example, to update a PostgreSQL cluster to use 8 CPU cores and has 32
gibibytes of memory, you can use the following command:

```shell
pgo update cluster hacluster --cpu=8 --memory=32Gi
```

The resource allocations apply to all instances in a PostgreSQL cluster: this
means your primary and any replicas will have the same cluster resource
allocations. Be sure to specify resource requests that your Kubernetes
environment can support.

**NOTE**: This operation can cause downtime. Modifying the resource requests
allocated to a Deployment requires that the Pods in a Deployment must be
restarted. Each PostgreSQL instance is safely shutdown using the ["fast"](https://www.postgresql.org/docs/current/app-pg-ctl.html)
shutdown method to help ensure it will not enter crash recovery mode when a new
Pod is created.

When the operation completes, each PostgreSQL instance will have the new
resource allocations.

#### Adding a Tablespace to a Cluster

Based on your workload or volume of data, you may wish to add a
[tablespace](https://www.postgresql.org/docs/current/manage-ag-tablespaces.html) to
your PostgreSQL cluster.

You can add a tablespace to an existing PostgreSQL cluster with the
[`pgo update cluster`](/pgo-client/reference/pgo_update_cluster/) command.
Adding a tablespace to a cluster uses a similar syntax to
[creating a cluster with a tablespace](#create-a-postgresql-cluster-with-a-tablespace), for example:

```shell
pgo update cluster hacluster \
    --tablespace=name=tablespace3:storageconfig=storageconfigname
```

**NOTE**: This operation can cause downtime. In order to add a tablespace to a
PostgreSQL cluster, persistent volume claims (PVCs) need to be created and
mounted to each PostgreSQL instance in the cluster. The act of mounting a new
PVC to a Kubernetes Deployment causes the Pods in the deployment to restart.

Each PostgreSQL instance is safely shutdown using the ["fast"](https://www.postgresql.org/docs/current/app-pg-ctl.html)
shutdown method to help ensure it will not enter crash recovery mode when a new
Pod is created.

When the operation completes, the tablespace will be set up and accessible to
use within the PostgreSQL cluster.

For more information on tablespaces, please visit the [tablespace](/architecture/tablespaces/)
section of the documentation.

## Clone a PostgreSQL Cluster

You can create a copy of an existing PostgreSQL cluster in a new PostgreSQL
cluster by using the [`pgo create cluster`]({{< relref "/pgo-client/reference/pgo_create_cluster.md" >}})
command with the `--restore-from` flag (and, if needed, `--restore-opts`).
The command copies the pgBackRest repository from either an active PostgreSQL
cluster, or a pgBackRest repository that exists from a former cluster that was
deleted using `pgo delete cluster --keep-backups`.

You can clone a PostgreSQL cluster by running the following command:

```
pgo create cluster newcluster --restore-from=oldcluster
```

By leveraging `pgo create cluster`, you are able to copy the data from a
PostgreSQL cluster while creating the topology of a new cluster the way you want
to. For instance, if you want to copy data from an existing cluster that does
not have metrics to a new cluster that does, you can accomplish that with the
following command:

```
pgo create cluster newcluster --restore-from=oldcluster --metrics
```

### Clone a PostgreSQL Cluster to Different PVC Size

You can have a cloned PostgreSQL cluster use a different PVC size, which is
useful when moving your PostgreSQL cluster to a larger PVC. For example, to
clone a PostgreSQL cluster to a 256GiB PVC, you can execute the following
command:

```shell
pgo create cluster bighippo --restore-from=hippo  --pvc-size=256Gi
```

You can also have the cloned PostgreSQL cluster use a larger pgBackRest
backup repository by setting its PVC size. For example, to have a cloned
PostgreSQL cluster use a 1TiB pgBackRest repository, you can execute the
following command:

```shell
pgo create cluster bighippo --restore-from=hippo --pgbackrest-pvc-size=1Ti
```

## Enable TLS

TLS allows secure TCP connections to PostgreSQL, and the PostgreSQL Operator
makes it easy to enable this PostgreSQL feature. The TLS support in the
PostgreSQL Operator does not make an opinion about your PKI, but rather loads in
your TLS key pair that you wish to use for the PostgreSQL server as well as its
corresponding certificate authority (CA) certificate. Both of these Secrets are
required to enable TLS support for your PostgreSQL cluster when using the
PostgreSQL Operator, but it in turn allows seamless TLS support.

### Setup

There are three items that are required to enable TLS in your PostgreSQL
clusters:

- A CA certificate
- A TLS private key
- A TLS certificate

There are a variety of methods available to generate these items: in fact,
Kubernetes comes with its own [certificate management system](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/)!
It is up to you to decide how you want to manage this for your cluster. The
PostgreSQL documentation also provides an example for how to
[generate a TLS certificate](https://blog.crunchydata.com/blog/tls-postgres-kubernetes-openssl)
as well.

To set up TLS for your PostgreSQL cluster, you have to create two [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/):
one that contains the CA certificate, and the other that contains the server
TLS key pair.

First, create the Secret that contains your CA certificate. Create the Secret
as a generic Secret, and note that the following requirements **must** be met:

- The Secret must be created in the same Namespace as where you are deploying
your PostgreSQL cluster
- The `name` of the key that is holding the CA **must** be `ca.crt`

There are optional settings for setting up the CA secret:

- You can pass in a certificate revocation list (CRL) for the CA secret by
passing in the CRL using the `ca.crl` key name in the Secret.

For example, to create a CA Secret with the trusted CA to use for the PostgreSQL
clusters, you could execute the following command:

```shell
kubectl create secret generic postgresql-ca --from-file=ca.crt=/path/to/ca.crt
```

To create a CA Secret that includes a CRL, you could execute the following
command:

```shell
kubectl create secret generic postgresql-ca \
  --from-file=ca.crt=/path/to/ca.crt \
  --from-file=ca.crl=/path/to/ca.crl
```

Note that you can reuse this CA Secret for other PostgreSQL clusters deployed by
the PostgreSQL Operator.

Next, create the Secret that contains your TLS key pair. Create the Secret as a
a TLS Secret, and note the following requirement must be met:

- The Secret must be created in the same Namespace as where you are deploying
your PostgreSQL cluster

```shell
kubectl create secret tls hacluster-tls-keypair \
  --cert=/path/to/server.crt \
  --key=/path/to/server.key
```

Now you can create a TLS-enabled PostgreSQL cluster!

### Create a TLS Enabled PostgreSQL Cluster

Using the above example, to create a TLS-enabled PostgreSQL cluster that can
accept both TLS and non-TLS connections, execute the following command:

```shell
pgo create cluster hacluster-tls \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hacluster-tls-keypair
```

Including the `--server-ca-secret` and `--server-tls-secret` flags automatically
enable TLS connections in the PostgreSQL cluster that is deployed. These flags
should reference the CA Secret and the TLS key pair Secret, respectively.

If deployed successfully, when you connect to the PostgreSQL cluster, assuming
your `PGSSLMODE` is set to `prefer` or higher, you will see something like this
in your `psql` terminal:

```
SSL connection (protocol: TLSv1.2, cipher: ECDHE-RSA-AES256-GCM-SHA384, bits: 256, compression: off)
```

### Force TLS in a PostgreSQL Cluster

There are many environments where you want to force all remote connections to
occur over TLS, for example, if you deploy your PostgreSQL cluster's in a public
cloud or on an untrusted network. The PostgreSQL Operator lets you force all
remote connections to occur over TLS by using the `--tls-only` flag.

For example, using the setup above, you can force TLS in a PostgreSQL cluster by
executing the following command:

```shell
pgo create cluster hacluster-tls-only \
  --tls-only \
  --server-ca-secret=postgresql-ca --server-tls-secret=hacluster-tls-keypair
```

If deployed successfully, when you connect to the PostgreSQL cluster, assuming
your `PGSSLMODE` is set to `prefer` or higher, you will see something like this
in your `psql` terminal:

```
SSL connection (protocol: TLSv1.2, cipher: ECDHE-RSA-AES256-GCM-SHA384, bits: 256, compression: off)
```

If you try to connect to a PostgreSQL cluster that is deployed using the
`--tls-only` with TLS disabled (i.e. `PGSSLMODE=disable`), you will receive an
error that connections without TLS are unsupported.

### TLS Authentication for PostgreSQL Replication

PostgreSQL supports [certificate-based authentication](https://www.postgresql.org/docs/current/auth-cert.html),
which allows for PostgreSQL to authenticate users based on the common name (CN)
in a certificate. Using this feature, the PostgreSQL Operator allows you to
configure PostgreSQL replicas in a cluster to authenticate using a certificate
instead of a password.

To use this feature, first you will need to set up a Kubernetes TLS Secret that
has a CN of `primaryuser`. If you do not wish to have this as your CN, you will
need to map the CN of this certificate to the value of `primaryuser` using a
[pg_ident](https://www.postgresql.org/docs/current/auth-username-maps.html)
username map, which you can configure as part of a
[custom PostgreSQL configuration]({{< relref "/advanced/custom-configuration.md" >}}).

You also need to ensure that the certificate is verifiable by the certificate
authority (CA) chain that you have provided for your PostgreSQL cluster. The CA
is provided as part of the `--server-ca-secret` flag in the
[`pgo create cluster`]({{< relref "/pgo-client/reference/pgo_create_cluster.md" >}})
command.

To create a PostgreSQL cluster that uses TLS authentication for replication,
first create Kubernetes Secrets for the server and the CA. For the purposes of
this example, we will use the ones that were created earlier: `postgresql-ca`
and `hacluster-tls-keypair`. After generating a certificate that has a CN of
`primaryuser`, create a Kubernetes Secret that references this TLS keypair
called `hacluster-tls-replication-keypair`:

```
kubectl create secret tls hacluster-tls-replication-keypair \
  --cert=/path/to/replication.crt \
  --key=/path/to/replication.key
```

We can now create a PostgreSQL cluster and allow for it to use TLS
authentication for its replicas! Let's create a PostgreSQL cluster with two
replicas that also requires TLS for any connection:

```
pgo create cluster hippo \
  --tls-only \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hacluster-tls-keypair \
  --replication-tls-secret=hacluster-tls-replication-keypair \
  --replica-count=2
```

By default, the PostgreSQL Operator has each replica connect to PostgreSQL using
a [PostgreSQL TLS mode](https://blog.crunchydata.com/blog/tls-postgres-kubernetes-openssl)
of `verify-ca`. If you wish to perform TLS mutual authentication between
PostgreSQL instances (i.e. certificate-based authentication with SSL mode of
`verify-full`), you will need to create a
[PostgreSQL custom configuration]({{< relref "/advanced/custom-configuration.md" >}}).

### Add TLS to an Existing PostgreSQL Cluster

You can add TLS support to an existing PostgreSQL cluster using the
[`pgo update cluster`]({{< relref "/pgo-client/reference/pgo_update_cluster.md" >}})
command. The following flags are used to manage TLS in a Postgres cluster,
including:

- `--disable-server-tls`: removes TLS from a cluster
- `--disable-tls-only`: removes the TLS-only requirement from a cluster
- `--enable-tls-only`: adds the TLS-only requirement to a cluster
- `--server-ca-secret`: combined with `--server-tls-secret`, enables TLS in a
cluster
- `--server-tls-secret`: combined with `--server-ca-secret`, enables TLS in a
cluster
- `--replication-tls-secret`: enables certificate-based authentication between
Postgres instances.

Using the above examples, to add TLS to a PostgreSQL cluster named `hippo` and
require TLS, you can use the following command:

```
pgo update cluster hippo \
  --enable-tls-only \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hacluster-tls-keypair
```

Conversely, you can disable TLS with the `--disable-tls` flag:

```
pgo update cluster hippo --disable-server-tls
```

## [Custom PostgreSQL Configuration]({{< relref "/advanced/custom-configuration.md" >}})

Customizing PostgreSQL configuration is currently not subject to the `pgo`
client, but given it is a common question, we thought it may be helpful to link
to how to do it from here. To find out more about how to
[customize your PostgreSQL configuration]({{< relref "/advanced/custom-configuration.md" >}}),
please refer to the [Custom PostgreSQL Configuration]({{< relref "/advanced/custom-configuration.md" >}})
section of the documentation.

## pgAdmin 4: PostgreSQL Administration

[pgAdmin 4](https://www.pgadmin.org/) is a popular graphical user interface that
lets you work with PostgreSQL databases from both a desktop or web-based client.
In the case of the PostgreSQL Operator, the pgAdmin 4 web client can be deployed
and synchronized with PostgreSQL clusters so that users can administrate their
databases with their PostgreSQL username and password.

For example, let's work with a PostgreSQL cluster called `hippo` that has a user named `hippo` with password `datalake`, e.g.:

```
pgo create cluster hippo --username=hippo --password=datalake
```

Once the `hippo` PostgreSQL cluster is ready, create the pgAdmin 4 deployment
with the [`pgo create pgadmin`]({{< relref "/pgo-client/reference/pgo_create_pgadmin.md" >}})
command:

```
pgo create pgadmin hippo
```

This creates a pgAdmin 4 deployment unique to this PostgreSQL cluster and
synchronizes the PostgreSQL user information into it. To access pgAdmin 4, you
can set up a port-forward to the Service, which follows the
pattern `<clusterName>-pgadmin`, to port `5050`:

```
kubectl port-forward svc/hippo-pgadmin 5050:5050
```

Point your browser at `http://localhost:5050` and use your database username
(e.g. `hippo`) and password (e.g. `datalake`) to log in.

![pgAdmin 4 Login Page](/images/pgadmin4-login.png)

(Note: if your password does not appear to work, you can retry setting up the
user with the [`pgo update user`]({{< relref "/pgo-client/reference/pgo_update_user.md" >}})
command: `pgo update user hippo --password=datalake`)

The `pgo create user`, `pgo update user`, and `pgo delete user` commands are
synchronized with the pgAdmin 4 deployment. Any user with credentials to this
PostgreSQL cluster will be able to log in and use pgAdmin 4:

![pgAdmin 4 Query](/images/pgadmin4-query.png)

You can remove the pgAdmin 4 deployment with the [`pgo delete pgadmin`]({{< relref "/pgo-client/reference/pgo_delete_pgadmin.md" >}})
command.

For more information, please read the [pgAdmin 4 Architecture]({{< relref "/architecture/pgadmin4.md" >}})
section of the documentation.

## Standby Clusters: Multi-Cluster Kubernetes Deployments

A [standby PostgreSQL cluster]({{< relref "/architecture/high-availability/multi-cluster-kubernetes.md" >}})
can be used to create an advanced high-availability set with a PostgreSQL
cluster running in a different Kubernetes cluster, or used for other operations
such as migrating from one PostgreSQL cluster to another. Note: this is not
[high availability]({{< relref "/architecture/high-availability/_index.md" >}})
per se: a high-availability PostgreSQL cluster will automatically fail over upon
a downtime event, whereas a standby PostgreSQL cluster must be explicitly
promoted.

With that said, you can run multiple PostgreSQL Operators in different
Kubernetes clusters, and the below functionality will work!

Below are some commands for setting up and using standby PostgreSQL clusters.
For more details on how standby clusters work, please review the section on
[Kubernetes Multi-Cluster Deployments]({{< relref "/architecture/high-availability/multi-cluster-kubernetes.md" >}}).

### Creating a Standby Cluster

Before creating a standby cluster, you will need to ensure that your primary
cluster is created properly. Standby clusters require the use of S3 or
equivalent S3-compatible storage system that is accessible to both the primary
and standby clusters. For example, to create a primary cluster to these
specifications:

```shell
pgo create cluster hippo --pgbouncer --replica-count=2 \
  --pgbackrest-storage-type=posix,s3 \
  --pgbackrest-s3-key=<redacted> \
  --pgbackrest-s3-key-secret=<redacted> \
  --pgbackrest-s3-bucket=watering-hole \
  --pgbackrest-s3-endpoint=s3.amazonaws.com \
  --pgbackrest-s3-region=us-east-1 \
  --pgbackrest-s3-uri-style=host \
  --pgbackrest-s3-verify-tls=true \
  --password-superuser=supersecrethippo \
  --password-replication=somewhatsecrethippo \
  --password=opensourcehippo
  ```

Before setting up the standby PostgreSQL cluster, you will need to wait a few
moments for the primary PostgreSQL cluster to be ready. Once your primary
PostgreSQL cluster is available, you can create a standby cluster by using the
following command:

```shell
pgo create cluster hippo-standby --standby --replica-count=2 \
  --pgbackrest-storage-type=s3 \
  --pgbackrest-s3-key=<redacted> \
  --pgbackrest-s3-key-secret=<redacted> \
  --pgbackrest-s3-bucket=watering-hole \
  --pgbackrest-s3-endpoint=s3.amazonaws.com \
  --pgbackrest-s3-region=us-east-1 \
  --pgbackrest-s3-uri-style=host \
  --pgbackrest-s3-verify-tls=true \
  --pgbackrest-repo-path=/backrestrepo/hippo-backrest-shared-repo \
  --password-superuser=supersecrethippo \
  --password-replication=somewhatsecrethippo \
  --password=opensourcehippo
```

If you are unsure of your user credentials form the original `hippo` cluster,
you can retrieve them using the [`pgo show user`]({{< relref "/pgo-client/reference/pgo_show_user.md" >}})
command with the `--show-system-accounts` flag:

```
pgo show user hippo --show-system-accounts
```

The standby cluster will take a few moments to bootstrap, but it is now set up!

### Promoting a Standby Cluster

Before promoting a standby cluster, it is first necessary to shut down the
primary cluster, otherwise you can run into a potential "[split-brain](https://en.wikipedia.org/wiki/Split-brain_(computing))"
scenario (if your primary Kubernetes cluster is down, it may not be possible to
do this).

To shutdown, run the following command:

```
pgo update cluster hippo --shutdown
```

Once it is shut down, you can promote the standby cluster:

```
pgo update cluster hippo-standby --promote-standby
```

The standby is now an active PostgreSQL cluster and can start to accept writes.

To convert the previous active cluster into a standby cluster, you can run the
following command:

```
pgo update cluster hippo --enable-standby
```

This will take a few moments to make this PostgreSQL cluster into a standby
cluster. When it is ready, you can start it up with the following command:

```
pgo update cluster hippo --startup
```

## Labels

Labels are a helpful way to organize PostgreSQL clusters, such as by application
type or environment. The PostgreSQL Operator supports managing Kubernetes Labels
as a convenient way to group PostgreSQL clusters together.

You can view which labels are assigned to a PostgreSQL cluster using the
[`pgo show cluster`](/pgo-client/reference/pgo_show_cluster/) command. You are also
able to see these labels when using `kubectl` or `oc`.

### Add a Label to a PostgreSQL Cluster

Labels can be added to PostgreSQL clusters using the [`pgo label`](/pgo-client/reference/pgo_label/)
command. For example, to add a label with a key/value pair of `env=production`,
you could execute the following command:

```shell
pgo label hacluster --label=env=production
```

### Add a Label to Multiple PostgreSQL Clusters

You can add also add a label to multiple PostgreSQL clusters simultaneously
using the `--selector` flag on the `pgo label` command. For example, to add a
label with a key/value pair of `env=production` to clusters that have a label
key/value pair of `app=payment`, you could execute the following command:

```shell
pgo label --selector=app=payment --label=env=production
```

## Custom Annotations

There are a variety of reasons why one may want to add additional
[Annotations](https://kubernetes.io/docs/concepts/overview/working-with-objects/annotations/)
to the Deployments, and by extension Pods, managed by the PostgreSQL Operator:

- External applications that extend functionality via details in an annotation
- Tracking purposes for an internal application

etc.

As such the `pgo` client allows you to manage your own custom annotations on the
Operator. There are four different ways to add annotations:

- On PostgreSQL instances
- On pgBackRest instances
- On pgBouncer instances
- On all of the above

The custom annotation feature follows the same syntax as Kubernetes for adding
and removing annotations, e.g.:

`--annotation=name=value`

would add an annotation called `name` with a value of `value`, and:

`--annotation=name-`

would remove an annotation called `name`

### Adding an Annotation

There are two ways to add an Annotation during the lifecycle of a PostgreSQL
cluster:

- Cluster creation: ([`pgo create cluster`]({{< relref "/pgo-client/reference/pgo_create_cluster.md" >}}))
- Updating a cluster: ([`pgo update cluster`]({{< relref "/pgo-client/reference/pgo_update_cluster.md" >}}))

There are several flags available for managing Annotations, i.e.:

- `--annotation`: adds an Annotation to all managed Deployments (PostgreSQL, pgBackRest, pgBouncer)
- `--annotation-postgres`: adds an Annotation only to PostgreSQL Deployments
- `--annotation-pgbackrest`: adds an Annotation only to pgBackrest Deployments
- `--annotation-pgbouncer`: adds an Annotation only to pgBouncer Deployments

To add an Annotation with key `hippo` and value `awesome` to all of the managed
Deployments when creating a cluster, you would run the following command:

`pgo create cluster hippo --annotation=hippo=awesome`

To add an Annotation with key `elephant` and value `cool` to only the PostgreSQL
Deployments when creating a cluster, you would run the following command:

`pgo create cluster hippo --annotation-postgres=elephant=cool`

To add an Annotation to all the managed Deployments in an existing cluster, you
can use the `pgo update cluster` command:

`pgo update cluster hippo --annotation=zebra=nice`

### Adding Multiple Annotations

There are two syntaxes you could use to add multiple Annotations to a cluster:

`pgo create cluster hippo --annotation=hippo=awesome,elephant=cool`

or

`pgo create cluster hippo --annotation=hippo=awesome --annotation=elephant=cool`

### Updating Annotations

To update an Annotation, you can use the [`pgo update cluster`]({{< relref "/pgo-client/reference/pgo_update_cluster.md" >}})
command and reference the original Annotation key. For intance, if I wanted to
update the `hippo` annotation to be `rad`:

`pgo update cluster hippo --annotation=hippo=rad`

### Removing Annotations

To remove an Annotation, you need to add a `-` to the end of the Annotation
name. For example, to remove the `hippo` annotation:

`pgo update cluster hippo --annotation=hippo-`

## Policy Management

### Create a Policy

To create a SQL policy, enter the following:

    pgo create policy mypolicy --in-file=mypolicy.sql -n pgouser1

This examples creates a policy named *mypolicy* using the contents
of the file *mypolicy.sql* which is assumed to be in the current
directory.

You can view policies as following:

    pgo show policy --all -n pgouser1


### Apply a Policy

    pgo apply mypolicy --selector=environment=prod
    pgo apply mypolicy --selector=name=hacluster

## Advanced Operations

### Connection Pooling via pgBouncer

Please see the [tutorial on pgBouncer]({{< relref "tutorial/pgbouncer.md" >}}).

### Query Analysis via pgBadger

You can create a pgbadger sidecar container in your Postgres cluster
pod as follows:

    pgo create cluster hacluster --pgbadger -n pgouser1

### Create a Cluster using Specific Storage

    pgo create cluster hacluster --storage-config=somestorageconfig -n pgouser1

Likewise, you can specify a storage configuration when creating
a replica:

    pgo scale hacluster --storage-config=someslowerstorage -n pgouser1

This example specifies the *somestorageconfig* storage configuration
to be used by the Postgres cluster.  This lets you specify a storage
configuration that is defined in the *pgo.yaml* file specifically for
a given Postgres cluster.

You can create a Cluster using a Preferred Node as follows:

    pgo create cluster hacluster --node-label=speed=superfast -n pgouser1

That command will cause a node affinity rule to be added to the
Postgres pod which will influence the node upon which Kubernetes
will schedule the Pod.

Likewise, you can create a Replica using a Preferred Node as follows:

    pgo scale hacluster --node-label=speed=slowerthannormal -n pgouser1

### Create a Cluster with LoadBalancer ServiceType

    pgo create cluster hacluster --service-type=LoadBalancer -n pgouser1

This command will cause the Postgres Service to be of a specific
type instead of the default ClusterIP service type.

### Namespace Operations

Create an Operator namespace where Postgres clusters can be created
and managed by the Operator:

    pgo create namespace mynamespace

Update a Namespace to be able to be used by the Operator:

    pgo update namespace somenamespace

Delete a Namespace:

    pgo delete namespace mynamespace

### PostgreSQL Operator User Operations

PGO users are users defined for authenticating to the PGO REST API.  You
can manage those users with the following commands:

    pgo create pgouser someuser --pgouser-namespaces="pgouser1,pgouser2" --pgouser-password="somepassword" --pgouser-roles="pgoadmin"
    pgo create pgouser otheruser --all-namespaces --pgouser-password="somepassword" --pgouser-roles="pgoadmin"

Update a user:

    pgo update pgouser someuser --pgouser-namespaces="pgouser1,pgouser2" --pgouser-password="somepassword" --pgouser-roles="pgoadmin"
    pgo update pgouser otheruser --all-namespaces --pgouser-password="somepassword" --pgouser-roles="pgoadmin"

Delete a PGO user:

    pgo delete pgouser someuser

PGO roles are also managed as follows:

    pgo create pgorole somerole --permissions="Cat,Ls"

Delete a PGO role with:

    pgo delete pgorole somerole

Update a PGO role with:

    pgo update pgorole somerole --permissions="Cat,Ls"

### PostgreSQL Cluster User Operations

Managed Postgres users can be viewed using the following command:

    pgo show user hacluster

Postgres users can be created using the following command examples:

    pgo create user hacluster --username=somepguser --password=somepassword --managed
    pgo create user --selector=name=hacluster --username=somepguser --password=somepassword --managed

Those commands are identical in function, and create on the hacluster Postgres cluster, a user named *somepguser*, with a password of *somepassword*, the account is *managed* meaning that
these credentials are stored as a Secret on the Kubernetes cluster in the Operator
namespace.

Postgres users can be deleted using the following command:

    pgo delete user hacluster --username=somepguser

That command deletes the user on the hacluster Postgres cluster.

Postgres users can be updated using the following command:

    pgo update user hacluster --username=somepguser --password=frodo

That command changes the password for the user on the hacluster Postgres cluster.
