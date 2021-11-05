---
title: "Logical Replication"
date:
draft: false
weight: 150
---

[Logical replication](https://www.postgresql.org/docs/current/logical-replication.html) is a Postgres feature that provides a convenient way for moving data between databases, particularly Postgres clusters that are in an active state.

You can set up your PGO managed Postgres clusters to use logical replication. This guide provides an example for how to do so.

## Set Up Logical Replication

This example creates two separate Postgres clusters named `hippo` and `rhino`. We will logically replicate data from `rhino` to `hippo`. We can create these two Postgres clusters using the manifests below:

```
---
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
---
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: rhino
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
  users:
    - name: logic
      databases:
        - zoo
      options: "REPLICATION"
```

The key difference between the two Postgres clusters is this section in the `rhino` manifest:

```
users:
  - name: logic
    databases:
      - zoo
    options: "REPLICATION"
```

This creates a database called `zoo` and a user named `logic` with `REPLICATION` privileges. This will allow for replicating data logically to the `hippo` Postgres cluster.

Create these two Postgres clusters. When the `rhino` cluster is ready, [log into the `zoo` database]({{< relref "tutorial/connect-cluster.md" >}}). For convenience, you can use the `kubectl exec` method of logging in:

```
kubectl exec -it -n postgres-operator -c database \
  $(kubectl get pods -n postgres-operator --selector='postgres-operator.crunchydata.com/cluster=rhino,postgres-operator.crunchydata.com/role=master' -o name) -- psql zoo
```

Let's create a simple table called `abc` that contains just integer data. We will also populate this table:

```
CREATE TABLE abc (id int PRIMARY KEY);
INSERT INTO abc SELECT * FROM generate_series(1,10);
```

We need to grant `SELECT` privileges to the `logic` user in order for it to perform an initial data synchronization during logical replication. You can do so with the following command:

```
GRANT SELECT ON abc TO logic;
```

Finally, create a [publication](https://www.postgresql.org/docs/current/logical-replication-publication.html) that allows for the replication of data from `abc`:

```
CREATE PUBLICATION zoo FOR ALL TABLES;
```

Quit out of the `rhino` Postgres cluster.

For the next step, you will need to get the connection information for how to connection as the `logic` user to the `rhino` Postgres database. You can get the key information from the following commands, which return the hostname, username, and password:

```
kubectl -n postgres-operator get secrets rhino-pguser-logic -o jsonpath={.data.host} | base64 -d
kubectl -n postgres-operator get secrets rhino-pguser-logic -o jsonpath={.data.user} | base64 -d
kubectl -n postgres-operator get secrets rhino-pguser-logic -o jsonpath={.data.password} | base64 -d
```

The host will be something like `rhino-primary.postgres-operator.svc` and the user will be `logic`. Further down, the guide references the password as `<LOGIC-PASSWORD>`. You can substitute the actual password there.

Log into the `hippo` Postgres cluster. Note that we are logging into the `postgres` database within the `hippo` cluster:

```
kubectl exec -it -n postgres-operator -c database \
  $(kubectl get pods -n postgres-operator --selector='postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master' -o name) -- psql
```

Create a table called `abc` that is identical to the table in the `rhino` database:

```
CREATE TABLE abc (id int PRIMARY KEY);
```

Finally, create a [subscription](https://www.postgresql.org/docs/current/logical-replication-subscription.html) that will manage the data replication from `rhino` into `hippo`:

```
CREATE SUBSCRIPTION zoo
    CONNECTION 'host=rhino-primary.postgres-operator.svc user=logic dbname=zoo password=<LOGIC-PASSWORD>'
    PUBLICATION zoo;
```

In a few moments, you should see the data replicated into your table:

```
TABLE abc;
```

which yields:

```
 id
----
  1
  2
  3
  4
  5
  6
  7
  8
  9
  10
(10 rows)
```

You can further test that logical replication is working by modifying the data on `rhino` in the `abc` table, and the verifying that it is replicated into `hippo`.
