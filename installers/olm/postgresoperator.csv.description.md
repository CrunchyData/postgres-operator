Crunchy PostgreSQL for Kubernetes lets you run your own production-grade PostgreSQL-as-a-Service on Kubernetes!

Powered by the Crunchy [PostgreSQL Operator](https://github.com/CrunchyData/postgres-operator), Crunchy PostgreSQL
for Kubernetes automates and simplifies deploying and managing open source PostgreSQL clusters on Kubernetes by providing the
essential features you need to keep your PostgreSQL clusters up and running, including:

- **PostgreSQL Cluster Provisioning**: [Create, Scale, & Delete PostgreSQL clusters with ease][provisioning],
while fully customizing your Pods and PostgreSQL configuration!
- **High-Availability**: Safe, automated failover backed by a [distributed consensus based high-availability solution][high-availability].
Uses [Pod Anti-Affinity][anti-affinity] to help resiliency; you can configure how aggressive this can be!
Failed primaries automatically heal, allowing for faster recovery time. You can even create regularly scheduled
backups as well and set your backup retention policy
- **Disaster Recovery**: Backups and restores leverage the open source [pgBackRest][] utility
and [includes support for full, incremental, and differential backups as well as efficient delta restores][disaster-recovery].
Set how long you want your backups retained for. Works great with very large databases!
- **Monitoring**: Track the health of your PostgreSQL clusters using the open source [pgMonitor][] library.
- **Clone**: Create new clusters from your existing clusters with a simple [`pgo clone`][pgo-clone] command.
- **Full Customizability**: Crunchy PostgreSQL for Kubernetes makes it easy to get your own PostgreSQL-as-a-Service up and running on
and lets make further enhancements to customize your deployments, including:
  - Selecting different storage classes for your primary, replica, and backup storage
  - Select your own container resources class for each PostgreSQL cluster deployment; differentiate between resources applied for primary and replica clusters!
  - Use your own container image repository, including support `imagePullSecrets` and private repositories
  - Bring your own trusted certificate authority (CA) for use with the Operator API server
  - Override your PostgreSQL configuration for each cluster

and much more!

[anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity
[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/
[pgo-clone]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/reference/pgo_clone/
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/provisioning/

[pgBackRest]: https://www.pgbackrest.org
[pgMonitor]: https://github.com/CrunchyData/pgmonitor

## Before You Begin

There are several manual steps that the cluster administrator must perform prior to installing the operator. The
operator must be provided with an initial configuration to run in the cluster, as well as certificates and
credentials that need to be generated.

Start by cloning the operator repository locally.

```
git clone -b v${PGO_VERSION} https://github.com/CrunchyData/postgres-operator.git
cd postgres-operator
```

### PostgreSQL Operator Configuration

Edit `conf/postgres-operator/pgo.yaml` to configure the operator deployment. Look over all of the options and make any
changes necessary for your environment.

#### Image

Update the `CCPImageTag` tag to configure the PostgreSQL image being used, updating for the version of PostgreSQL as needed.

```
CCPImageTag:  ${CCP_IMAGE_TAG}
```

#### Storage

Configure the backend storage for the Persistent Volumes used by each PostgreSQL cluster. Depending on the type of persistent
storage you wish to make available, adjust the `StorageClass` as necessary. For example, to deploy on AWS using `gp2`, you
would set the following:

```
storageos:
  AccessMode:  ReadWriteOnce
  Size:  1G
  StorageType:  dynamic
  StorageClass:  gp2
  Fsgroup:  26
```

Once the storage backend is defined, enable the new storage option as needed.

```
PrimaryStorage: storageos
ReplicaStorage: storageos
BackrestStorage: storageos
```

### Certificates

You will need to either generate new TLS certificates or use existing certificates for the operator API.

You can generate new self-signed certificates using scripts in the operator repository.

```
export PGOROOT=$(pwd)
cd $PGOROOT/deploy
$PGOROOT/deploy/gen-api-keys.sh
$PGOROOT/deploy/gen-sshd-keys.sh
cd $PGOROOT
```

### Configuration and Secrets

Once the configuration changes have been updated and certificates are in place, we can save the information to the cluster.

Create the pgo namespace if it does not exist already. This single namespace is where the operator should be deployed to. PostgreSQL clusters will also be deployed here.

```
kubectl create namespace pgo
```

Create the `pgo-backrest-repo-config` Secret that is used by the operator.

```
kubectl create secret generic -n pgo pgo-backrest-repo-config \
  --from-file=config=$PGOROOT/conf/pgo-backrest-repo/config \
  --from-file=sshd_config=$PGOROOT/conf/pgo-backrest-repo/sshd_config \
  --from-file=aws-s3-credentials.yaml=$PGOROOT/conf/pgo-backrest-repo/aws-s3-credentials.yaml \
  --from-file=aws-s3-ca.crt=$PGOROOT/conf/pgo-backrest-repo/aws-s3-ca.crt
```

Create the `pgo-auth-secret` Secret that is used by the operator.

```
kubectl create secret generic -n pgo pgo-auth-secret \
  --from-file=server.crt=$PGOROOT/conf/postgres-operator/server.crt \
  --from-file=server.key=$PGOROOT/conf/postgres-operator/server.key
```

Install the bootstrap credentials:

```
$PGOROOT/deploy/install-bootstrap-creds.sh
```

Remove existing credentials for pgo-apiserver TLS REST API, if they exist.

```
kubectl delete secret -n pgo tls pgo.tls
```

Create credentials for pgo-apiserver TLS REST API
```
kubectl create secret -n pgo tls pgo.tls \
  --key=$PGOROOT/conf/postgres-operator/server.key \
  --cert=$PGOROOT/conf/postgres-operator/server.crt
```

Create the `pgo-config` ConfigMap that is used by the operator.

```
kubectl create configmap -n pgo pgo-config \
  --from-file=$PGOROOT/conf/postgres-operator
```

Once these resources are in place, the operator can be installed into the cluster.

## After You Install

Once the operator is installed in the cluster, you will need to perform several steps to enable usage.

### Service

```
kubectl expose deployment -n pgo postgres-operator --type=LoadBalancer
```

For the pgo client to communicate with the operator, it needs to know where to connect.
Export the service URL as `PGO_APISERVER_URL` in the shell.

```
export PGO_APISERVER_URL=https://<url of exposed service>:8443
```

### Security

When postgres operator deploys, it creates a set of certificates the pgo client will need to communicate.

### Client Certificates

Copy the client certificates from the apiserver to the local environment - we use /tmp for this example.

```
kubectl cp <pgo-namespace>/<postgres-operator-pod>:/tmp/server.key /tmp/server.key -c apiserver
kubectl cp <pgo-namespace>/<postgres-operator-pod>:/tmp/server.crt /tmp/server.crt -c apiserver
```

Configure the shell for the pgo command line to use the certificates

```
export PGO_CA_CERT=/tmp/server.crt
export PGO_CLIENT_CERT=/tmp/server.crt
export PGO_CLIENT_KEY=/tmp/server.key
```
