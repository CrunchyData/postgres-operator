---
title: "PostgreSQL Operator Backup and Restore Capability"
date:
draft: false
weight: 5
---

## PostgreSQL Operator Backup and Restore Capability

The PostgreSQL Operator provides users with the ability to manage PostgreSQL cluster backups through both native PostgreSQL backup functionality, as well as using pgbackrest, an open source backup and restore solution designed to scale up to the largest databases. By default, beginning with verison 4.0, the PostgreSQL Operator backup command performs a PostgreSQL pgbackrest backup.

The three backup types that can be configured through the PostgreSQL Operator CLI are:

* pgbackrest a simple, reliable backup and restore solution that can seamlessly scale up to the largest databases and workloads.  It provides full, incremental, differential
backups, and point-in-time recovery.

* pg_basebackup is used to take base backups of a running PostgreSQL database cluster. These are taken without affecting other clients to the database, and can be used both for
point-in-time recovery and as the starting point for a log shipping or streaming replication standby servers. 

* pg_dump is a utility for backing up a single PostgreSQL database. It makes consistent backups even if the database is being used concurrently. pg_dump does not block other users
accessing the database (readers or writers).pg_dump 

### pgBackRest Integration

The PostgreSQL Operator integrates various features of the [pgbackrest backup and restore project](https://pgbackrest.org) to support backup and restore capability.

The *pgo-backrest-repo* container acts as a pgBackRest remote repository for the PostgreSQL cluster to use for storing archive files and backups.

The following diagrams depicts some of the integration features:

![alt text](/operator-backrest-integration.png "Operator Backrest Integration")

In this diagram, starting from left to right we see the following:

 * a user when they enter *pgo backup mycluster --backup-type=pgbackrest* will cause a pgo-backrest container to be run as a Job, that container will execute a   *pgbackrest backup* command in the pgBackRest repository container to perform the backup function.

 * a user when they enter *pgo show backup mycluster --backup-type=pgbackrest* will cause a *pgbackrest info* command to be executed on the pgBackRest repository container, the *info* output is sent directly back to the user to view

 * the PostgreSQL container itself will use an archive command, *pgbackrest archive-push* to send archives to the pgBackRest repository container

 * the user entering *pgo create cluster mycluster --pgbackrest* will cause a pgBackRest repository container deployment to be created, that repository is exclusively used for this Postgres cluster

 * lastly, a user entering *pgo restore mycluster* will cause a *pgo-backrest-restore* container to be created as a Job, that container executes the *pgbackrest restore* command

### Support for pgBackRest Use of S3 Buckets

The PostgreSQL Operator supports the use AWS S3 storage buckets for the pgbackrest repository in any pgbackrest-enabled cluster.  When S3 support is enabled for a cluster, all archives will automatically be pushed to a pre-configured S3 storage bucket, and that same bucket can then be utilized for the creation of any backups as well as when performing restores.  Please note that once a storage type has been selected for a cluster during cluster creation (specifically `local`, `s3`, or _both_, as described in detail below), it cannot be changed.    

The PostgreSQL Operator allows for the configuration of a single storage bucket, which can then be utilized across multiple clusters.  Once S3 support has been enabled for a cluster, pgbackrest will create a `backrestrepo` directory in the root of the configured S3 storage bucket (if it does not already exist), and subdirectories will then be created under the `backrestrepo` directory for each cluster created with S3 storage enabled.
