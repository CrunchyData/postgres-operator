---
title: "Operator CLI Overview"
date:
draft: false
weight: 6
---

The command line tool, pgo, is used to interact with the Postgres Operator.

Most users will work with the Operator using the *pgo* CLI tool.  That tool is downloaded from the GitHub Releases page for the Operator (https://github.com/crunchydata/postgres-operator/releases).

The *pgo* client is provided in Mac, Windows, and Linux binary formats, download the appropriate client to your local laptop or workstation to work with a remote Operator.

## Syntax
Use the following syntax to run  `pgo`  commands from your terminal window:

    pgo [command] ([TYPE] [NAME]) [flags]

Where *command* is a verb like:

 * show
 * create
 * delete

And *type* is a resource type like:

 * cluster
 * policy
 * user

And *name* is the name of the resource type like:

 * mycluster
 * somesqlpolicy
 * john

To get detailed help information and command flag descriptions on each *pgo* command, enter:

    pgo [command] -h
 
## Operations

The following table shows the *pgo* operations currently implemented:

| Operation   |      Syntax      |  Description |
|:----------|:-------------|:------|
| apply |pgo apply mypolicy  --selector=name=mycluster  | Apply a SQL policy on a Postgres cluster(s) that have a label matching service-name=mycluster|
| backup |pgo backup mycluster  |Perform a backup on a Postgres cluster(s) |
| create |pgo create cluster mycluster  |Create an Operator resource type (e.g. cluster, policy, schedule, user) |
| delete |pgo delete cluster mycluster  |Delete an Operator resource type (e.g. cluster, policy, user, schedule) |
| ls |pgo ls mycluster  |Perform a Linux *ls* command on the cluster. |
| cat |pgo cat mycluster  |Perform a Linux *ls* command on the cluster. |
| df |pgo df mycluster  |Display the disk status/capacity of a Postgres cluster. |
| failover |pgo failover mycluster  |Perform a manual failover of a Postgres cluster. |
| help |pgo help |Display general *pgo* help information. |
| label |pgo label mycluster --label=environment=prod  |Create a metadata label for a Postgres cluster(s). |
| load |pgo load --load-config=load.json --selector=name=mycluster  |Perform a data load into a Postgres cluster(s).|
| reload |pgo reload mycluster  |Perform a pg_ctl reload command on a Postgres cluster(s). |
| restore |pgo restore mycluster |Perform a pgbackrest or pgdump restore on a Postgres cluster. |
| scale |pgo scale mycluster  |Create a Postgres replica(s) for a given Postgres cluster. |
| scaledown |pgo scaledown mycluster --query  |Delete a replica from a Postgres cluster. |
| show |pgo show cluster mycluster  |Display Operator resource information (e.g. cluster, user, policy, schedule). |
| status |pgo status  |Display Operator status. |
| test |pgo test mycluster  |Perform a SQL test on a Postgres cluster(s). |
| update |pgo update cluster --label=autofail=false  |Update a Postgres cluster(s). |
| upgrade |pgo upgrade mycluster  |Perform a minor upgrade to a Postgres cluster(s). |
| user |pgo user --selector=name=mycluster --update-passwords  |Perform Postgres user maintenance on a Postgres cluster(s). |
| version |pgo version  |Display Operator version information. |

## Common Operations

In all the examples below, the user is specifying the *pgouser1* namespace
as the target of the operator.  Replace this value with your own namespace
value.  You can specify a default namespace to be used by setting the
PGO_NAMESPACE environment variable on the *pgo* client environment.

### Cluster Operations

A user will typically start using the Operator by creating a Postgres
cluster as follows:

    pgo create cluster mycluster -n pgouser1

This command creates a Postgres cluster in the *pgouser1* namespace 
that has a single Postgres primary container. 

You can see the Postgres cluster using the following:

    pgo show cluster mycluster -n pgouser1

You can test the Postgres cluster by entering:

    pgo test mycluster -n pgouser1

You can optionally add a Postgres replica to your Postgres
cluster as follows:

    pgo scale mycluster -n pgouser1

You can create a Postgres cluster initially with a Postgres replica as follows:

    pgo create cluster mycluster --replica-count=1 -n pgouser1

To view the Postgres logs, you can enter commands such as:

    pgo ls mycluster -n pgouser1 /pgdata/mycluster/pg_log 
    pgo cat mycluster -n pgouser1 /pgdata/mycluster/pg_log/postgresql-Mon.log | tail -3


#### Backups

By default the Operator deploys pgbackrest for a Postgres cluster to
hold database backup data.  

You can create a pgbackrest backup job as follows:

    pgo backup mycluster -n pgouser1

You can perform a pgbasebackup job as follows:

    pgo backup mycluster --backup-type=pgbasebackup -n pgouser1

You can optionally pass pgbackrest command options into the backup
command as follows:

    pgo backup mycluster --backup-type=pgbackrest --backup-opts="--type=diff" -n pgouser1

See pgbackrest.org for command flag descriptions.

You can create a Postgres cluster that does not include pgbackrest 
if you specify the following:

    pgo create cluster mycluster --pgbackrest=false -n pgouser1

#### Scaledown a Cluster

You can remove a Postgres replica using the following:

    pgo scaledown mycluster --query -n pgouser1
    pgo scaledown mycluster --target=sometarget -n pgouser1

#### Delete a Cluster

You can remove a Postgres cluster by entering:

    pgo delete cluster mycluster -n pgouser1

#### Delete a Cluster and Its Persistent Volume Claims

You can remove the persistent volumes when removing a Postgres cluster
by specifying the following command flag:

    pgo delete cluster mycluster --delete-data -n pgouser1

#### View Disk Utilization

You can see a comparison of Postgres data size versus the Persistent
volume claim size by entering the following:

    pgo df mycluster -n pgouser1

### Label Operations
#### Apply a Label to a Cluster

You can apply a Kubernetes label to a Postgres cluster as follows:

    pgo label mycluster --label=environment=prod -n pgouser1

In this example, the label key is *environment* and the label
value is *prod*.

You can apply labels across a category of Postgres clusters by
using the *--selector* command flag as follows:

    pgo label --selector=clustertypes=research --label=environment=prod -n pgouser1

In this example, any Postgres cluster with the label of *clustertypes=research*
will have the label *environment=prod* set.

In the following command, you can also view Postgres clusters by
using the *--selector* command flag which specifies a label key value
to search with:

    pgo show cluster --selector=environment=prod -n pgouser1

### Policy Operations
#### Create a Policy

To create a SQL policy, enter the following:

    pgo create policy mypolicy --in-file=mypolicy.sql -n pgouser1

This examples creates a policy named *mypolicy* using the contents
of the file *mypolicy.sql* which is assumed to be in the current
directory.

You can view policies as following:

    pgo show policy --all -n pgouser1


#### Apply a Policy

    pgo apply mypolicy --selector=environment=prod
    pgo apply mypolicy --selector=name=mycluster

### Operator Status
#### Show Operator Version

To see what version of the Operator client and server you are using, enter:

    pgo version

To see the Operator server status, enter:

    pgo status -n pgouser1

To see the Operator server configuration, enter:

    pgo show config -n pgouser1

To see what namespaces exist and if you have access to them, enter:

    pgo show namespace -n pgouser1

#### Perform a pgdump backup

	pgo backup mycluster --backup-type=pgdump -n pgouser1
	pgo backup mycluster --backup-type=pgdump --backup-opts="--dump-all --verbose" -n pgouser1
	pgo backup mycluster --backup-type=pgdump --backup-opts="--schema=myschema" -n pgouser1

Note: To run pgdump_all instead of pgdump, pass '--dump-all' flag in --backup-opts as shown above. All --backup-opts should be space delimited.

#### Perform a pgbackrest restore

    pgo restore mycluster -n pgouser1

Or perform a restore based on a point in time:

    pgo restore mycluster --pitr-target="2019-01-14 00:02:14.921404+00" --backup-opts="--type=time" -n pgouser1

You can also set the any of the [pgbackrest restore options](https://pgbackrest.org/command.html#command-restore) :

    pgo restore mycluster --pitr-target="2019-01-14 00:02:14.921404+00" --backup-opts=" see pgbackrest options " -n pgouser1

You can also target specific nodes when performing a restore:

    pgo restore mycluster --node-label=failure-domain.beta.kubernetes.io/zone=us-central1-a -n pgouser1

Here are some steps to test PITR:

 * pgo create cluster mycluster
 * create a table on the new cluster called *beforebackup*
 * pgo backup mycluster -n pgouser1
 * create a table on the cluster called *afterbackup*
 * execute *select now()* on the database to get the time, use this timestamp minus a couple of minutes when you perform the restore
 * pgo restore mycluster --pitr-target="2019-01-14 00:02:14.921404+00" --backup-opts="--type=time --log-level-console=info" -n pgouser1
 * wait for the database to be restored
 * execute *\d* in the database and you should see the database state prior to where the *afterbackup* table was created

See the Design section of the Operator documentation for things to consider
before you do a restore.

#### Restore from pgbasebackup

You can find available pgbasebackup backups to use for a pgbasebackup restore using the `pgo show backup` command:

```
$ pgo show backup mycluster --backup-type=pgbasebackup -n pgouser1 | grep "Backup Path"
        Backup Path:    mycluster-backups/2019-05-21-09-53-20
        Backup Path:    mycluster-backups/2019-05-21-06-58-50
        Backup Path:    mycluster-backups/2019-05-21-09-52-52
```

You can then perform a restore using any available backup path:

    pgo restore mycluster --backup-type=pgbasebackup --backup-path=mycluster/2019-05-21-06-58-50 --backup-pvc=mycluster-backup -n pgouser1

When performing the restore, both the backup path and backup PVC can be omitted, and the Operator will use the last pgbasebackup backup created, along with the PVC utilized for that backup:

    pgo restore mycluster --backup-type=pgbasebackup -n pgouser1

Once the pgbasebackup restore is complete, a new PVC will be available with a randomly generated ID that contains the restored database, e.g. PVC  **mycluster-ieqe** in the output below:

```
$ pgo show pvc --all
All Operator Labeled PVCs
        mycluster
        mycluster-backup
        mycluster-ieqe
```

A new cluster can then be created with the same name as the new PVC, as well with the secrets from the original cluster, in order to deploy a new cluster using the restored database:

    pgo create cluster mycluster-ieqe --secret-from=mycluster

If you would like to control the name of the PVC created when performing a pgbasebackup restore, use the `--restore-to-pvc` flag:

    pgo restore mycluster --backup-type=pgbasebackup --restore-to-pvc=mycluster-restored -n pgouser1

#### Restore from pgdump backup

	pgo restore mycluster --backup-type=pgdump --backup-pvc=mycluster-pgdump-pvc --pitr-target="2019-01-15-00-03-25" -n pgouser1
	
To restore the most recent pgdump at the default path, leave off a timestamp:
	
	pgo restore mycluster --backup-type=pgdump --backup-pvc=mycluster-pgdump-pvc -n pgouser1
	

### Fail-over Operations

To perform a manual failover, enter the following:

    pgo failover mycluster --query -n pgouser1

That example queries to find the available Postgres replicas that
could be promoted to the primary.

    pgo failover mycluster --target=sometarget -n pgouser1

That command chooses a specific target, and starts the failover workflow.

#### Create a Cluster with Auto-fail Enabled

To support an automated failover, you can specify the *--autofail* flag
on a Postgres cluster when you create it as follows:

    pgo create cluster mycluster --autofail --replica-count=1 -n pgouser1

You can set the auto-fail flag on a Postgres cluster after it is created
by the following command:

    pgo update cluster --label=autofail=false -n pgouser1
    pgo update cluster --label=autofail=true -n pgouser1

Note that if you do a pgbackrest restore, you will need to reset the
autofail flag to true after the restore is completed.

### Add-On Operations

To add a pgbouncer Deployment to your Postgres cluster, enter:

    pgo create cluster mycluster --pgbouncer -n pgouser1

You can add pgbouncer after a Postgres cluster is created as follows:

    pgo create pgbouncer mycluster
	pgo create pgbouncer --selector=name=mycluster

You can also specify a pgbouncer password as follows:

    pgo create cluster mycluster --pgbouncer --pgbouncer-pass=somepass -n pgouser1

Note, the pgbouncer configuration defaults to specifying only
a single entry for the primary database.  If you want it to
have an entry for the replica service, add the following
configuration to pgbouncer.ini:

    {{.PG_REPLICA_SERVICE_NAME}} = host={{.PG_REPLICA_SERVICE_NAME}} port={{.PG_PORT}} auth_user={{.PG_USERNAME}} dbname={{.PG_DATABASE}}


To add a pgpool Deployment to your Postgres cluster, enter:

    pgo create cluster mycluster --pgpool -n pgouser1

You can also add a pgpool to a cluster after initial creation as follows:

    pgo create pgpool mycluster -n pgouser1

You can remove a pgbouncer or pgpool from a cluster as follows:

    pgo delete pgbouncer mycluster -n pgouser1
    pgo delete pgpool mycluster -n pgouser1

You can create a pgbadger sidecar container in your Postgres cluster
pod as follows:

    pgo create cluster mycluster --pgbadger -n pgouser1

Likewise, you can add the Crunchy Collect Metrics sidecar container
into your Postgres cluster pod as follows:

    pgo create cluster mycluster --metrics -n pgouser1

Note: backend metric storage such as Prometheus and front end 
visualization software such as Grafana are not created automatically 
by the PostgreSQL Operator.  For instructions on installing Grafana and 
Prometheus in your environment, see the [Crunchy Container Suite documentation](https://access.crunchydata.com/documentation/crunchy-containers/2.4.1/examples/metrics/metrics/).

### Scheduled Tasks

There is a cron based scheduler included into the Operator Deployment
by default.  

You can create automated full pgBackRest backups every Sunday at 1 am
as follows:

    pgo create schedule mycluster --schedule="0 1 * * SUN" \
        --schedule-type=pgbackrest --pgbackrest-backup-type=full -n pgouser1

You can create automated diff pgBackRest backups every Monday-Saturday at 1 am
as follows:

    pgo create schedule mycluster --schedule="0 1 * * MON-SAT" \
        --schedule-type=pgbackrest --pgbackrest-backup-type=diff -n pgouser1

You can create automated pgBaseBackup backups every day at 1 am as
follows:

In order to have a backup PVC created, users should run the `pgo backup` command
against the target cluster prior to creating this schedule.

    pgo create schedule mycluster --schedule="0 1 * * *" \
        --schedule-type=pgbasebackup --pvc-name=mycluster-backup -n pgouser1

You can create automated Policy every day at 1 am as follows:

    pgo create schedule --selector=pg-cluster=mycluster --schedule="0 1 * * *" \
         --schedule-type=policy --policy=mypolicy --database=userdb \
         --secret=mycluster-testuser-secret -n pgouser1

### Benchmark Clusters

The pgbench utility containerized and made available to Operator
users.

To create a Benchmark via Cluster Name you enter:

    pgo benchmark mycluster -n pgouser1

To create a Benchmark via Selector, enter:

    pgo benchmark --selector=pg-cluster=mycluster -n pgouser1

To create a Benchmark with a custom transactions, enter:

    pgo create policy --in-file=/tmp/transactions.sql mytransactions -n pgouser1
    pgo benchmark mycluster --policy=mytransactions -n pgouser1

To create a Benchmark with custom parameters, enter:

    pgo benchmark mycluster --clients=10 --jobs=2 --scale=10 --transactions=100 -n pgouser1

You can view benchmarks by entering:

    pgo show benchmark -n pgouser1

### Complex Deployments
#### Create a Cluster using Specific Storage

    pgo create cluster mycluster --storage-config=somestorageconfig -n pgouser1

Likewise, you can specify a storage configuration when creating
a replica:

    pgo scale mycluster --storage-config=someslowerstorage -n pgouser1

This example specifies the *somestorageconfig* storage configuration
to be used by the Postgres cluster.  This lets you specify a storage
configuration that is defined in the *pgo.yaml* file specifically for
a given Postgres cluster.

You can create a Cluster using a Preferred Node as follows:

    pgo create cluster mycluster --node-label=speed=superfast -n pgouser1

That command will cause a node affinity rule to be added to the
Postgres pod which will influence the node upon which Kubernetes
will schedule the Pod.

Likewise, you can create a Replica using a Preferred Node as follows:

    pgo scale mycluster --node-label=speed=slowerthannormal -n pgouser1

#### Create a Cluster with LoadBalancer ServiceType

    pgo create cluster mycluster --service-type=LoadBalancer -n pgouser1

This command will cause the Postgres Service to be of a specific
type instead of the default ClusterIP service type.

#### Miscellaneous 

Create a cluster using the Crunchy Postgres + PostGIS container image:

    pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis -n pgouser1

Create a cluster with a Custom ConfigMap:

    pgo create cluster mycustomcluster --custom-config myconfigmap -n pgouser1

## pgo Global Flags
*pgo* global command flags include:

| Flag | Description |
|:--|:--|
|n | namespace targeted for the command|
|apiserver-url | URL of the Operator REST API service, override with CO_APISERVER_URL environment variable |
|debug |enable debug messages |
|pgo-ca-cert |The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver. Override with PGO_CA_CERT environment variable|
|pgo-client-cert |The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.  Override with PGO_CLIENT_CERT environment variable|
|pgo-client-key |The Client Key file path for authenticating to the PostgreSQL Operator apiserver.  Override with PGO_CLIENT_KEY environment variable|

## pgo Global Environment Variables
*pgo* will pick up these settings if set in your environment:

| Name | Description | NOTES |
|PGOUSERNAME |The username (role) used for auth on the operator apiserver. | Requires that PGOUSERPASS be set. |
|PGOUSERPASS |The password for used for auth on the operator apiserver. | Requires that PGOUSERNAME be set. |
|PGOUSER |The path the the pgorole file. | Will be ignored if either PGOUSERNAME or PGOUSERPASS are set. |
