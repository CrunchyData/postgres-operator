---
title: "Connect to a Postgres Cluster"
draft: false
weight: 120
---

Naturally, once the [PostgreSQL cluster is created]({{< relref "tutorial/create-cluster.md" >}}), you may want to connect to it. You can get the credentials of the users of the cluster using the [`pgo show user`]({{< relref "pgo-client/reference/pgo_show_user.md" >}}) command, i.e.:

```
pgo show user hippo
```

yields output similar to:

```
CLUSTER USERNAME PASSWORD                         EXPIRES STATUS ERROR
------- -------- -------------------------------- ------- ------ -----
hippo   testuser securerandomlygeneratedpassword  never   ok
```

If you need to get the password of one of the system or privileged accounts, you will need to use the `--show-system-accounts` flag, i.e.:

```
pgo show user hippo --show-system-accounts
```

```
CLUSTER USERNAME    PASSWORD                         EXPIRES STATUS ERROR
------- ----------- -------------------------------- ------- ------ -----         
hippo   postgres    B>xy}9+7wTVp)gkntf}X|H@N         never   ok           
hippo   primaryuser ^zULckQy-\KPws:2UoC+szXl         never   ok  
hippo   testuser    securerandomlygeneratedpassword  never   ok
```

Let's look at three different ways we can connect to the PostgreSQL cluster.

## Connecting via `psql`

Let's see how we can connect to `hippo` using [`psql`](https://www.postgresql.org/docs/current/app-psql.html), the command-line tool for accessing PostgreSQL. Ensure you have [installed the `psql` client](https://www.crunchydata.com/developers/download-postgres/binaries/postgresql12).

The PostgreSQL Operator creates a service with the same name as the cluster. See for yourself! Get a list of all of the Services available in the `pgo` namespace:

```
kubectl -n pgo get svc

NAME                         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
hippo                        ClusterIP   10.96.218.63    <none>        2022/TCP,5432/TCP            59m
hippo-backrest-shared-repo   ClusterIP   10.96.75.175    <none>        2022/TCP                     59m
postgres-operator            ClusterIP   10.96.121.246   <none>        8443/TCP,4171/TCP,4150/TCP   71m
```

Let's connect the `hippo` cluster. First, in a different console window, set up a port forward to the `hippo` service:

```
kubectl -n pgo port-forward svc/hippo 5432:5432
```

You can connect to the database with the following command, substituting `datalake` for your actual password:

```
PGPASSWORD=datalake psql -h localhost -p 5432 -U testuser hippo
```

You should then be greeted with the PostgreSQL prompt:

```
psql ({{< param postgresVersion >}})
Type "help" for help.

hippo=>
```

## Connecting via [pgAdmin 4]({{< relref "architecture/pgadmin4.md" >}})

[pgAdmin 4]({{< relref "architecture/pgadmin4.md" >}}) is a graphical tool that can be used to manage and query a PostgreSQL database from a web browser. The PostgreSQL Operator provides a convenient integration with pgAdmin 4 for managing how users can log into the database.

To add pgAdmin 4 to `hippo`, you can execute the following command:

```
pgo create pgadmin -n pgo hippo
```

It will take a few moments to create the pgAdmin 4 instance. The PostgreSQL Operator also creates a pgAdmin 4 service. See for yourself! Get a list of all of the Services available in the `pgo` namespace:

```
kubectl -n pgo get svc

NAME                         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
hippo                        ClusterIP   10.96.218.63    <none>        2022/TCP,5432/TCP            59m
hippo-backrest-shared-repo   ClusterIP   10.96.75.175    <none>        2022/TCP                     59m
hippo-pgadmin                ClusterIP   10.96.165.27    <none>        5050/TCP                     5m1s
postgres-operator            ClusterIP   10.96.121.246   <none>        8443/TCP,4171/TCP,4150/TCP   71m
```

Let's connect to our `hippo` cluster via pgAdmin 4! In a different terminal, set up a port forward to pgAdmin 4:

```
kubectl -n pgo port-forward svc/hippo-pgadmin 5050:5050
```

Navigate your browser to http://localhost:5050 and use your database username (`testuser`) and password (e.g. `datalake`) to log in. Though the prompt says “email address”, using your PostgreSQL username will work:

![pgAdmin 4 Login Page](/images/pgadmin4-login2.png)

(There are occasions where the initial credentials do not properly get set in pgAdmin 4. If you have trouble logging in, try running the command `pgo update user -n pgo hippo --username=testuser --password=datalake`).

Once logged into pgAdmin 4, you will be automatically connected to your database. Explore pgAdmin 4 and run some queries!

## Connecting from a Kubernetes Application

### Within a Kubernetes Cluster

Connecting a Kubernetes application that is within the same cluster that your PostgreSQL cluster is deployed in is as simple as understanding the default [Kubernetes DNS system](https://kubernetes.io/docs/concepts/services-networking/dns-pod-service/#what-things-get-dns-names). A cluster created by the PostgreSQL Operator automatically creates a Service of the same name (e.g. `hippo`).

Following the example we've created, the hostname for our PostgreSQL cluster is `hippo.pgo` (or `hippo.pgo.svc.cluster.local`). To get your exact [DNS resolution rules](https://kubernetes.io/docs/tasks/administer-cluster/dns-debugging-resolution/), you may need to consult with your Kubernetes administrator.

Knowing this, we can construct a [Postgres URI](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING) that contains all of the connection info:

`postgres://testuser:securerandomlygeneratedpassword@hippo.jkatz.svc.cluster.local:5432/hippo`

which breaks down as such:

- `postgres`: the scheme, i.e. a Postgres URI
- `testuser`: the name of the PostgreSQL user
- `securerandomlygeneratedpassword`: the password for `testuser`
- `hippo.jkatz.svc.cluster.local`: the hostname
- `5432`: the port
- `hippo`: the database you want to connect to

If your application or connection driver cannot use the Postgres URI, the above should allow for you to break down the connection string into its appropriate components.

### Outside a Kubernetes Cluster

To connect to a database from an application that is outside a Kubernetes cluster, you will need to set one of the following:

- A Service type of [`LoadBalancer`](https://kubernetes.io/docs/concepts/services-networking/service/#loadbalancer) or [`NodePort`](https://kubernetes.io/docs/concepts/services-networking/service/#nodeport)
- An [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/). The PostgreSQL Operator does not provide any management for Ingress types.

To have the PostgreSQL Operator create a Service that is of type `LoadBalancer` or `NodePort`, you can use the `--service-type` flag as part of creating a PostgreSQL cluster, e.g.:

```
pgo create cluster hippo --service-type=LoadBalancer
```

You can also set the `ServiceType` attribute of the [PostgreSQL Operator configuration]({{< relref "configuration/pgo-yaml-configuration.md" >}}) to provide a default Service type for all PostgreSQL clusters that are created.

## Next Steps

We've created a cluster and we've connected to it! Now, let's learn what customizations we can make as part of the cluster creation process.
