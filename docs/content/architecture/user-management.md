---
title: "User Management"
date:
draft: false
weight: 125
---

PGO manages PostgreSQL users that you define in [`PostgresCluster.spec.users`]({{< relref "/references/crd#postgresclusterspecusersindex" >}}).
There, you can list their [role attributes](https://www.postgresql.org/docs/current/role-attributes.html) and which databases they can access.

Below is some information on how the user and database management systems work. To try out some examples, please see the [user and database management]({{< relref "tutorial/user-management.md" >}}) section of the [tutorial]({{< relref "tutorial/_index.md" >}}).

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

You can see this default behavior in the [connect to a cluster]({{< relref "tutorial/connect-cluster.md" >}}) portion of the tutorial.

As an example, using our `hippo` Postgres cluster, we would see the following created:

- A database named `hippo`.
- A Postgres user named `hippo`.
- A Secret named `hippo-pguser-hippo` that contains the user credentials and connection information.

While the above defaults may work for your application, there are certain cases where you may need to customize your user and databases:

- You may require access to the `postgres` superuser.
- You may need to define privileges for your users.
- You may need multiple databases in your cluster, e.g. in a multi-tenant application.
- Certain users may only be able to access certain databases.

## Custom Users and Databases

Users and databases can be customized in the `spec.users` section of the custom resource. These can be adding during cluster creation and adjusted over time, but it's important to note the following:

- If `spec.users` is set during cluster creation, PGO will **not** create any default users or databases except for `postgres`. If you want additional databases, you will need to specify them.
- For any users added in `spec.users`, PGO will created a Secret of the format `<clusterName>-pguser-<userName>`. This will contain the user credentials.
  - If no databases are specified, `dbname` and `uri` will not be present in the Secret.
  - If at least one `spec.users.databases` is specified, the first database in the list will be populated into the connection credentials.
- To prevent accidental data loss, PGO will not automatically drop users. We will see how to drop a user below.
- Similarly, to prevent accidental data loss PGO will not automatically drop databases. We will see how to drop a database below.
- Role attributes are not automatically dropped if you remove them. You will have to set the inverse attribute to drop them (e.g. `NOSUPERUSER`).
- The special `postgres` user can be added as one of the custom users; however, the privileges of the users cannot be adjusted.

For specific examples for how to manage users, please see the [user and database management]({{< relref "tutorial/user-management.md" >}}) section of the [tutorial]({{< relref "tutorial/_index.md" >}}).

## Custom Passwords

There are cases where you may want to explicitly provide your own password for a Postgres user. PGO determines the password from an attribute in the user Secret called `verifier`. This contains a hashed copy of your password. When `verifier` changes, PGO will load the contents of the verifier into your Postgres cluster. This method allows for the secure transmission of the password into the Postgres database.

Postgres provides two methods for hashing password: SCRAM-SHA-256 and md5. The preferred (and as of PostgreSQL 14, default) method is to use SCRAM, which is also what PGO uses as a default.

You can still provide a plaintext password in the `password` field, but this merely for convenience: this makes it easier for your application to connect with an updated password.

### Example

For example, let's say we have a Postgres cluster named `hippo` and a Postgres user named `hippo`. The Secret then would be called `hippo-pguser-hippo`.

Let's say we want to set the password for `hippo` to be `datalake`. We would first need to create a SCRAM version of the password. You can find a script that [creates Postgres SCRAM-SHA-256](https://gist.github.com/jkatz/e0a1f52f66fa03b732945f6eb94d9c21) passwords [here](https://gist.github.com/jkatz/e0a1f52f66fa03b732945f6eb94d9c21).

Below is an example of a SCRAM verifier that may be generated for the password `datalake`, stored in two environmental variables:

```
PASSWORD=datalake
VERIFIER='SCRAM-SHA-256$4096:FAnSkUrL3jH6Bp17P5FTuQ==$BecLU0YXzBUwnTk3b3ghIZ7/zS2XvD8wX50Hz5JL0q4=:pxuEtozYTZpAc6wINV53sCr4Afxk8LLfor5MvkYp21s='
```

To store these in a Kubernetes Secret, both values need to be base64 encoded. You can do so with the following example:

```
PASSWORD=$(echo -n $PASSWORD | base64 | tr -d '\n')
VERIFIER=$(echo -n $VERIFIER | base64 | tr -d '\n')
```

Finally, you can update the Secret using `kubectl patch`. The below assumes that the Secret is stored in the `postgres-operator` namespace:

```
kubectl patch secret -n postgres-operator hippo-pguser-hippo -p \
   "{\"data\":{\"password\":\"${PASSWORD}\",\"verifier\":\"${VERIFIER}\"}}"
```

PGO will apply the updated password to Postgres, and you will be able to login with the password `datalake`.
