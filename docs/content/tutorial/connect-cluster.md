---
title: "Connect to a Postgres Cluster"
date:
draft: false
weight: 30
---

It's one thing to [create a Postgres cluster]({{< relref "./create-cluster.md" >}}); it's another thing to connect to it. Let's explore how PGO makes it possible to connect to a Postgres cluster!

## Background: Services, Secrets, and TLS

PGO creates a series of Kubernetes [Services](https://kubernetes.io/docs/concepts/services-networking/service/) to provide stable endpoints for connecting to your Postgres databases. These endpoints make it easy to provide a consistent way for your application to maintain connectivity to your data. To inspect what services are available, you can run the following command:

```
kubectl -n postgres-operator get svc --selector=postgres-operator.crunchydata.com/cluster=hippo
```

will yield something similar to:

```
NAME              TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)    AGE
hippo-ha          ClusterIP   10.103.73.92   <none>        5432/TCP   3h14m
hippo-ha-config   ClusterIP   None           <none>        <none>     3h14m
hippo-pods        ClusterIP   None           <none>        <none>     3h14m
hippo-primary     ClusterIP   None           <none>        5432/TCP   3h14m
hippo-replicas    ClusterIP   10.98.110.215  <none>        5432/TCP   3h14m
```

You do not need to worry about most of these Services, as they are used to help manage the overall health of your Postgres cluster. For the purposes of connecting to your database, the Service of interest is called `hippo-primary`. Thanks to PGO, you do not need to even worry about that, as that information is captured within a Secret!

When your Postgres cluster is initialized, PGO will bootstrap a database and Postgres user that your application can access. This information is stored in a Secret named with the pattern `<clusterName>-pguser-<userName>`. For our `hippo` cluster, this Secret is called `hippo-pguser-hippo`. This Secret contains the information you need to connect your application to your Postgres database:

- `user`: The name of the user account.
- `password`: The password for the user account.
- `dbname`: The name of the database that the user has access to by default.
- `host`: The name of the host of the database. This references the [Service](https://kubernetes.io/docs/concepts/services-networking/service/) of the primary Postgres instance.
- `port`: The port that the database is listening on.
- `uri`: A [PostgreSQL connection URI](https://www.postgresql.org/docs/current/libpq-connect.html#LIBPQ-CONNSTRING) that provides all the information for logging into the Postgres database.
- `jdbc-uri`: A [PostgreSQL JDBC connection URI](https://jdbc.postgresql.org/documentation/head/connect.html) that provides all the information for logging into the Postgres database via the JDBC driver.

All connections are over TLS. PGO provides its own certificate authority (CA) to allow you to securely connect your applications to your Postgres clusters. This allows you to use the [`verify-full` "SSL mode"](https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS) of Postgres, which provides eavesdropping protection and prevents MITM attacks. You can also choose to bring your own CA, which is described later in this tutorial in the [Customize Cluster]({{< relref "./customize-cluster.md" >}}) section.

### Modifying Service Type

By default, PGO deploys Services with the `ClusterIP` Service type. Based on how you want to expose your database, you may want to modify the Services to use a different [Service type](https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types).

You can modify the Services that PGO manages from the following attributes:

- `spec.service` - this manages the Service for connecting to a Postgres primary.
- `spec.proxy.pgBouncer.service` - this manages the Service for connecting to the PgBouncer connection pooler.

For example, to set the Postgres primary to use a `NodePort` service, you would add the following to your manifest:

```yaml
spec:
  service:
    type: NodePort
```

For our `hippo` cluster, you would see the Service type modification in the . For example:

```
kubectl -n postgres-operator get svc --selector=postgres-operator.crunchydata.com/cluster=hippo
```

will yield something similar to:

```
NAME              TYPE        CLUSTER-IP     EXTERNAL-IP   PORT(S)          AGE
hippo-ha          NodePort    10.96.17.210   <none>        5432:32751/TCP   2m37s
hippo-ha-config   ClusterIP   None           <none>        <none>           2m37s
hippo-pods        ClusterIP   None           <none>        <none>           2m37s
hippo-primary     ClusterIP   None           <none>        5432/TCP         2m37s
hippo-replicas    ClusterIP   10.96.151.53   <none>        5432/TCP         2m37s
```

(Note that if you are exposing your Services externally and are relying on TLS verification, you will need to use the [custom TLS]({{< relref "tutorial/customize-cluster.md" >}}#customize-tls) features of PGO).

## Connect an Application

For this tutorial, we are going to connect [Keycloak](https://www.keycloak.org/), an open source
identity management application. Keycloak can be deployed on Kubernetes and is backed by a Postgres
database. While we provide an [example of deploying Keycloak and a PostgresCluster](https://github.com/CrunchyData/postgres-operator-examples/tree/main/kustomize/keycloak)
in the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples)
repository, the manifest below deploys it using our `hippo` cluster that is already running:

```
kubectl apply --filename=- <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak
  namespace: postgres-operator
  labels:
    app.kubernetes.io/name: keycloak
spec:
  selector:
    matchLabels:
      app.kubernetes.io/name: keycloak
  template:
    metadata:
      labels:
        app.kubernetes.io/name: keycloak
    spec:
      containers:
      - image: quay.io/keycloak/keycloak:latest
        name: keycloak
        env:
        - name: DB_VENDOR
          value: "postgres"
        - name: DB_ADDR
          valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: host } }
        - name: DB_PORT
          valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: port } }
        - name: DB_DATABASE
          valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: dbname } }
        - name: DB_USER
          valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: user } }
        - name: DB_PASSWORD
          valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: password } }
        - name: KEYCLOAK_USER
          value: "admin"
        - name: KEYCLOAK_PASSWORD
          value: "admin"
        - name: PROXY_ADDRESS_FORWARDING
          value: "true"
        ports:
        - name: http
          containerPort: 8080
        - name: https
          containerPort: 8443
        readinessProbe:
          httpGet:
            path: /auth/realms/master
            port: 8080
      restartPolicy: Always
EOF
```

Notice this part of the manifest:

```
- name: DB_ADDR
  valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: host } }
- name: DB_PORT
  valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: port } }
- name: DB_DATABASE
  valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: dbname } }
- name: DB_USER
  valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: user } }
- name: DB_PASSWORD
  valueFrom: { secretKeyRef: { name: hippo-pguser-hippo, key: password } }
```

The above manifest shows how all of these values are derived from the `hippo-pguser-hippo` Secret. This means that we do not need to know any of the connection credentials or have to insecurely pass them around -- they are made directly available to the application!

Using this method, you can tie application directly into your GitOps pipeline that connect to Postgres without any prior knowledge of how PGO will deploy Postgres: all of the information your application needs is propagated into the Secret!

## Next Steps

Now that we have seen how to connect an application to a cluster, let's learn how to create a [high availability Postgres]({{< relref "./high-availability.md" >}}) cluster!
