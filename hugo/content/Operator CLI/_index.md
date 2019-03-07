---
title: "Operator CLI"
date:
draft: false
weight: 3
---

The command line tool, pgo, is used to interact with the Postgres Operator.

Most users will work with the Operator using the *pgo* CLI tool.  That tool is downloaded from the GitHub Releases page for the Operator (https://github.com/crunchydata/postgres-operator/releases).

The *pgo* client is provided in Mac, Windows, and Linux binary formats, download the appropriate client to your local laptop or workstation to work with a remote Operator.

## Syntax
Use the following syntax to run  `pgo`  commands from your terminal window:

    pgo [command] ([TYPE] [NAME]) [flags]

Where *command* is a verb like:
 - show
 - get
 - create
 - delete

And *type* is a resource type like:
 - cluster
 - policy
 - user

And *name* is the name of the resource type like:
 - mycluster
 - somesqlpolicy
 - john

To get detailed help information and command flag descriptions on each *pgo* command, enter:

    pgo [command] -h
 
## Operations


See the various *pgo* commands implemented by issuing:

    pgo --help

## Common Operations

### General Operations
#### Show Namespaces

    pgo show namespace

This command shows you all the available namespaces your username
has access to.  In pgo commands where you create resources such
as *pgo create cluster*, you can specify what namespace that resource
should be created in by passing the  *--namespace* command flag.

If you do not pass *--namespace* in on a command, the default
namespace your username is set up to use will be used.

### Cluster Operations
#### Create Cluster With a Primary Only

    pgo create cluster mycluster --namespace=somenamespace

Create a cluster using the Crunchy Postgres + PostGIS container image:

    pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis

#### Create Cluster With a Primary and a Replica

    pgo create cluster mycluster --replica-count=1

#### Scale a Cluster with Additional Replicas

    pgo scale mycluster

#### Create a Cluster with pgbackrest Configured

    pgo create cluster mycluster --pgbackrest

#### Scaledown a Cluster

    pgo scaledown mycluster --query
    pgo scaledown mycluster --target=sometarget

#### Delete a Cluster

    pgo delete cluster mycluster

#### Delete a Cluster and It's Persistent Volume Claims

    pgo delete cluster mycluster --delete-data

#### Test a Cluster

    pgo test mycluster

#### View Disk Utilization

    pgo df mycluster

### Label Operations
#### Apply a Label to a Cluster

    pgo label mycluster --label=environment=prod

#### Appy a Label to a Set of Clusters

    pgo label --selector=clustertypes=research --label=environment=prod

#### Show Clusters by Label

    pgo show cluster --selector=environment=prod

### Policy Operations
#### Create a Policy

    pgo create policy mypolicy --in-file=mypolicy.sql

#### View Policies

    pgo show policy all

#### Apply a Policy

    pgo apply mypolicy --selector=environment=prod
    pgo apply mypolicy --selector=name=mycluster

### Operator Status
#### Show Operator Version

    pgo version

#### Show Operator Status

    pgo status

#### Show Operator Configuration

    pgo show config

### Backup and Restore
#### Perform a pgbasebackup

    pgo backup mycluster

#### Perform a pgbackrest backup

    pgo backup mycluster --backup-type=pgbackrest
    pgo backup mycluster --backup-type=pgbackrest --backup-opts="--type=diff"

The last example passes in pgbackrest flags to the backup command.  See
pgbackrest.org for command flag descriptions.

#### Perform a pgdump backup

	pgo backup mycluster --backup-type=pgdump
	pgo backup mycluster --backup-type=pgdump --backup-opts="--dump-all --verbose"
	pgo backup mycluster --backup-type=pgdump --backup-opts="--schema=myschema"

Note: To run pgdump_all instead of pgdump, pass '--dump-all' flag in --backup-opts as shown above. All --backup-opts should be space delimited.

#### Perform a pgbackrest restore

    pgo restore mycluster

Or perform a restore based on a point in time:

    pgo restore mycluster --pitr-target="2019-01-14 00:02:14.921404+00" --backup-opts="--type=time"

You can also target specific nodes when performing a restore:

    pgo restore mycluster --node-label=failure-domain.beta.kubernetes.io/zone=us-central1-a

Here are some steps to test PITR:

 * pgo create cluster mycluster --pgbackrest
 * create a table on the new cluster called *beforebackup*
 * pgo backup mycluster --backup-type=pgbackrest
 * create a table on the cluster called *afterbackup*
 * execute *select now()* on the database to get the time, use this timestamp minus a couple of minutes when you perform the restore
 * pgo restore mycluster --pitr-target="2019-01-14 00:02:14.921404+00" --backup-opts="--type=time --log-level-console=info"
 * wait for the database to be restored
 * execute *\d* in the database and you should see the database state prior to where the *afterbackup* table was created

See the Design section of the Operator documentation for things to consider
before you do a restore.

#### Restore from pgbasebackup

    pgo create cluster restoredcluster --backup-path=/somebackup/path --backup-pvc=somebackuppvc --secret-from=mycluster

#### Restore from pgdump backup

	pgo restore mycluster --backup-type=pgdump --backup-pvc=mycluster-pgdump-pvc --pitr-target="2019-01-15 00:03:25"
	
To restore the most recent pgdump at the default path, leave off a timestamp:
	
	pgo restore mycluster --backup-type=pgdump --backup-pvc=mycluster-pgdump-pvc
	

### Fail-over Operations

#### Perform a Manual Fail-over

    pgo failover mycluster --query
    pgo failover mycluster --target=sometarget

#### Create a Cluster with Auto-fail Enabled

    pgo create cluster mycluster --autofail --replica-count=1

### Add-On Operations
#### Create a Cluster with pgbouncer

    pgo create cluster mycluster --pgbouncer
	pgo create cluster mycluster --pgbouncer --pgbouncer-user=someuser --pgbouncer-pass=somepass

#### Create a Cluster with pgpool

    pgo create cluster mycluster --pgpool

#### Add pgbouncer to a Cluster

    pgo create pgbouncer mycluster
	pgo create pgbouncer mycluster --pgbouncer-user=someuser --pgbouncer-pass=somepass

Note, the pgbouncer configuration defaults to specifying only
a single entry for the primary database.  If you want it to
have an entry for the replica service, add the following
configuration to pgbouncer.ini:

    {{.PG_REPLICA_SERVICE_NAME}} = host={{.PG_REPLICA_SERVICE_NAME}} port=5432 auth_user={{.PG_USERNAME}} dbname=userdb


#### Add pgpool to a Cluster

    pgo create pgpool mycluster

#### Remove pgbouncer from a Cluster

    pgo delete pgbouncer mycluster

#### Remove pgpool from a Cluster

    pgo delete pgpool mycluster

#### Create a Cluster with pgbadger

    pgo create cluster mycluster --pgbadger

#### Create a Cluster with Metrics Collection

    pgo create cluster mycluster --metrics

Note: backend metric storage such as Prometheus and front end 
visualization software such as Grafana are not created automatically 
by the PostgreSQL Operator.  For instructions on installing Grafana and 
Prometheus in your environment, see the [Crunchy Container Suite documentation](http://crunchydata.github.io/crunchy-containers/stable/examples/metrics/metrics/).

### Scheduled Tasks

#### Automated full pgBackRest backups every Sunday at 1 am

    pgo create schedule mycluster --schedule="0 1 * * SUN" \
        --schedule-type=pgbackrest --pgbackrest-backup-type=full

#### Automated diff pgBackRest backups every Monday-Saturday at 1 am

    pgo create schedule mycluster --schedule="0 1 * * MON-SAT" \
        --schedule-type=pgbackrest --pgbackrest-backup-type=diff

#### Automated pgBaseBackup backups every day at 1 am

In order to have a backup PVC created, users should run the `pgo backup` command
against the target cluster prior to creating this schedule.

    pgo create schedule mycluster --schedule="0 1 * * *" \
        --schedule-type=pgbasebackup --pvc-name=mycluster-backup

#### Automated Policy every day at 1 am

    pgo create schedule --selector=pg-cluster=mycluster --schedule="0 1 * * *" \
         --schedule-type=policy --policy=mypolicy --database=userdb \
         --secret=mycluster-testuser-secret

### Complex Deployments
#### Create a Cluster using Specific Storage

    pgo create cluster mycluster --storage-config=somestorageconfig

#### Create a Cluster using a Preferred Node

    pgo create cluster mycluster --node-label=speed=superfast

#### Create a Replica using Specific Storage

    pgo scale mycluster --storage-config=someslowerstorage

#### Create a Replica using a Preferred Node

    pgo scale mycluster --node-label=speed=slowerthannormal

#### Create a Cluster with LoadBalancer ServiceType

    pgo create cluster mycluster --service-type=LoadBalancer
## pgo Global Flags
*pgo* global command flags include:

| Flag | Description |
|:--|:--|
|apiserver-url | URL of the Operator REST API service, override with CO_APISERVER_URL environment variable |
|debug |enable debug messages |
|pgo-ca-cert |The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver. Override with PGO_CA_CERT environment variable|
|pgo-client-cert |The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.  Override with PGO_CLIENT_CERT environment variable|
|pgo-client-key |The Client Key file path for authenticating to the PostgreSQL Operator apiserver.  Override with PGO_CLIENT_KEY environment variable|
