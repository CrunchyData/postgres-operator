Crunchy PostgreSQL for Kubernetes lets you run your own production-grade PostgreSQL-as-a-Service on Kubernetes!

Powered by the Crunchy [PostgreSQL Operator](https://github.com/CrunchyData/postgres-operator), Crunchy PostgreSQL
for Kubernetes automates and simplifies deploying and managing open source PostgreSQL clusters on Kubernetes by
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
- **Full Customizability**: Crunchy PostgreSQL for Kubernetes makes it easy to get your own PostgreSQL-as-a-Service up and running on
  and lets make further enhancements to customize your deployments, including:
    - Selecting different storage classes for your primary, replica, and backup storage
    - Select your own container resources class for each PostgreSQL cluster deployment; differentiate between resources applied for primary and replica clusters!
    - Use your own container image repository, including support `imagePullSecrets` and private repositories
    - Bring your own trusted certificate authority (CA) for use with the Operator API server
    - Override your PostgreSQL configuration for each cluster

and much more!

[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/
[pgo-create-cluster]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/reference/pgo_create_cluster/
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/provisioning/

[k8s-anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity

[pgBackRest]: https://www.pgbackrest.org
[pgMonitor]: https://github.com/CrunchyData/pgmonitor


## Before You Begin

There are a few manual steps that the cluster administrator must perform prior to installing the PostgreSQL Operator.
At the very least, it must be provided with an initial configuration.

First, select a namespace in which to install the PostgreSQL Operator. PostgreSQL clusters will also be deployed here.
If it does not exist, create it now.

```
export PGO_OPERATOR_NAMESPACE=pgo
kubectl create namespace "$PGO_OPERATOR_NAMESPACE"
```

### Secrets (optional)

If you plan to use AWS S3 to store backups, you can configure your environment to automatically provide your AWS S3 credentials to all newly created PostgreSQL clusters:

```
kubectl -n "$PGO_OPERATOR_NAMESPACE" create secret generic pgo-backrest-repo-config \
  --from-literal=aws-s3-key="<your-aws-s3-key>" \
  --from-literal=aws-s3-key-secret="<your-aws-s3-key-secret>"
kubectl -n "$PGO_OPERATOR_NAMESPACE" label secret pgo-backrest-repo-config vendor=crunchydata
```

### Certificates (optional)

The PostgreSQL Operator has an API that uses TLS to communicate securely with clients. If one is not provided, the API will automatically generated one for you.

If you have a certificate bundle validated by your organization, you can install it now.

```
kubectl -n "$PGO_OPERATOR_NAMESPACE" create secret tls pgo.tls \
  --cert=/path/to/server.crt \
  --key=/path/to/server.key
```

Once these resources are in place, the PostgreSQL Operator can be installed into the cluster.


## After You Install

Once the PostgreSQL Operator is installed in your Kubernetes cluster, you will need to do a few things
to use the [PostgreSQL Operator Client][pgo-client].

[pgo-client]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/

Install the first set of client credentials and download the `pgo` binary and client certificates.

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v${PGO_VERSION}/deploy/install-bootstrap-creds.sh > install-bootstrap-creds.sh
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v${PGO_VERSION}/installers/kubectl/client-setup.sh > client-setup.sh

chmod +x install-bootstrap-creds.sh client-setup.sh

PGO_CMD=kubectl ./install-bootstrap-creds.sh
PGO_CMD=kubectl ./client-setup.sh
```

The client needs to be able to reach the PostgreSQL Operator API from outside the Kubernetes cluster.
Create an external service or forward a port locally.

```
kubectl -n "$PGO_OPERATOR_NAMESPACE" expose deployment postgres-operator --type=LoadBalancer

export PGO_APISERVER_URL="https://$(
  kubectl -n "$PGO_OPERATOR_NAMESPACE" get service postgres-operator \
    -o jsonpath="{.status.loadBalancer.ingress[*]['ip','hostname']}"
):8443"
```
_or_
```
kubectl -n "$PGO_OPERATOR_NAMESPACE" port-forward deployment/postgres-operator 8443

export PGO_APISERVER_URL="https://127.0.0.1:8443"
```

Verify connectivity using the `pgo` command.

```
pgo version
# pgo client version ${PGO_VERSION}
# pgo-apiserver version ${PGO_VERSION}
```
