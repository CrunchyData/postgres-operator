| Tables   |      Are      |  Cool |
|----------|:-------------:|------:|
| col 1 is |  left-aligned | $1600 |
| col 2 is |    centered   |   $12 |
| col 3 is | right-aligned |    $1 |


First Header | Second Header
------------ | -------------
Content from cell 1 | Content from cell 2
Content in the first column | Content in the second column

# pgo CLI
The command line tool, pgo, is used to interact with the Postgres Operator.

Most users will work with the Operator using the *pgo* CLI tool.  That tool is downloaded from the Github Releases page for the Operator (https://github.com/crunchydata/postgres-operator/releases).

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

## Operations


| Header1 | Header2 | Header3 |
|:--------|:-------:|--------:|
| cell1   | cell2   | cell3   |
| cell4   | cell5   | cell6   |
|----
| cell1   | cell2   | cell3   |
| cell4   | cell5   | cell6   |
|=====
| Foot1   | Foot2   | Foot3
{: rules="groups"}

The following table shows the *pgo* operations currently implemented: 
| Operation |Syntax  | Description |
|--|--|--|
| apply |`pgo apply mypolicy  --selector=name=mycluster`  | Apply a SQL policy on a Postgres cluster(s)|

| backup |`pgo backup mycluster`  |Perform a backup on a Postgres cluster(s) |
| create |pgo create cluster mycluster  |Create an Operator resource type (e.g. cluster, policy, user) |
| delete |pgo delete cluster mycluster  |Delete an Operator resource type (e.g. cluster, policy, user) |
| df |pgo df mycluster  |Display the disk status/capacity of a Postgres cluster. |
| failover |pgo failover mycluster  |Perform a manual failover of a Postgres cluster. |
| help |pgo help |Display general *pgo* help information. |
| label |pgo label mycluster --label=environment=prod  |Create a metadata label for a Postgres cluster(s). |
| load |pgo load --load-config=load.json --selector=name=mycluster  |Perform a data load into a Postgres cluster(s).|
| reload |pgo reload mycluster  |Perform a pg_ctl reload command on a Postgres cluster(s). |
| restore |pgo restore mycluster --to-pvc=restored  |Perform a pgbackrest restore on a Postgres cluster. |
| scale |pgo scale mycluster  |Create a Postgres replica(s) for a given Postgres cluster. |
| scaledown |pgo scaledown  mycluster --query  |Delete a replica from a Postgres cluster. |
| show |pgo show cluster mycluster  |Display Operator resource information (e.g. cluster, user, policy). |
| status |pgo status  |Display Operator status. |
| test |pgo test mycluster  |Perform a SQL test on a Postgres cluster(s). |
| update |pgo update cluster --label=autofail=false  |Update a Postgres cluster(s). |
| upgrade |pgo upgrade mycluster  |Perform a minor upgrade to a Postgres cluster(s). |
| user |pgo user --selector=name=mycluster --update-passwords  |Perform Postgres user maintenance on a Postgres cluster(s). |
| version |pgo version  |Display Operator version information. |

## Common Operations

### Cluster Operations
#### Create Cluster With a Primary Only

    pgo create cluster mycluster

#### Create Cluster With a Primary and a Replica

    pgo create cluster mycluster --replica-count=1

#### Scale a Cluster with Additional Replicas

    pgo scale cluster mycluster

#### Create a Cluster with pgbackrest Configured

    pgo create cluster mycluster --pgbackrest

#### Scaledown a Cluster 

    pgo scaledown cluster mycluster --query
    pgo scaledown cluster mycluster --target=sometarget

#### Delete a Cluster

    pgo delete cluster mycluster

#### Delete a Cluster and It's Data

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

#### Perform a pgbackrest restore

    pgo restore mycluster --to-pvc=restoredname
    pgo create restoredname --pgbackrest --pgbackrest-restore-from=mycluster

#### Restore from pgbasebackup

    pgo create cluster restoredcluster --backup-path=/somebackup/path --backup-pvc=somebackuppvc

### Fail-over Operations

#### Perform a Manual Fail-over

    pgo failover mycluster --query
    pgo failover mycluster --target=sometarget

#### Create a Cluster with Auto-fail Enabled

    pgo create cluster mycluster --autofail

### Add-On Operations
#### Create a Cluster with pgbouncer

    pgo create cluster mycluster --pgbouncer

#### Create a Cluster with pgpool

    pgo create cluster mycluster --pgpool

#### Add pgbouncer to a Cluster

    pgo create pgbouncer mycluster

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
## Flags
*pgo* command flags include:

| Flag | Description |
|:--|:--|
| apiserver-url | URL of the Operator REST API service, override with CO_APISERVER_URL environment variable |
|debug |enable debug messages |
|pgo-ca-cert |The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver. Override with PGO_CA_CERT environment variable|
|pgo-client-cert |The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.  Override with PGO_CLIENT_CERT environment variable|
|pgo-client-key |The Client Key file path for authenticating to the PostgreSQL Operator apiserver.  Override with PGO_CLIENT_KEY environment variable|



