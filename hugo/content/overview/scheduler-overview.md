---
title: "PGO Scheduler Overview"
date:
draft: false
weight: 5
---

## PGO Scheduler

The PostgreSQL Operator includes a cronlike scheduler application called `pgo-scheduler`.  Its purpose is to run automated tasks such as PostgreSQL backups or SQL policies against PostgreSQL instances and clusters created by the PostgreSQL Operator.

PGO Scheduler watches Kubernetes for configmaps with the label `crunchy-scheduler=true` in the same namespace the Operator is deployed.  The configmaps are json objects that describe the schedule such as:

* Cron like schedule such as: * * * * *
* Type of task: `pgbackrest`, `pgbasebackup` or `policy`

Schedules are removed automatically when the configmaps are deleted.

PGO Scheduler uses the `UTC` timezone for all schedules.

### Schedule Expression Format

Schedules are expressed using the following rules:

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

### pgBackRest Schedules

pgBackRest schedules require pgBackRest enabled on the cluster to backup.  The scheduler will not do this on its own.

### pgBaseBackup Schedules

pgBaseBackup schedules require a backup PVC to already be created.  The operator will make this PVC using the backup commands:

    pgo backup mycluster

### Policy Schedules

Policy schedules require a SQL policy already created using the Operator.  Additionally users can supply both the database in which the policy should run and a secret that contains the username and password of the PostgreSQL role that will run the SQL.  If no user is specified the scheduler will default to the replication user provided during cluster creation.
