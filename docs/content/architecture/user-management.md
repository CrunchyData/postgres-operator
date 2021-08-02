---
title: "User Management"
date:
draft: false
weight: 125
---

PGO manages PostgreSQL users that you define in [`PostgresCluster.spec.users`]({{< relref "/references/crd#postgresclusterspecusersindex" >}}).
There, you can list their [role attributes](https://www.postgresql.org/docs/current/role-attributes.html)
and which databases they can access.

For each user defined, PGO creates a secret named with the pattern `<clusterName>-pguser-<userName>`.
The fields within are PostgreSQL [connection parameters](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-PARAMKEYWORDS).
If you delete the `password` field, PGO will generate another one.

To avoid any risk of data loss, removing a user from `PostgresCluster.spec.users`
does NOT drop that user from PostgreSQL nor does it revoke their access.
To completely remove a user and all the objects they own, you must run
[`DROP OWNED`](https://www.postgresql.org/docs/current/sql-drop-owned.html) and
`DROP USER` or [`DROP ROLE`](https://www.postgresql.org/docs/current/sql-droprole.html)
in PostgreSQL.

Similarly, removing a database from `PostgresCluster.spec.users` does not drop
that database nor revoke any access to it. To completely remove it, you must run
[`DROP DATABASE`](https://www.postgresql.org/docs/current/sql-dropdatabase.html)
as a superuser in PostgreSQL.

The built-in `postgres` superuser can be managed the same way, with one restriction:
its role attributes cannot be changed. This user is always able to `LOGIN` with
`SUPERUSER` access to every database. Including the `postgres` user in `PostgresCluster.spec.users`
gives it a password, allowing it to login over the network.
