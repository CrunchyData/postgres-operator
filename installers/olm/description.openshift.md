Crunchy PostgreSQL for OpenShift lets you run your own production-grade PostgreSQL-as-a-Service on OpenShift!

Powered by the Crunchy [PostgreSQL Operator](https://github.com/CrunchyData/postgres-operator), Crunchy PostgreSQL
for OpenShift automates and simplifies deploying and managing open source PostgreSQL clusters on OpenShift by
providing the essential features you need to keep your PostgreSQL clusters up and running, including:

- **PostgreSQL Cluster Provisioning**: [Create, Scale, & Delete PostgreSQL clusters with ease][provisioning],
  while fully customizing your Pods and PostgreSQL configuration!
- **High-Availability**: Safe, automated failover backed by a [distributed consensus based high-availability solution][high-availability].
  Uses [Pod Anti-Affinity][k8s-anti-affinity] to help resiliency; you can configure how aggressive this can be!
  Failed primaries automatically heal, allowing for faster recovery time. You can even create regularly scheduled
  backups as well and set your backup retention policy
- **Disaster Recovery**: Backups and restores leverage the open source [pgBackRest][] utility
  and [includes support for full, incremental, and differential backups as well as efficient delta restores][disaster-recovery].
  Set how long you want your backups retained for. Works great with very large databases!
- **Monitoring**: Track the health of your PostgreSQL clusters using the open source [pgMonitor][] library.
- **Clone**: Create new clusters from your existing clusters or backups with a single [`pgo create cluster --restore-from`][pgo-create-cluster] command.
- **TLS**: Secure communication between your applications and data servers by [enabling TLS for your PostgreSQL servers][pgo-task-tls], including the ability to enforce that all of your connections to use TLS.
- **Connection Pooling**: Use [pgBouncer][] for connection pooling
- **Affinity and Tolerations**: Have your PostgreSQL clusters deployed to [Kubernetes Nodes][k8s-nodes] of your preference with [node affinity][high-availability-node-affinity], or designate which nodes Kubernetes can schedule PostgreSQL instances to with Kubernetes [tolerations][high-availability-tolerations].
- **Full Customizability**: Crunchy PostgreSQL for OpenShift makes it easy to get your own PostgreSQL-as-a-Service up and running on
  and lets make further enhancements to customize your deployments, including:
    - Selecting different storage classes for your primary, replica, and backup storage
    - Select your own container resources class for each PostgreSQL cluster deployment; differentiate between resources applied for primary and replica clusters!
    - Use your own container image repository, including support `imagePullSecrets` and private repositories
    - Bring your own trusted certificate authority (CA) for use with the Operator API server
    - Override your PostgreSQL configuration for each cluster

and much more!

[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/
[high-availability-node-affinity]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/#node-affinity
[high-availability-tolerations]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/#tolerations
[pgo-create-cluster]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/reference/pgo_create_cluster/
[pgo-task-tls]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorial/tls/
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/provisioning/

[k8s-anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity
[k8s-nodes]: https://kubernetes.io/docs/concepts/architecture/nodes/

[pgBackRest]: https://www.pgbackrest.org
[pgBouncer]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorial/pgbouncer/
[pgMonitor]: https://github.com/CrunchyData/pgmonitor

## Pre-Installation

There are a few manual steps that the cluster administrator must perform prior to installing the PostgreSQL Operator.
At the very least, it must be provided with an initial configuration.

First, select a namespace in which to install the PostgreSQL Operator. PostgreSQL clusters will also be deployed here.
If it does not exist, create it now.

```
export PGO_OPERATOR_NAMESPACE=pgo
oc create namespace "$PGO_OPERATOR_NAMESPACE"
```

### Security

For the PostgreSQL Operator and PostgreSQL clusters to run in the recommended `restricted` [Security Context Constraint][],
edit `conf/postgres-operator/pgo.yaml` and set `DisableFSGroup` to `true`.

[Security Context Constraint]: https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html

### Secrets (optional)

If you plan to use AWS S3 to store backups, you can configure your environment to automatically provide your AWS S3 credentials to all newly created PostgreSQL clusters:

```
oc -n "$PGO_OPERATOR_NAMESPACE" create secret generic pgo-backrest-repo-config \
  --from-literal=aws-s3-key="<your-aws-s3-key>" \
  --from-literal=aws-s3-key-secret="<your-aws-s3-key-secret>"
oc -n "$PGO_OPERATOR_NAMESPACE" label secret pgo-backrest-repo-config vendor=crunchydata
```

### Certificates (optional)

The PostgreSQL Operator has an API that uses TLS to communicate securely with clients. If one is not provided, the API will automatically generated one for you.

If you have a certificate bundle validated by your organization, you can install it now.

```
oc -n "$PGO_OPERATOR_NAMESPACE" create secret tls pgo.tls \
  --cert=/path/to/server.crt \
  --key=/path/to/server.key
```

Once these resources are in place, the PostgreSQL Operator can be installed into the cluster.

## Installation

You can now go ahead and install the PostgreSQL Operator from OperatorHub.

### Security

For the PostgreSQL Operator and PostgreSQL clusters to run in the recommended `restricted` [Security Context Constraint][],
edit the ConfigMap `pgo-config`, find the `pgo.yaml` entry, and set `DisableFSGroup` to `true`.

[Security Context Constraint]: https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html

You will have to scale the `postgres-operator` Deployment down and up for the above change to take effect:

```
oc -n pgo scale --replicas 0 deployment/postgres-operator
oc -n pgo scale --replicas 1 deployment/postgres-operator
```

## Post-Installation

### Tutorial

For a guide on how to perform many of the daily functions of the PostgreSQL Operator, we recommend that you read the [Postgres Operator tutorial][pgo-tutorial]

[pgo-tutorial]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorial/create-cluster/

However, the below guide will show you how to create a Postgres cluster from a custom resource or from using the `pgo-client`.

### Create a PostgreSQL Cluster from a Custom Resource

The fundamental workflow for interfacing with a PostgreSQL Operator Custom
Resource Definition is for creating a PostgreSQL cluster. There are several
that a PostgreSQL cluster requires to be deployed, including:

- Secrets
  - Information for setting up a pgBackRest repository
  - PostgreSQL superuser bootstrap credentials
  - PostgreSQL replication user bootstrap credentials
  - PostgresQL standard user bootstrap credentials

Additionally, if you want to add some of the other sidecars, you may need to
create additional secrets.

The good news is that if you do not provide these objects, the PostgreSQL
Operator will create them for you to get your Postgres cluster up and running!

The following goes through how to create a PostgreSQL cluster called
`hippo` by creating a new custom resource.

```
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo
# this variable is set to the location of your image repository
export cluster_image_prefix=registry.developers.crunchydata.com/crunchydata

cat <<-EOF > "${pgo_cluster_name}-pgcluster.yaml"
apiVersion: crunchydata.com/v1
kind: Pgcluster
metadata:
  annotations:
    current-primary: ${pgo_cluster_name}
  labels:
    crunchy-pgha-scope: ${pgo_cluster_name}
    deployment-name: ${pgo_cluster_name}
    name: ${pgo_cluster_name}
    pg-cluster: ${pgo_cluster_name}
    pgo-version: ${PGO_VERSION}
    pgouser: admin
  name: ${pgo_cluster_name}
  namespace: ${cluster_namespace}
spec:
  BackrestStorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ""
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  PrimaryStorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ${pgo_cluster_name}
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  ReplicaStorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ""
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  annotations: {}
  ccpimage: crunchy-postgres-ha
  ccpimageprefix: ${cluster_image_prefix}
  ccpimagetag: centos8-13.3-${PGO_VERSION}
  clustername: ${pgo_cluster_name}
  database: ${pgo_cluster_name}
  exporterport: "9187"
  limits: {}
  name: ${pgo_cluster_name}
  pgDataSource:
    restoreFrom: ""
    restoreOpts: ""
  pgbadgerport: "10000"
  pgoimageprefix: ${cluster_image_prefix}
  podAntiAffinity:
    default: preferred
    pgBackRest: preferred
    pgBouncer: preferred
  port: "5432"
  tolerations: []
  user: hippo
  userlabels:
    pgo-version: ${PGO_VERSION}
EOF

oc apply -f "${pgo_cluster_name}-pgcluster.yaml"
```

And that's all! The PostgreSQL Operator will go ahead and create the cluster.

If you have the PostgreSQL client `psql` installed on your host machine, you can
test connection to the PostgreSQL cluster using the following command:

```
# namespace that the cluster is running in
export PGO_OPERATOR_NAMESPACE=pgo
# name of the cluster
export pgo_cluster_name=hippo
# name of the user whose password we want to get
export pgo_cluster_username=hippo

# get the password of the user and set it to a recognized psql environmental variable
export PGPASSWORD=$(oc -n "${PGO_OPERATOR_NAMESPACE}" get secrets \
  "${pgo_cluster_name}-${pgo_cluster_username}-secret" -o "jsonpath={.data['password']}" | base64 -d)

# set up a port-forward either in a new terminal, or in the same terminal in the background:
oc -n pgo port-forward svc/hippo 5432:5432 &

psql -h localhost -U "${pgo_cluster_username}" "${pgo_cluster_name}"
```

### Create a PostgreSQL Cluster the `pgo` Client

Once the PostgreSQL Operator is installed in your OpenShift cluster, you will need to do a few things
to use the [PostgreSQL Operator Client][pgo-client].

[pgo-client]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/

Install the first set of client credentials and download the `pgo` binary and client certificates.

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v${PGO_VERSION}/deploy/install-bootstrap-creds.sh > install-bootstrap-creds.sh
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v${PGO_VERSION}/installers/kubectl/client-setup.sh > client-setup.sh

chmod +x install-bootstrap-creds.sh client-setup.sh

PGO_CMD=oc ./install-bootstrap-creds.sh
PGO_CMD=oc ./client-setup.sh
```

The client needs to be able to reach the PostgreSQL Operator API from outside the OpenShift cluster.
Create an external service or forward a port locally.

```
oc -n "$PGO_OPERATOR_NAMESPACE" expose deployment postgres-operator
oc -n "$PGO_OPERATOR_NAMESPACE" create route passthrough postgres-operator --service=postgres-operator

export PGO_APISERVER_URL="https://$(oc -n "$PGO_OPERATOR_NAMESPACE" get route postgres-operator -o jsonpath="{.spec.host}")"
```
_or_
```
oc -n "$PGO_OPERATOR_NAMESPACE" port-forward deployment/postgres-operator 8443

export PGO_APISERVER_URL="https://127.0.0.1:8443"
```

Verify connectivity using the `pgo` command.

```
pgo version
# pgo client version ${PGO_VERSION}
# pgo-apiserver version ${PGO_VERSION}
```


You can then create a cluster with the `pgo` client as simply as this:

```
pgo create cluster -n pgo hippo
```

The cluster may take a few moments to provision. You can verify that the cluster is up and running by using the `pgo test` command:

```
pgo test cluster -n pgo hippo
```

If you have the PostgreSQL client `psql` installed on your host machine, you can
test connection to the PostgreSQL cluster using the following command:

```
# namespace that the cluster is running in
export PGO_OPERATOR_NAMESPACE=pgo
# name of the cluster
export pgo_cluster_name=hippo
# name of the user whose password we want to get
export pgo_cluster_username=hippo

# get the password of the user and set it to a recognized psql environmental variable
export PGPASSWORD=$(kubectl -n "${PGO_OPERATOR_NAMESPACE}" get secrets \
  "${pgo_cluster_name}-${pgo_cluster_username}-secret" -o "jsonpath={.data['password']}" | base64 -d)

# set up a port-forward either in a new terminal, or in the same terminal in the background:
kubectl -n pgo port-forward svc/hippo 5432:5432 &

psql -h localhost -U "${pgo_cluster_username}" "${pgo_cluster_name}"
```
