---
title: "Logical Backups (pg_dump)"
draft: false
weight: 200
---

The PostgreSQL Operator supports taking logical backups with `pg_dump` and `pg_dumpall`. While they do not provide the same performance and storage optimizations as the physical backups provided by pgBackRest, logical backups are helpful when one wants to upgrade between major PostgreSQL versions, or provide only a subset of a database, such as a table.

### Create a Logical Backup

To create a logical backup of the `postgres` database, you can run the following command:

```
pgo backup hippo --backup-type=pgdump
```

To create a logical backup of a specific database, you can use the `--database` flag, as in the following command:

```
pgo backup hippo --backup-type=pgdump --database=hippo
```

You can pass in specific options to `--backup-opts`, which can accept most of the options that the [`pg_dump`](https://www.postgresql.org/docs/current/app-pgdump.html) command accepts. For example, to only dump the data from a specific table called `users`:

```
pgo backup hippo --backup-type=pgdump --backup-opts="-t users"
```

To use `pg_dumpall` to create a logical backup of all the data in a PostgreSQL cluster, you must pass the `--dump-all` flag in `--backup-opts`, i.e.:

```
pgo backup hippo --backup-type=pgdump --backup-opts="--dump-all"
```

### Viewing Logical Backups

To view an available list of logical backups, you can use the `pgo show backup`
command with the `--backup-type=pgdump` flag:

```
pgo show backup --backup-type=pgdump hippo
```

This provides information about the PVC that the logical backups are stored on as well as the timestamps required to perform a restore from a logical backup.

### Restore from a Logical Backup

To restore from a logical backup, you need to reference the PVC that the logical backup is stored to, as well as the timestamp that was created by the logical backup.

You can get the timestamp from the `pgo show backup --backup-type=pgdump` command.

You can restore a logical backup using the following command:

```
pgo restore hippo --backup-type=pgdump --backup-pvc=hippo-pgdump-pvc \
  --pitr-target="2019-01-15-00-03-25" -n pgouser1
```

To restore to a specific database, add the `--pgdump-database` flag to the command from above:

```
pgo restore hippo --backup-type=pgdump --backup-pvc=hippo-pgdump-pvc \
  --pgdump-database=mydb --pitr-target="2019-01-15-00-03-25" -n pgouser1
```
