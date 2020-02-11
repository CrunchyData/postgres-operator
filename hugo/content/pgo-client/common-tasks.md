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
guide, you will have two namespaces installed: `pgouser1` and `pgouser2`. We
can choose to always use one of these namespaces by setting the `PGO_NAMESPACE`
environmental variable, which is detailed in the global [`pgo` Client](/pgo-client/)
reference,

For convenience, we will use the `pgouser1` namespace in the examples below.
For even more convenience, we recommend setting `pgouser1` to be the value of
the `PGO_NAMESPACE` variable. In the shell that you will be executing the `pgo`
commands in, run the following command:

```shell
export PGO_NAMESPACE=pgouser1
```

If you do not wish to set this environmental variable, or are in an environment
where you are unable to use environmental variables, you will have to use the
`--namespace` (or `-n`) flag for most commands, e.g.

`pgo version -n pgouser1`

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
pgo client version 4.3.0
pgo-apiserver version 4.3.0
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
  CCPImageTag: centos7-12.2-4.3.0
  PrimaryNodeLabel: ""
  ReplicaNodeLabel: ""
  Policies: ""
  LogStatement: none
  LogMinDurationStatement: "60000"
  Metrics: false
  Badger: false
  Port: "5432"
  PGBadgerPort: "10000"
  ExporterPort: "9187"
  User: testuser
  ArchiveTimeout: "60"
  Database: userdb
  PasswordAgeDays: "60"
  PasswordLength: "8"
  Strategy: "1"
  Replicas: "0"
  ServiceType: ClusterIP
  BackrestPort: 2022
  Backrest: true
  BackrestS3Bucket: ""
  BackrestS3Endpoint: ""
  BackrestS3Region: ""
  DisableAutofail: false
  PgmonitorPassword: ""
  EnableCrunchyadm: false
  DisableReplicaStartFailReinit: false
  PodAntiAffinity: preferred
  SyncReplication: false
Pgo:
  PreferredFailoverNode: ""
  Audit: false
  PGOImagePrefix: crunchydata
  PGOImageTag: centos7-4.3.0
ContainerResources:
  large:
    RequestsMemory: 2Gi
    RequestsCPU: "2.0"
    LimitsMemory: 2Gi
    LimitsCPU: "4.0"
  small:
    RequestsMemory: 256Mi
    RequestsCPU: "0.1"
    LimitsMemory: 256Mi
    LimitsCPU: "0.1"
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
    Fsgroup: ""
    SupplementalGroups: "65534"
    MatchLabels: ""
DefaultContainerResources: ""
DefaultLoadResources: ""
DefaultRmdataResources: ""
DefaultBackupResources: ""
DefaultBadgerResources: ""
DefaultPgbouncerResources: ""
```

### Viewing PostgreSQL Operator Key Metrics

The [`pgo status`](/pgo-client/reference/pgo_status/) command provides a
generalized statistical view of the overall resource consumption of the
PostgreSQL Operator. These stats include:

- The total number of PostgreSQL instances
- The total number of Persistent Volume Claims (PVC) that are allocated, along with the total amount of disk the claims specify
- The types of container images that are deployed, along with how many are deployed
- The nodes that are used by the PostgreSQL Operator

and more

You can use the `pgo status` command by running:

```shell
pgo status
```

which yields output similar to:

```
Operator Start:          2019-12-26 17:53:45 +0000 UTC
Databases:               8
Claims:                  8
Total Volume Size:       8Gi       

Database Images:
                         4	crunchydata/crunchy-postgres-ha:centos7-12.2-4.3.0
                         4	crunchydata/pgo-backrest-repo:centos7-4.3.0
                         8	crunchydata/pgo-backrest:centos7-4.3.0

Databases Not Ready:

Nodes:
	master                        
		Status:Ready                         
		Labels:
			beta.kubernetes.io/arch=amd64
			beta.kubernetes.io/os=linux
			kubernetes.io/arch=amd64
			kubernetes.io/hostname=master
			kubernetes.io/os=linux
			node-role.kubernetes.io/master=
	node01                        
		Status:Ready                         
		Labels:
			beta.kubernetes.io/arch=amd64
			beta.kubernetes.io/os=linux
			kubernetes.io/arch=amd64
			kubernetes.io/hostname=node01
			kubernetes.io/os=linux

Labels (count > 1): [count] [label]
	[8]	[vendor=crunchydata]
	[4]	[pgo-backrest-repo=true]
	[4]	[pgouser=pgoadmin]
	[4]	[pgo-pg-database=true]
	[4]	[crunchy_collect=false]
	[4]	[pg-pod-anti-affinity=]
	[4]	[pgo-version=4.3.0]
	[4]	[archive-timeout=60]
	[2]	[pg-cluster=hacluster]
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
pgo username: pgoadmin
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

#### Create a PostgreSQL Cluster with PostGIS

To create a PostgreSQL cluster that uses the geospatial extension PostGIS, you
can execute the following command:

```shell
pgo create cluster hagiscluster --ccp-image=crunchy-postgres-gis-ha
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
pgo create cluster hactsluster --tablespaces=ts1=nfsstorage
```

You can use your preferred storage engine instead of `nfsstorage`. For example,
to create multiple tablespaces on GKE, you can execute the following command:

```shell
pgo create cluster hactsluster --tablespaces=ts1=gce,ts2=gce
```

Tablespaces are immediately available once the PostgreSQL cluster is
provisioned. For example, to create a table using the tablespace.

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
cluster : hacluster (crunchy-postgres-ha:centos7-12.2-4.3.0)
	pod : hacluster-6dc6cfcfb9-f9knq (Running) on node01 (1/1) (primary)
	pvc : hacluster
	resources : CPU Limit= Memory Limit=, CPU Request= Memory Request=
	storage : Primary=200M Replica=200M
	deployment : hacluster
	deployment : hacluster-backrest-shared-repo
	service : hacluster - ClusterIP (10.102.20.42)
	labels : pg-pod-anti-affinity= archive-timeout=60 crunchy-pgbadger=false crunchy_collect=false deployment-name=hacluster pg-cluster=hacluster crunchy-pgha-scope=hacluster autofail=true pgo-backrest=true pgo-version=4.3.0 current-primary=hacluster name=hacluster pgouser=pgoadmin workflowid=ae714d12-f5d0-4fa9-910f-21944b41dec8
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
PostgreSQL cluster as well as a point-in-time-recovery using the `pgo restore`
command. Note that both of these options are **destructive** to the existing
PostgreSQL cluster; to "restore" the PostgreSQL cluster to a new deployment,
please see the [clone](#clone-a-postgresql-cluster) section.

After a restore, there are some cleanup steps you will need to perform. Please
review the [Post Restore Cleanup](#post-restore-cleanup) section.

#### Full Restore

To perform a full restore of a PostgreSQL cluster, you can execute the following
command:

```shell
pgo restore hacluster
```

If you want your PostgreSQL cluster to be restored to a specific node, you can
execute the following command:

```shell
pgo restore hacluster --node-label=failure-domain.beta.kubernetes.io/zone=us-central1-a
```

There are very few reasons why you will want to execute a full restore. If you
want to make a copy of your PostgreSQL cluster, please use
[`pgo clone`](/pgo-client/reference/pgo_clone).

#### Point-in-time-Recovery (PITR)

The more likely scenario when performing a PostgreSQL cluster restore is to
recover to a particular point-in-time (e.g. before a key table was dropped). For
example, to restore a cluster to December 23, 2019 at 8:00am:

```shell
pgo restore hacluster --pitr-target="2019-12-23 08:00:00.000000+00" \
  --backup-opts="--type=time"
```

The PostgreSQL Operator supports the full set of pgBackRest restore options,
which can be passed into the `--backup-opts` parameter. For more information,
please review the [pgBackRest restore options](https://pgbackrest.org/command.html#command-restore)

#### Post Restore Cleanup

After a restore is complete, you will need to re-enable high-availability on a
PostgreSQL cluster manually. You can re-enable high-availability by executing
the following command:

```shell
pgo update cluster hacluster --autofail=true
```

### Logical Backups (`pg_dump` / `pg_dumpall`)

The PostgreSQL Operator supports taking logical backups with `pg_dump` and
`pg_dumpall`. While they do not provide the same performance and storage
optimizations as the physical backups provided by pgBackRest, logical backups
are helpful when one wants to upgrade between major PostgreSQL versions, or
provide only a subset of a database, such as a table.

#### Create a Logical Backup

To create a logical backup of a full database, you can run the following
command:

```shell
pgo backup hacluster --backup-type=pgdump
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
cluster manually failover. If you wish to have your cluster manually failover,
first, query your cluster to determine which failover targets are available.
The query command also provides information that may help your decision, such as
replication lag:

```shell
pgo failover --query hacluster
```

Once you have selected the replica that is best for your to failover to, you can
perform a failover with the following command:

```shell
pgo failover hacluster --target=hacluster-abcd
```

where `hacluster-abcd` is the name of the PostgreSQL instance that you want to
promote to become the new primary

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

## Clone a PostgreSQL Cluster

You can create a copy of an existing PostgreSQL cluster in a new PostgreSQL
cluster by using the [`pgo clone`](/pgo-client/reference/pgo_clone/) command. To
create a new copy of a PostgreSQL cluster, you can execute the following
command:

```shell
pgo clone hacluster newhacluster
```

## Monitoring

### View Disk Utilization

You can see a comparison of Postgres data size versus the Persistent
volume claim size by entering the following:

```shell
pgo df hacluster -n pgouser1
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

To add a pgbouncer Deployment to your Postgres cluster, enter:

    pgo create cluster hacluster --pgbouncer -n pgouser1

You can add pgbouncer after a Postgres cluster is created as follows:

    pgo create pgbouncer hacluster
    pgo create pgbouncer --selector=name=hacluster

You can also specify a pgbouncer password as follows:

    pgo create cluster hacluster --pgbouncer --pgbouncer-pass=somepass -n pgouser1

Note, the pgbouncer configuration defaults to specifying only
a single entry for the primary database.  If you want it to
have an entry for the replica service, add the following
configuration to pgbouncer.ini:

    {{.PG_REPLICA_SERVICE_NAME}} = host={{.PG_REPLICA_SERVICE_NAME}} port={{.PG_PORT}} auth_user={{.PG_USERNAME}} dbname={{.PG_DATABASE}}

You can remove a pgbouncer from a cluster as follows:

    pgo delete pgbouncer hacluster -n pgouser1

You can create a pgbadger sidecar container in your Postgres cluster
pod as follows:

    pgo create cluster hacluster --pgbadger -n pgouser1

Likewise, you can add the Crunchy Collect Metrics sidecar container
into your Postgres cluster pod as follows:

    pgo create cluster hacluster --metrics -n pgouser1

Note: backend metric storage such as Prometheus and front end
visualization software such as Grafana are not created automatically
by the PostgreSQL Operator.  For instructions on installing Grafana and
Prometheus in your environment, see the [Crunchy Container Suite documentation](https://access.crunchydata.com/documentation/crunchy-containers/4.3.0/examples/metrics/metrics/).

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
