---
title: "pgAdmin 4"
date:
draft: false
weight: 900
---

![pgAdmin 4 Query](/images/pgadmin4-query.png)

[pgAdmin 4](https://www.pgadmin.org/) is a popular graphical user interface that
makes it easy to work with PostgreSQL databases from both a desktop or web-based
client. With its ability to manage and orchestrate changes for PostgreSQL users,
the PostgreSQL Operator is a natural partner to keep a pgAdmin 4 environment
synchronized with a PostgreSQL environment.

The PostgreSQL Operator lets you deploy a pgAdmin 4 environment alongside a
PostgreSQL cluster and keeps users' database credentials synchronized. You can
simply log into pgAdmin 4 with your PostgreSQL username and password and
immediately have access to your databases.

## Deploying pgAdmin 4

For example, let's use a PostgreSQL cluster called hippo `hippo` that has a user
named `hippo` with password `datalake`:

```
pgo create cluster hippo --username=hippo --password=datalake
```

After the PostgreSQL cluster becomes ready, you can create a pgAdmin 4
deployment with the [`pgo create pgadmin`]({{< relref "/pgo-client/reference/pgo_create_pgadmin.md" >}})
command:

```
pgo create pgadmin hippo
```

This will use the configured storage configuration and default PVC size. If desired, 
you can set a custom storage configuration (in this case `gce`) and PVC size using:

```
pgo create pgadmin hippo --storage-config=gce --pvc-size=1G
```

This creates a pgAdmin 4 deployment unique to this PostgreSQL cluster and
synchronizes the PostgreSQL user information into it. To access pgAdmin 4, you
can set up a port-forward to the Service, which follows the pattern `<clusterName>-pgadmin`, to port `5050`:

```
kubectl port-forward svc/hippo-pgadmin 5050:5050
```

Point your browser at `http://localhost:5050` and use your database
username (e.g. `hippo`) and password (e.g. `datalake`) to log in. Though the
prompt says "email address", using your PostgreSQL username will work.

![pgAdmin 4 Login Page](/images/pgadmin4-login.png)

(**Note**: if your password does not appear to work, you can retry setting up
the user with the [`pgo update user`]({{< relref "/pgo-client/reference/pgo_update_user.md" >}})
command: `pgo update user hippo --password=datalake`)

## User Synchronization

The [`pgo create user`]({{< relref "/pgo-client/reference/pgo_create_user.md" >}}),
[`pgo update user`]({{< relref "/pgo-client/reference/pgo_update_user.md" >}}),
and [`pgo delete user`]({{< relref "/pgo-client/reference/pgo_delete_user.md" >}})
commands are synchronized with the pgAdmin 4 deployment. Note that if you use
`pgo create user` without the `--managed` flag prior to deploying pgAdmin 4,
then the user's credentials will not be synchronized to the pgAdmin 4
deployment. However, a subsequent run of `pgo update user --password` will
synchronize the credentials with pgAdmin 4.

## Deleting pgAdmin 4

You can remove the pgAdmin 4 deployment with the
[`pgo delete pgadmin`]({{< relref "/pgo-client/reference/pgo_delete_pgadmin.md" >}})
command.
