---
title: "pgBouncer"
draft: false
weight: 170
---

[pgBouncer](https://www.pgbouncer.org/) is a lightweight connection poooler and state manager that provides an efficient gateway to metering connections to PostgreSQL. The PostgreSQL Operator provides an integration with pgBouncer that allows you to deploy it alongside your PostgreSQL cluster.

This tutorial covers how you can set up pgBouncer, functionality that the PostgreSQL Operator provides to manage it, and more.

## Setup pgBouncer

pgBouncer lives as an independent Deployment next to your PostgreSQL cluster but, thanks to the PostgreSQL Operator, is synchronized with various aspects of your environment.

There are two ways you can set up pgBouncer for your cluster. You can add pgBouncer when you create your cluster, e.g.:

```
pgo create cluster hippo --pgbouncer
```

or after your PostgreSQL cluster has been provisioned with the [`pgo create pgbouncer`]({{< relref "pgo-client/reference/pgo_create_pgbouncer.md" >}}):

```
pgo create pgbouncer hippo
```

There are several managed objects that are created alongside the pgBouncer Deployment, these include:

- The pgBouncer Deployment itself
  - One or more pgBouncer Pods
- A pgBouncer ConfigMap, e.g. `hippo-pgbouncer-cm` which has two entries:
  - `pgbouncer.ini`, which is the configuration for the pgBouncer instances
  - `pg_hba.conf`, which controls how clients can connect to `pgBouncer`
- A pgBouncer Secret e.g. `hippo-pgbouncer-secret`, that contains the following values:
  - `password`: the password for the `pgbouncer` user. The `pgbouncer` user is described in more detail further down.
  - `users.txt`: the description for how the `pgbouncer` user and only the `pgbouncer` user can explicitly connect to a pgBouncer instance.

### The `pgbouncer` user

The `pgbouncer` user is a special type of PostgreSQL user that is solely for the administration of pgBouncer. It performs several roles, including:

- Securely load PostgreSQL user credentials into pgBouncer so pgBouncer can perform authentication and connection forwarding
- The ability to log into `pgBouncer` itself for administration, introspection, and looking at statistics

The pgBouncer user **is not meant to be used to log into PostgreSQL directly**: the account is given permissions for ad hoc tasks. More information on how to connect to pgBouncer is provided in the next section.

## Connect to a Postgres Cluster Through pgBouncer

Connecting to a PostgreSQL cluster through pgBouncer is similar to how you [connect to PostgreSQL directly]({{< relref "tutorial/connect-cluster.md">}}), but you are connecting through a different service. First, note the types of users that can connect to PostgreSQL through `pgBouncer`:

- Any regular user that's created through [`pgo create user`]({{< relref "pgo-client/reference/pgo_create_user.md" >}}) or a user that is not a system account.
- The `postgres` superuser

The following example will follow similar steps for how you would connect to a [Postgres Cluster via `psql`]({{< relref "tutorial/connect-cluster.md">}}#connection-via-psql), but applies to all other connection methods.

First, get a list of Services that are available in your namespace:

```
kubectl -n pgo get svc
```

You should see a list similar to:

```
NAME                         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
hippo                        ClusterIP   10.96.104.207   <none>        2022/TCP,5432/TCP            12m
hippo-backrest-shared-repo   ClusterIP   10.96.134.253   <none>        2022/TCP                     12m
hippo-pgbouncer              ClusterIP   10.96.85.35     <none>        5432/TCP                     11m
```

We are going to want to create a port forward to the `hippo-pgbouncer` service. In a separate terminal window, run the following command:

```
kubectl -n pgo port-forward svc/hippo-pgbouncer 5432:5432
```

Recall in the [earlier part of the tutorial]({{< relref "tutorial/connect-cluster.md">}}) that we created a user called `testuser` with a password of `securerandomlygeneratedpassword`. We can the connect to PostgreSQL via pgBouncer by executing the following command:

```
PGPASSWORD=securerandomlygeneratedpassword psql -h localhost -p 5432 -U testuser hippo
```

You should then be greeted with the PostgreSQL prompt:

```
psql ({{< param postgresVersion >}})
Type "help" for help.

hippo=>
```

### Validation: Did this actually work?

This looks just like how we connected to PostgreSQL before, so how do we know that we are connected to PostgreSQL via pgBouncer? Let's log into pgBoucner as the `pgbouncer` user and demonstrate this.

In another terminal window, get the credential for the pgBouncer user. This can be done with the [`pgo show pgbouncer`]({{< relref "pgo-client/reference/pgo_show_pgbouncer.md" >}}) command:

```
pgo show pgbouncer hippo
```

which yields something that looks like:

```
CLUSTER SERVICE         USERNAME  PASSWORD                 CLUSTER IP  EXTERNAL IP
------- --------------- --------- ------------------------ ----------- -----------
hippo   hippo-pgbouncer pgbouncer randompassword           10.96.85.35             
```

Copy the actual password and log into pgbouncer with the following command:

```
PGPASSWORD=randompassword psql -h localhost -p 5432 -U pgbouncer pgbouncer
```

You should see something similar to this:

```
psql ({{< param postgresVersion >}}, server 1.14.0/bouncer)
Type "help" for help.

pgbouncer=#
```

In the `pgboucner` terminal, run the following command. This will show you the overall connection statistics for pgBouncer:

```
SHOW stats;
```

Success, you have connected to pgBouncer!

## Setup pgBouncer with TLS

Similarly to how you can [setup TLS for PostgreSQL]({{< relref "tutorial/tls.md" >}}), you can set up TLS connections for pgBouncer. To do this, the PostgreSQL Operator takes the following steps:

- Ensuring TLS communication between a client (e.g. `psql`, your application, etc.) and pgBouncer
- Ensuring TLS communication between pgBouncer and PostgreSQL

When TLS is enabled, the PostgreSQL Operator configures pgBouncer to require each client to use TLS to communicate with pgBouncer. Additionally, the PostgreSQL Operator requires that pgBouncer and the PostgreSQL cluster share the same certificate authority (CA) bundle, which allows for pgBouncer to communicate with the PostgreSQL cluster using PostgreSQL's [`verify-ca` SSL mode](https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-PROTECTION).

The below guide will show you how set up TLS for pgBouncer.

### Prerequisites

In order to set up TLS connections for pgBouncer, you must first [enable TLS on your PostgreSQL cluster]({{< relref "tutorial/tls.md" >}}).

For the purposes of this exercise, we will re-use the Secret TLS keypair `hippo-tls-keypair` that was created for the PostgreSQL server. This is only being done for convenience: you can substitute `hippo-tls-keypair` with a different TLS key pair as long as it can be verified by the certificate authority (CA) that you selected for your PostgreSQL cluster. Recall that the certificate authority (CA) bundle is stored in a Secret named `postgresql-ca`.

### Create pgBouncer with TLS

Knowing that our TLS key pair is stored in a Secret called `hippo-tls-keypair`, you can setup pgBouncer with TLS using the following command:

```
pgo create pgbouncer hippo --tls-secret=hippo-tls-keypair
```

And that's it! So long as the prerequisites are satisfied, this will create a pgBouncer instance that is TLS enabled.

Don't believe it? Try logging in. First, ensure you have a port-forward from pgBouncer to your host machine:

```
kubectl -n pgo port-forward svc/hippo-pgbouncer 5432:5432
```

Then, connect to the pgBouncer instances:

```
PGPASSWORD=securerandomlygeneratedpassword psql -h localhost -p 5432 -U testuser hippo
```

You should see something similar to this:

```
psql ({{< param postgresVersion >}})
SSL connection (protocol: TLSv1.3, cipher: TLS_AES_256_GCM_SHA384, bits: 256, compression: off)
Type "help" for help.

hippo=>
```

Still don't believe it? You can verify your connection using the PostgreSQL `get_backend_pid()` function and the [`pg_stat_ssl`](https://www.postgresql.org/docs/current/monitoring-stats.html#MONITORING-PG-STAT-SSL-VIEW) monitoring view:

```
hippo=> SELECT * FROM pg_stat_ssl WHERE pid = pg_backend_pid();
  pid  | ssl | version |         cipher         | bits | compression | client_dn | client_serial | issuer_dn
-------+-----+---------+------------------------+------+-------------+-----------+---------------+-----------
 15653 | t   | TLSv1.3 | TLS_AES_256_GCM_SHA384 |  256 | f           |           |               |
(1 row)
```

### Create a PostgreSQL cluster with pgBouncer and TLS

Want to create a PostgreSQL cluster with pgBouncer with TLS enabled? You can with the [`pgo create cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) command and using the `--pgbouncer-tls-secret` flag. Using the same Secrets that were created in the [creating a PostgreSQL cluster with TLS]({{ relref "tutorial/tls.md" }}) tutorial, you can create a PostgreSQL cluster with pgBouncer and TLS with the following command:

```
pgo create cluster hippo \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hippo-tls-keypair \
  --pgbouncer \
  --pgbouncer-tls-secret=hippo-tls-keypair
```

## Customize CPU / Memory for pgBouncer

### Provisioning

The PostgreSQL Operator provides several flags for [`pgo create cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) to help manage resources for pgBouncer:

- `--pgbouncer-cpu`: Specify the CPU Request for pgBouncer
- `--pgbouncer-cpu-limit`: Specify the CPU Limit for pgBouncer
- `--pgbouncer-memory`: Specify the Memory Request for pgBouncer
- `--pgbouncer-memory-limit`: Specify the Memory Limit for pgBouncer

Additional, the PostgreSQL Operator provides several flags for [`pgo create pgbouncer`]({{< relref "pgo-client/reference/pgo_create_pgbouncer.md" >}}) to help manage resources for pgBouncer:

- `--cpu`: Specify the CPU Request for pgBouncer
- `--cpu-limit`: Specify the CPU Limit for pgBouncer
- `--memory`: Specify the Memory Request for pgBouncer
- `--memory-limit`: Specify the Memory Limit for pgBouncer

To create a pgBouncer Deployment that makes a CPU Request of 1.0 with a CPU Limit of 2.0 and a Memory Request of 64Mi with a Memory Limit of 256Mi:

```
pgo create pgbouncer hippo \
  --cpu=1.0 --cpu-limit=2.0 \
  --memory=64Mi --memory-limit=256Mi
```

### Updating

You can also add more memory and CPU resources to pgBouncer with the [`pgo update pgbouncer`]({{< relref "pgo-client/reference/pgo_update_pgbouncer.md" >}}) command, including:

- `--cpu`: Specify the CPU Request for pgBouncer
- `--cpu-limit`: Specify the CPU Limit for pgBouncer
- `--memory`: Specify the Memory Request for pgBouncer
- `--memory-limit`: Specify the Memory Limit for pgBouncer

For example, to update a pgBouncer to a CPU Request of 2.0 with a CPU Limit of 3.0 and a Memory Request of 128Mi with a Memory Limit of 512Mi:

```
pgo update pgbouncer hippo \
  --cpu=2.0 --cpu-limit=3.0 \
  --memory=128Mi --memory-limit=512Mi
```

## Scaling pgBouncer

You can add more pgBouncer instances when provisioning pgBouncer and to an existing pgBouncer Deployment.

### Provisioning

To add pgBouncer instances when creating a PostgreSQL cluster, use the `--pgbouncer-replicas` flag on `pgo create cluster`. For example, to add 2 replicas:

```
pgo create cluster hippo --pgbouncer --pgbouncer-replicas=2
```

If adding a pgBouncer to an already provisioned PostgreSQL cluster, use the `--replicas` flag on `pgo create pgbouncer`. For example, to add a pgBouncer instance with 2 replicas:

```
pgo create pgbouncer hippo --replicas=2
```

### Updating

To update pgBouncer instances to scale the replicas, use the `pgo update pgbouncer` command with the `--replicas` flag. This flag can scale pgBouncer up and down. For example, to run 3 pgBouncer replicas:

```
pgo update pgbouncer hippo --replicas=3
```

## Rotate pgBouncer Password

If you wish to rotate the pgBouncer password, you can use the `--rotate-password` flag on `pgo update pgbouncer`:

```
pgo update pgbouncer hippo --rotate-password
```

This will change the pgBouncer password and synchronize the change across all pgBouncer instances.

## Next Steps

Now that you have connection pooling set up, let's create a [high availability PostgreSQL cluster]({{< relref "tutorial/high-availability.md" >}})!
