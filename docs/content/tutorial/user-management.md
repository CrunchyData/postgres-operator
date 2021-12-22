---
title: "User / Database Management"
date:
draft: false
weight: 65
---

PGO comes with some out-of-the-box conveniences for managing users and databases in your Postgres cluster. However, you may have requirements where you need to create additional users, adjust user privileges or add additional databases to your cluster.

For detailed information for how user and database management works in PGO, please see the [User Management]({{< relref "architecture/user-management.md" >}}) section of the architecture guide.

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

If you want to add multiple privileges, you can add each privilege with a space between them in `options`, e.g.:

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

PGO does not delete users automatically: after you remove the user from the spec, it will still exist in your cluster. To remove a user and all of its objects, as a superuser you will need to run [`DROP OWNED`](https://www.postgresql.org/docs/current/sql-drop-owned.html) in each database the user has objects in, and [`DROP ROLE`](https://www.postgresql.org/docs/current/sql-droprole.html)
in your Postgres cluster.

For example, with the above `rhino` user, you would run the following:

```
DROP OWNED BY rhino;
DROP ROLE rhino;
```

Note that you may need to run `DROP OWNED BY rhino CASCADE;` based upon your object ownership structure -- be very careful with this command!

## Deleting a Database

PGO does not delete databases automatically: after you remove all instances of the database from the spec, it will still exist in your cluster. To completely remove the database, you must run the [`DROP DATABASE`](https://www.postgresql.org/docs/current/sql-dropdatabase.html)
command as a Postgres superuser.

For example, to remove the `zoo` database, you would execute the following:

```
DROP DATABASE zoo;
```

## Next Steps

You now know how to manage users and databases in your cluster and have now a well-rounded set of tools to support your "Day 1" operations. Let's start looking at some of the "Day 2" work you can do with PGO, such as [updating to the next Postgres version]({{< relref "./update-cluster.md" >}}), in the [next section]({{< relref "./update-cluster.md" >}}).
