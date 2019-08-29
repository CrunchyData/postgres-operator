---
title: "pgBackrest with S3"
date:
draft: false
weight: 5
---

## pgbackrest Configuration

The PostgreSQL Operator integrates various features of the [pgbackrest backup and restore project](https://pgbackrest.org).  

The *pgo-backrest-repo* container acts as a pgBackRest remote repository for the Postgres cluster to use for storing archive files and backups.

The following diagrams depicts some of the integration features:

![alt text](/operator-backrest-integration.png "Operator Backrest Integration")

In this diagram, starting from left to right we see the following:

 * a user when they enter *pgo backup mycluster --backup-type=pgbackrest* will cause a pgo-backrest container to be run as a Job, that container will execute a   *pgbackrest backup*
 command in the pgBackRest repository container to perform the backup function.

 * a user when they enter *pgo show backup mycluster --backup-type=pgbackrest* will cause a *pgbackrest info* command to be executed on the pgBackRest repository container, the
 *info* output is sent directly back to the user to view

 * the PostgreSQL container itself will use an archive command, *pgbackrest archive-push* to send archives to the pgBackRest repository container

 * the user entering *pgo create cluster mycluster --pgbackrest* will cause
a pgBackRest repository container deployment to be created, that repository
is exclusively used for this Postgres cluster

 * lastly, a user entering *pgo restore mycluster* will cause a *pgo-backrest-restore* container to be created as a Job, that container executes the *pgbackrest restore* command

### pgbackrest Restore

The pgbackrest restore command is implemented as the *pgo restore* command.  This command is destructive in the sense that it is meant to *restore* a PG cluster meaning it will
revert the PG cluster to a restore point that is kept in the pgbackrest repository.   The prior primary data is not deleted but left in a PVC to be manually cleaned up by a DBA. 
The restored PG cluster will work against a new PVC created from the restore workflow.  

When doing a *pgo restore*, here is the workflow the PostgreSQL Operator executes:

 * turn off autofail if it is enabled for this PG cluster
 * allocate a new PVC to hold the restored PG data
 * delete the the current primary database deployment
 * update the pgbackrest repo for this PG cluster with a new data path of the new PVC
 * create a pgo-backrest-restore job, this job executes the *pgbackrest restore* command from the pgo-backrest-restore container, this Job mounts the newly created PVC
 * once the restore job completes, a new primary Deployment is created which mounts the restored PVC volume

At this point the PostgreSQL database is back in a working state.  DBAs are still responsible to re-enable autofail using *pgo update cluster* and also perform a pgBackRest backup after the new primary is ready.  This version of the PostgreSQL Operator also does not handle any errors in the PG replicas after a restore, that is left for the DBA to handle.

Other things to take into account before you do a restore:

 * if a schedule has been created for this PostgreSQL cluster, delete that schedule prior to performing a restore
 * If your database has been paused after the target restore was completed, then you would need to run the psql command select pg_wal_replay_resume() to complete the recovery, on PostgreSQL 9.6⁄9.5 systems, the command you will use is select pg_xlog_replay_resume(). You can confirm the status of your database by using the built in postgres admin functions
 found [here:] (https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-RECOVERY-CONTROL-TABLE)
 * a pgBackRest restore is destructive in the sense that it deletes the existing primary deployment for the cluster prior to creating a new deployment containing the restored
 primary database.  However, in the event that the pgBackRest restore job fails, the `pgo restore` command be can be run again, and instead of first deleting the primary deployment
 (since one no longer exists), a new primary will simply be created according to any options specified.  Additionally, even though the original primary deployment will be deleted,
 the original primary PVC will remain.
 * there is currently no Operator validation of user entered pgBackRest command options, you will need to make sure to enter these correctly, if not the pgBackRest restore command
 can fail.
 * the restore workflow does not perform a backup after the restore nor does it verify that any replicas are in a working status after the restore, it is possible you might have to
 take actions on the replica to get them back to replicating with the new restored primary.
 * pgbackrest.org suggests running a pgbackrest backup after a restore, this needs to be done by the DBA as part of a restore
 * when performing a pgBackRest restore, the **node-label** flag can be utilized to target a specific node for both the pgBackRest restore job and the new (i.e. restored) primary
 deployment that is then created for the cluster.  If a node label is not specified, the restore job will not target any specific node, and the restored primary deployment will
 inherit any node labels defined for the original primary deployment.

### pgbackrest AWS S3 Support

The PostgreSQL Operator supports the use AWS S3 storage buckets for the pgbackrest repository in any pgbackrest-enabled cluster.  When S3 support is enabled for a cluster, all
archives will automatically be pushed to a pre-configured S3 storage bucket, and that same bucket can then be utilized for the creation of any backups as well as when performing
restores.  Please note that once a storage type has been selected for a cluster during cluster creation (specifically `local`, `s3`, or _both_, as described in detail below), it
cannot be changed.    

The PostgreSQL Operator allows for the configuration of a single storage bucket, which can then be utilized across multiple clusters.  Once S3 support has been enabled for a
cluster, pgbackrest will create a `backrestrepo` directory in the root of the configured S3 storage bucket (if it does not already exist), and subdirectories will then be created
under the `backrestrepo` directory for each cluster created with S3 storage enabled.

#### S3 Configuration

In order to enable S3 storage, you must provide the required AWS S3 configuration information prior to deploying the Operator.  First, you will need to add the proper S3 bucket
name, AWS S3 endpoint and AWS S3 region to the `Cluster` section of the `pgo.yaml` configuration file (additional information regarding the configuration of the `pgo.yaml` file can
be found [here](/configuration/pgo-yaml-configuration/))  :

```yaml
Cluster:
  BackrestS3Bucket: containers-dev-pgbackrest
  BackrestS3Endpoint: s3.amazonaws.com
  BackrestS3Region: us-east-1
```

You will then need to specify the proper credentials for authenticating into the S3 bucket specified by adding a **key** and **key secret** to the `$PGOROOT/conf/pgo-backrest
repo/aws-s3-credentials.yaml` configuration file:

```yaml
---
aws-s3-key: ABCDEFGHIJKLMNOPQRST
aws-s3-key-secret: ABCDEFG/HIJKLMNOPQSTU/VWXYZABCDEFGHIJKLM
```

Once the above configuration details have been provided, you can deploy the Operator per the [PGO installation instructions](/installation/operator-install/).  

#### Enabling S3 Storage in a Cluster

With S3 storage properly configured within your PGO installation, you can now select either local storage, S3 storage, or _both_ when creating a new cluster.  The type of storage
selected upon creation of the cluster will determine the type of storage that can subsequently be used when performing pgbackrest backups and restores.  A storage type is specified
using the `--pgbackrest-storage-type` flag, and can be one of the following values:

* `local` - pgbackrest will use volumes local to the container (e.g. Persistent Volumes) for storing archives, creating backups and locating backups for restores.  This is the
default value for the `--pgbackrest-storage-type` flag.
* `s3` - pgbackrest will use the pre-configured AWS S3 storage bucket for storing archives, creating backups and locating backups for restores
* `local,s3` (both) - pgbackrest will use both volumes local to the container (e.g. persistent volumes), as well as the pre-configured AWS S3 storage bucket, for storing archives. 
Also allows the use of local and/or S3 storage when performing backups and restores.

For instance, the following command enables both `local` and `s3` storage in a new cluster:

```bash
pgo create cluster mycluster --pgbackrest-storage-type=local,s3 -n pgouser1
```

As described above, this will result in pgbackrest pushing archives to both local and S3 storage, while also allowing both local and S3 storage to be utilized for backups and
restores.  However, you could also enable S3 storage only when creating the cluster:

```bash
pgo create cluster mycluster --pgbackrest-storage-type=s3 -n pgouser1
```

Now all archives for the cluster will be pushed to S3 storage only, and local storage will not be utilized for storing archives (nor can local storage be utilized for backups and
restores).

#### Using S3 to Backup & Restore

As described above, once S3 storage has been enabled for a cluster, it can also be used when backing up or restoring a cluster.  Here a both local and S3 storage is selected when
performing a backup:

```bash
pgo backup mycluster --backup-type=pgbackrest --pgbackrest-storage-type=local,s3 -n pgouser1
```

This results in pgbackrest creating a backup in a local volume (e.g. a persistent volume), while also creating an additional backup in the configured S3 storage bucket.  However, a
backup can be created using S3 storage only:

```bash
pgo backup mycluster --backup-type=pgbackrest --pgbackrest-storage-type=s3 -n pgouser1
```

Now pgbackrest will only create a backup in the S3 storage bucket only.

When performing a restore, either `local` or `s3` must be selected (selecting both for a restore will result in an error).  For instance, the following command specifies S3 storage
for the restore:

```bash
pgo restore mycluster --pgbackrest-storage-type=s3 -n pgouser1
```

This will result in a full restore utilizing the backups and archives stored in the configured S3 storage bucket.

_Please note that because `local` is the default storage type for the `backup` and `restore` commands, `s3` must be explicitly set using the `--pgbackrest-storage-type` flag when
performing backups and restores on clusters where only S3 storage is enabled._

#### AWS Certificate Authority

The PostgreSQL Operator installation includes a default certificate bundle that is utilized by default to establish trust between pgbackrest and the AWS S3 endpoint used for S3
storage.  Please modify or replace this certificate bundle as needed prior to deploying the Operator if another certificate authority is needed to properly establish trust between pgbackrest
and your S3 endpoint.  

The certificate bundle can be found here: `$PGOROOT/pgo-backrest-repo/aws-s3-ca.crt`.  

When modifying or replacing the certificate bundle, please be sure to maintain the same path and filename.
