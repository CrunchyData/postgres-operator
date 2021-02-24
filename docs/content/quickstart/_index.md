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

export PGO_IMAGE_TAG="centos8-v1alpha1"
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

Get the name of the primary PostgreSQL pod

```
PG_CLUSTER_PRIMARY_POD=$(kubectl get pod -n postgres-operator -o name \
  -l postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master)
```

Create a user and a database:

```
kubectl exec -it -n postgres-operator "${PG_CLUSTER_PRIMARY_POD}" -- psql -c "SET password_encryption TO 'scram-sha-256'; CREATE ROLE hippo LOGIN PASSWORD 'datalake'"
kubectl exec -it -n postgres-operator "${PG_CLUSTER_PRIMARY_POD}" -- psql -c "CREATE DATABASE hippo OWNER hippo"
kubectl exec -it -n postgres-operator "${PG_CLUSTER_PRIMARY_POD}" -- psql hippo -c "CREATE SCHEMA hippo AUTHORIZATION hippo"
```

In a new terminal, create a port forward:

```
PG_CLUSTER_PRIMARY_POD=$(kubectl get pod -n postgres-operator -o name \
  -l postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master)
kubectl -n postgres-operator port-forward "${PG_CLUSTER_PRIMARY_POD}" 5432:5432
```

Connect to the Postgres cluster:

```
PGPASSWORD=datalake psql -h localhost -U hippo hippo
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

## Delete a Postgres Cluster

```
kubectl delete -k examples/postgrescluster
```
