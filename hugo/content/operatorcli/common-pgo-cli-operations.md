---
title: "Common PGO CLI Operations"
date:
draft: false
weight: 5
---

## Common PGO CLI Operations

In all the examples below, the user is specifying the *pgouser1* namespace as the target of the operator.  Replace this value with your own namespace value.  You can specify a default namespace to be used by setting the PGO_NAMESPACE environment variable on the *pgo* client environment.

### PostgreSQL Cluster Operations

#### Creating a Cluster

A user will typically start using the PostgreSQL Operator by creating PostgreSQL cluster including a single PostgreSQL instance as follows:

    pgo create cluster mycluster -n pgouser1

This command creates a PostgreSQL cluster with a single PostgreSQL instance in the *pgouser1* namespace.

You can see the PostgreSQL cluster using the following:

    pgo show cluster mycluster -n pgouser1

You can test the PostgreSQL cluster by entering:

    pgo test mycluster -n pgouser1

You can optionally add a PostgreSQL replica instance to your PostgreSQL cluster as follows:

    pgo scale mycluster -n pgouser1

You can create a PostgreSQL cluster initially with a PostgreSQL replica as follows:

    pgo create cluster mycluster --replica-count=1 -n pgouser1

You can cluster using the PostgreSQL + PostGIS container image:

    pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis -n pgouser1

You can cluster using with a Custom ConfigMap:

    pgo create cluster mycustomcluster --custom-config myconfigmap -n pgouser1

To view the PostgreSQL logs, you can enter commands such as:

    pgo ls mycluster -n pgouser1 /pgdata/mycluster/pg_log
    pgo cat mycluster -n pgouser1 /pgdata/mycluster/pg_log/postgresql-Mon.log | tail -3

#### Scaledown a Cluster

You can remove a PostgreSQL replica using the following:

    pgo scaledown mycluster --query -n pgouser1
    pgo scaledown mycluster --target=sometarget -n pgouser1

#### Delete a Cluster

You can remove a PostgreSQL cluster by entering:

    pgo delete cluster mycluster -n pgouser1

This removes any PostgreSQL instances from being accessed as well as deletes all of its data and backups.

##### Retain Backups

It can often be useful to keep the backups of a cluster even after its deleted, such as for archival purposes or for creating the cluster at a future date. You can delete the cluster but keep its backups using the `--keep-backups` flag:

```bash
pgo delete cluster mycluster --keep-backups -n pgouser1
```

##### Retain Cluster Data

There are rare circumstances in which you may want to keep a copy of the original cluster data, such as when upgrading manually to a newer version of the Operator. In these cases, you can use the `--keep-data` flag:

```bash
pgo delete cluster mycluster --keep-data -n pgouser1
```

**NOTE**: The `--keep-data` flag is deprecated.

#### View Disk Utilization

You can see a comparison of PostgreSQL data size versus the Persistent volume claim size by entering the following:

    pgo df mycluster -n pgouser1


### Backups

#### pgbackrest Operations

By default the PostgreSQL Operator deploys *pgbackrest* backup for a PostgreSQL cluster to hold database backup data.  

You can create a *pgbackrest* backup job as follows:

    pgo backup mycluster -n pgouser1

You can optionally pass *pgbackrest* command options into the backup command as follows:

    pgo backup mycluster --backup-type=pgbackrest --backup-opts="--type=diff" -n pgouser1

See [pgbackrest documentation](https://access.crunchydata.com/documentation/pgbackrest/latest/) for *pgbackrest* command flag descriptions.

You can create a PostgreSQL cluster that does not include pgbackrest if you specify the following:

    pgo create cluster mycluster --pgbackrest=false -n pgouser1

#### Perform a pgbasebackup backup

Alternatively, you can perform a *pgbasebackup* job as follows:

    pgo backup mycluster --backup-type=pgbasebackup -n pgouser1

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


### Label Operations

#### Apply a Label to a PostgreSQL Cluster

You can apply a Kubernetes label to a PostgreSQL cluster as follows:

    pgo label mycluster --label=environment=prod -n pgouser1

In this example, the label key is *environment* and the label value is *prod*.

You can apply labels across a category of PostgreSQL clusters by using the *--selector* command flag as follows:

    pgo label --selector=clustertypes=research --label=environment=prod -n pgouser1

In this example, any PostgreSQL cluster with the label of *clustertypes=research* will have the label *environment=prod* set.

In the following command, you can also view PostgreSQL clusters by using the *--selector* command flag which specifies a label key value to search with:

    pgo show cluster --selector=environment=prod -n pgouser1

### Policy Operations

#### Create a Policy

To create a SQL policy, enter the following:

    pgo create policy mypolicy --in-file=mypolicy.sql -n pgouser1

This examples creates a policy named *mypolicy* using the contents of the file *mypolicy.sql* which is assumed to be in the current directory.

You can view policies as following:

    pgo show policy --all -n pgouser1

#### Apply a Policy

    pgo apply mypolicy --selector=environment=prod
    pgo apply mypolicy --selector=name=mycluster

### Operator Status

#### Show Operator Version

To see what version of the PostgreSQL Operator client and server you are using, enter:

    pgo version

To see the PostgreSQL Operator server status, enter:

    pgo status -n pgouser1

To see the PostgreSQL Operator server configuration, enter:

    pgo show config -n pgouser1

To see what namespaces exist and if you have access to them, enter:

    pgo show namespace -n pgouser1


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

    pgo update cluster --autofail=false -n pgouser1
    pgo update cluster --autofail=true -n pgouser1

Note that if you do a pgbackrest restore, you will need to reset the
autofail flag to true after the restore is completed.

### Configuring pgbouncer, pgpool or pgbadger to Clusters

#### pgbouncer Deployment and Configuration

To add a pgbouncer Deployment to your PostgreSQL cluster, enter:

    pgo create cluster mycluster --pgbouncer -n pgouser1

You can add pgbouncer after a PostgreSQL cluster is created as follows:

    pgo create pgbouncer mycluster
	pgo create pgbouncer --selector=name=mycluster

You can also specify a pgbouncer password as follows:

    pgo create cluster mycluster --pgbouncer --pgbouncer-pass=somepass -n pgouser1

Note, the pgbouncer configuration defaults to specifying only
a single entry for the primary database.  If you want it to
have an entry for the replica service, add the following
configuration to pgbouncer.ini:

    {{.PG_REPLICA_SERVICE_NAME}} = host={{.PG_REPLICA_SERVICE_NAME}} port={{.PG_PORT}} auth_user={{.PG_USERNAME}} dbname={{.PG_DATABASE}}

You can remove a pgbouncer from a cluster as follows:

    pgo delete pgbouncer mycluster -n pgouser1

#### pgpool Deployment and Configuration

To add a pgpool Deployment to your PostgreSQL cluster, enter:

    pgo create cluster mycluster --pgpool -n pgouser1

You can also add a pgpool to a PostgreSQL cluster after initial creation as follows:

    pgo create pgpool mycluster -n pgouser1

You can remove a pgpool from a PostgreSQL cluster as follows:

    pgo delete pgpool mycluster -n pgouser1

#### pgbadger Deployment

You can create a pgbadger sidecar container in your PostgreSQL cluster
pod as follows:

    pgo create cluster mycluster --pgbadger -n pgouser1

#### Metrics Collection Deployment and Configuration

Likewise, you can add the Crunchy Collect Metrics sidecar container
into your PostgresQL cluster pod as follows:

    pgo create cluster mycluster --metrics -n pgouser1

Note: backend metric storage such as Prometheus and front end visualization software such as Grafana are not created automatically by the PostgreSQL Operator.  For instructions on installing Grafana and
Prometheus in your environment, see the [Crunchy Container Suite documentation](https://access.crunchydata.com/documentation/crunchy-containers/4.2.0/examples/metrics/metrics/).

### Scheduled Tasks

There is a cron based scheduler included into the PostgreSQL Operator Deployment
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

### Benchmark Clusters with pgbench

The pgbench utility containerized and made available to PostgreSQL Operator users.

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

Likewise, you can specify a storage configuration when creating a replica:

    pgo scale mycluster --storage-config=someslowerstorage -n pgouser1

This example specifies the *somestorageconfig* storage configuration to be used by the PostgreSQL cluster.  This lets you specify a storage configuration that is defined in the *pgo.yaml* file specifically for a given PostgreSQL cluster.

You can create a PostgreSQL Cluster using a Preferred Node as follows:

    pgo create cluster mycluster --node-label=speed=superfast -n pgouser1

That command will cause a node affinity rule to be added to the PostgreSQL pod which will influence the node upon which Kubernetes will schedule the Pod.

Likewise, you can create a Replica using a Preferred Node as follows:

    pgo scale mycluster --node-label=speed=slowerthannormal -n pgouser1

#### Create a Cluster with LoadBalancer ServiceType

    pgo create cluster mycluster --service-type=LoadBalancer -n pgouser1

This command will cause the PostgreSQL Service to be of a specific type instead of the default ClusterIP service type.

### User Management

#### Create a user
```
pgo create user mycluster --username=someuser --password=somepassword --valid-days=10
```

This command will create a Postgres user on `mycluster` using the given username and password. You can add the `--managed` flag and the user will be managed by the operator. This means that a kubernetes secret will be created along with the Postgres user. Any users created with the `create user` command will automatically have access to all databases that were created when the cluster was created. You will need to manually update their privliges either by using an SQL policy or by using psql if you want to restrict access.


#### Update a user
```
pgo update user mycluster --username=someuser --password=updatedpass
```

This command allows you to update the password for the given user on a cluster. The update user command also allows you to manage when users will expire.
```
pgo update user mycluster --username=someuser --valid-days=40
```

#### Delete a user
```
pgo delete user mycluster --username=someuser
```
This command will delete the give user from `mycluster`. You can delete the user from all clusters by using the `--all` flag instead of the cluster name.
