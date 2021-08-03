---
title: "User / Database Management"
date:
draft: false
weight: 65
---

PGO comes with some out-of-the-box conveniences for managing users and databases in your Postgres cluster. However, you may have requirements where you need to create additional users, adjust user privileges or add additional databases to your cluster.

First, let's understand the default behaviors around user and database management in PGO. From there, we will look at how we can manage users and databases.

## Understanding Default User Management

When you create a Postgres cluster with PGO and do not specify any additional users or databases, PGO will do the following:

- Create a database that matches the name of the Postgres cluster.
- Create an unprivileged Postgres user with the name of the cluster. This user has access to the database created in the previous step.
- Create a Secret with the login credentials and connection details for the Postgres user in relation to the database. This is stored in a Secret named `<clusterName>-pguser-<clusterName>`. These credentials include:
  - `user`: The name of the user account.
  - `password`: The password for the user account.
  - `dbname`: The name of the database that the user has access to by default.
  - `host`: The name of the host of the database. This references the [Service](https://kubernetes.io/docs/concepts/services-networking/service/) of the primary Postgres instance.
  - `port`: The port that the database is listening on.
  - `uri`: A [PostgreSQL connection URI](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING) that provides all the information for logging into the Postgres database.

You can see this default behavior in the [connect to a cluster]({{< relref "./connect-cluster.md" >}}) portion of the tutorial.

As an example, using our `hippo` Postgres cluster, we would see the following created:

- A database named `hippo`.
- A Postgres user named `hippo`.
- A Secret named `hippo-pguser-hippo` that contains the user credentials and connection information.

While the above defaults may work for your application, there are certain cases where you may need to customize your user and databases:

- You may require access to the `postgres` superuser.
- You may need to define privileges for your users.
- You may need multiple databases in your cluster, e.g. in a multi-tenant application.
- Certain users may only be able to access certain databases.

Let's look at how we can manage users and databases using PGO!

## Custom Users and Databases Overview

Before we look at specific examples of adding custom Postgres users and databases to your PGO cluster, first it's important to understand how this deviates from the defaults.

Users and databases can be customized in the `spec.users` section of the custom resource. These can be adding during cluster creation and adjusted over time, but it's important to note the following:

- If `spec.users` is set during cluster creation, PGO will **not** create any default users or databases except for `postgres`. If you want additional databases, you will need to specify them.
- For any users added in `spec.users`, PGO will created a Secret of the format `<clusterName>-pguser-<userName>`. This will contain the user credentials.
  - If no databases are specified, `dbname` and `uri` will not be present in the Secret.
  - If at least one `spec.users.databases` is specified, the first database in the list will be populated into the connection credentials.
- To prevent accidental data loss, PGO will not automatically drop users. We will see how to drop a user below.
- Similarly, to prevent accidental data loss PGO will not automatically drop databases. We will see how to drop a database below.
- Role attributes are not automatically dropped if you remove them. You will have to set the inverse attribute to drop them (e.g. `NOSUPERUSER`).
- The special `postgres` user can be added as one of the custom users; however, the privileges of the users cannot be adjusted.

Let's look at specific examples for how we can manage users.

## Creating a New User

You can create a new user with the following snippet in the `postgrescluster` custom resource. Let's add this to our `hippo` database:

```
spec:
  users:
    - name: rhino
```

You can now apply the changes and see that the new user is created. Note the following:

- The user would only be able to connect to the default `postgres` database.
- The user will not have any connection credentials populated into the `hippo-pguser-rhino` Secret.
- The user is unprivileged.

Let's create a new database named `zoo` that we will let the `rhino` user access:

```
spec:
  users:
    - name: rhino
      databases:
        - zoo
```

Inspect the `hippo-pguser-rhino` Secret. You should now see that the `dbname` and `uri` fields are now populated!

We can set role privileges by using the standard [role attributes](https://www.postgresql.org/docs/current/role-attributes.html) that Postgres provides and adding them to the `spec.users.options`. Let's say we want the rhino to become a superuser (be careful about doling out Postgres superuser privileges!). You can add the following to the spec:

```
spec:
  users:
    - name: rhino
      databases:
        - zoo
      options: "SUPERUSER"
```

There you have it: we have created a Postgres user named `rhino` with superuser privileges that has access to the `rhino` database (though a superuser has access to all databases!).

## Adjusting Privileges

Let's say you want to revoke the superuser privilege from `rhino`. You can do so with the following:

```
spec:
  users:
    - name: rhino
      databases:
        - zoo
      options: "NOSUPERUSER"
```

If you want to add multiple privileges, you can add each privilege with a space between them in `options`, e.g:

```
spec:
  users:
    - name: rhino
      databases:
        - zoo
      options: "CREATEDB CREATEROLE"
```

## Managing the `postgres` User

By default, PGO does not give you access to the `postgres` user. However, you can get access to this account by doing the following:

```
spec:
  users:
    - name: postgres
```

This will create a Secret of the pattern `<clusterName>-pguser-postgres` that contains the credentials of the `postgres` account. For our `hippo` cluster, this would be `hippo-pguser-postgres`.

## Deleting a User

As mentioned earlier, PGO does not let you delete a user automatically: if you remove the user from the spec, it will still exist in your cluster. To remove a user and all of its objects, as a superuser you will need to run [`DROP OWNED`](https://www.postgresql.org/docs/current/sql-drop-owned.html) in each database the user has objects in, and [`DROP ROLE`](https://www.postgresql.org/docs/current/sql-droprole.html)
in your Postgres cluster.

For example, with the above `rhino` user, you would run the following:

```
DROP OWNED BY rhino;
DROP ROLE rhino;
```

Note that you may need to run `DROP OWNED BY rhino CASCADE;` based upon your object ownership structure -- be very careful with this command!

Once you have removed the user in the database, you can remove the user from the custom resource.

## Deleting a Database

As mentioned earlier, PGO does not let you delete a database automatically: if you remove all instances of the database from the spec, it will still exist in your cluster. To completely remove the database, you must run the [`DROP DATABASE`](https://www.postgresql.org/docs/current/sql-dropdatabase.html)
command as a Postgres superuser.

For example, to remove the `zoo` database, you would execute the following:

```
DROP DATABASE zoo;
```

Once you have removed the database, you can remove any references to the database from the custom resource.

## Next Steps

You now know how to manage users and databases in your cluster and have now a well-rounded set of tools to support your "Day 1" operations. Let's start looking at some of the "Day 2" work you can do with PGO, such as [updating to the next Postgres version]({{< relref "./update-cluster.md" >}}), in the [next section]({{< relref "./update-cluster.md" >}}).
