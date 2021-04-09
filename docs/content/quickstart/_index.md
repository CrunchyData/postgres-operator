---
title: "Quickstart"
date:
draft: false
weight: 10
---

# PostgreSQL Operator Quickstart

Can't wait to try out the PostgreSQL Operator? Let us show you the quickest possible path to getting up and running.

## Prerequisites

Please be sure you have the following utilities installed on your host machine:

- `kubectl`

### Temporary

- [`git`](https://git-scm.com/)
- [`go`](https://golang.org/) 1.15+
- [`buildah`](https://buildah.io/)

## Build (Temporary) and Install the Postgres Operator

Download the PostgreSQL Operator codebase:

```
git clone --depth 1 git@github.com:CrunchyData/savannah.git
cd savannah
```

Check out the savannah branch:

```
git checkout savannah
```

Set an environmental variable to be the registry you are pushing to, e.g.:

```
export PGO_IMAGE_PREFIX=registry.developers.crunchydata.com/crunchydata
```

Build the Postgres Operator:

```
export PGOROOT=`pwd`

export PGO_IMAGE_TAG="centos8-v1beta1"
GOOS=linux GOARCH=amd64 make build-postgres-operator
make postgres-operator-image
buildah push "${PGO_IMAGE_PREFIX}/postgres-operator:${PGO_IMAGE_TAG}"
```

Create the namespaces (i.e. `postgres-operator`):

```
make createnamespaces
```

Install the Postgres Operator RBAC:

```
make install
```

Modify the file in `config/default/kustomization.yaml` to reference the image you pushed to your repository, e.g.:

```
namespace: postgres-operator

commonLabels:
  postgres-operator.crunchydata.com/control-plane: postgres-operator

bases:
- ../crd
- ../rbac
- ../manager

images:
- name: postgres-operator
  newName: ${PGO_IMAGE_PREFIX}/postgres-operator
  newTag: ${PGO_IMAGE_TAG}
```

Deploy the Postgres Operator:

```
make deploy
```

## Create a Postgres Cluster

You can create a Postgres Cluster using the example Kustomization file:

```
kubectl apply -k examples/postgrescluster
```

## Connect to the Postgres cluster

As part of creating a Postgres cluster, the Postgres Operator creates a PostgreSQL user account. The credentials for this account are stored in a Secret that has the name `<clusterName>-pguser`.

Within this Secret are attributes that provide information to let you log into the PostgreSQL cluster. These include:

- `user`: The name of the user account.
- `password`: The password for the user account.
- `dbname`: The name of the database that the user has access to by default.
- `host`: The name of the host of the database. This references the [Service](https://kubernetes.io/docs/concepts/services-networking/service/) of the primary Postgres instance.
- `port`: The port that the database is listening on.
- `uri`: A [PostgreSQL connection URI](https://www.postgresql.org/docs/current/libpq-connect.html#id-1.7.3.8.3.6) that provides all the information for logging into the Postgres database.

### Connect via `psql` in the Terminal

#### Connect Directly

If you are on the same network as your PostgreSQL cluster, you can connect directly to it using the following command:

```
psql $(kubectl -n postgres-operator get secrets hippo-pguser -o jsonpath='{.data.uri}' | base64 -d)
```

#### Connect Using a Port-Forward

In a new terminal, create a port forward:

```
PG_CLUSTER_PRIMARY_POD=$(kubectl get pod -n postgres-operator -o name \
  -l postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master)
kubectl -n postgres-operator port-forward "${PG_CLUSTER_PRIMARY_POD}" 5432:5432
```

Establish a connection to the PostgreSQL cluster.

```
PG_CLUSTER_USER_SECRET_NAME=hippo-pguser

PGPASSWORD=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o jsonpath="{.data.password}" | base64 -d) \
PGUSER=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o jsonpath="{.data.user}" | base64 -d) \
PGDBNAME=$(kubectl get secrets -n postgres-operator "${PG_CLUSTER_USER_SECRET_NAME}" -o jsonpath="{.data.dbname}" | base64 -d) \
psql -h localhost
```

### Connect an Application

The information provided in the user Secret will allow you to connect an application directly to your PostgreSQL database.

For example, let's connect [Keycloak](https://www.keycloak.org/). Keycloak is a popular open source identity management tool that is backed by a PostgreSQL database. Using the `hippo` cluster we created, we can deploy the the following manifest file:

```
cat <<EOF >> keycloak.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: keycloak
  namespace: postgres-operator
  labels:
    app: keycloak
spec:
  selector:
    matchLabels:
      app: keycloak
  template:
    metadata:
      labels:
        app: keycloak
    spec:
      containers:
      - image: quay.io/keycloak/keycloak:latest
        name: keycloak
        env:
        - name: DB_VENDOR
          value: "postgres"
        - name: DB_ADDR
          valueFrom:
            secretKeyRef:
              name: hippo-pguser
              key: host
        - name: DB_PORT
          valueFrom:
            secretKeyRef:
              name: hippo-pguser
              key: port
        - name: DB_DATABASE
          valueFrom:
            secretKeyRef:
              name: hippo-pguser
              key: dbname
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: hippo-pguser
              key: user
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: hippo-pguser
              key: password
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

kubectl apply -f keycloak.yaml
```

## Scale a Postgres Cluster

There are two ways to scale a cluster:

### Method 1

In the `examples/postgrescluster/postgrescluster.yaml` file, add `replicas: 2` to one of the instances entry.

Then run:

```
kubectl apply -k examples/postgrescluster
```

### Method 2

In the `examples/postgrescluster/postgrescluster.yaml` file, add a new array item name `instance2`, e.g.:

```
spec:
  instances:
    - name: instance1
    - name: instance2
```

## Add Custom Certificates to a Postgres Cluster

Create a Secret containing your TLS certificate, TLS key and the Certificate Authority certificate such that
the data of the secret in the `postgrescluster`'s namespace that contains the three values, similar to

```
data:
  ca.crt: <value>
  tls.crt: <value>
  tls.key: <value>
```

Then, in `examples/postgrescluster/postgrescluster.yaml`, add the following:

```
spec:
  customTLSSecret: 
    name: customcert
```
where 'customcert' is the name of your created secret. Your cluster will now use the provided certificate in place of
a dynamically generated one. Please note, if `CustomTLSSecret` is provided, `CustomReplicationClientTLSSecret` MUST 
be provided and the `ca.crt` provided must be the same in both.

In cases where the key names cannot be controlled, the item key and path values can be specified explicitly, as shown 
below:
```
spec:
  customTLSSecret: 
    name: customcert
    items: 
      - key: <tls.crt key>
        path: tls.crt
      - key: <tls.key key>
        path: tls.key
      - key: <ca.crt key>
        path: ca.crt
```
The Common Name setting will be expected to include the primary service name. For a `postgrescluster` named 'hippo', the
expected primary service name will be `hippo-primary`.

## Add Replication Client Certificates to a Postgres Cluster

Similar to the above, to provide your own TLS client certificates for use by the replication system account,
you will need to create a Secret containing your TLS certificate, TLS key and the Certificate Authority certificate
such that the data of the secret in the `postgrescluster`'s namespace that contains the three values, similar to

```
data:
  ca.crt: <value>
  tls.crt: <value>
  tls.key: <value>
```

Then, in `examples/postgrescluster/postgrescluster.yaml`, add the following:

```
spec:
  customReplicationTLSSecret: 
    name: customReplicationCert
```
where 'customReplicationCert' is the name of your created secret. Your cluster will now use the provided certificate in place of
a dynamically generated one. Please note, if `CustomReplicationClientTLSSecret` is provided, `CustomTLSSecret`
MUST be provided and the `ca.crt` provided must be the same in both.

In cases where the key names cannot be controlled, the item key and path values can be specified explicitly, as shown 
below:
```
spec:
  customReplicationTLSSecret: 
    name: customReplicationCert
    items: 
      - key: <tls.crt key>
        path: tls.crt
      - key: <tls.key key>
        path: tls.key
      - key: <ca.crt key>
        path: ca.crt
```

The Common Name setting will be expected to include the replication user name, `_crunchyrepl`.

For more information regarding secret projections, please see
https://k8s.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths

## Delete a Postgres Cluster

```
kubectl delete -k examples/postgrescluster
```
