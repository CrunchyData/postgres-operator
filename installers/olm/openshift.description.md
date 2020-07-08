Crunchy PostgreSQL for OpenShift lets you run your own production-grade PostgreSQL-as-a-Service on OpenShift!

Powered by the Crunchy [PostgreSQL Operator](https://github.com/CrunchyData/postgres-operator), Crunchy PostgreSQL
for OpenShift automates and simplifies deploying and managing open source PostgreSQL clusters on OpenShift by providing the
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
- **Full Customizability**: Crunchy PostgreSQL for OpenShift makes it easy to get your own PostgreSQL-as-a-Service up and running on
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

There are a few manual steps that the cluster administrator must perform prior to installing the PostgreSQL Operator.
At the very least, it must be provided with an initial configuration.

First, select a namespace in which to install the PostgreSQL Operator. PostgreSQL clusters will also be deployed here.
If it does not exist, create it now.

```
export PGO_OPERATOR_NAMESPACE=pgo
oc create namespace "$PGO_OPERATOR_NAMESPACE"
```

Next, clone the PostgreSQL Operator repository locally.

```
git clone -b v${PGO_VERSION} https://github.com/CrunchyData/postgres-operator.git
cd postgres-operator
```

### Security

For the PostgreSQL Operator and PostgreSQL clusters to run in the recommended `restricted` [Security Context Constraint][],
edit `conf/postgres-operator/pgo.yaml` and set `DisableFSGroup` to `true`.

[Security Context Constraint]: https://docs.openshift.com/container-platform/latest/authentication/managing-security-context-constraints.html

### PostgreSQL Operator Configuration

Edit `conf/postgres-operator/pgo.yaml` to configure the deployment. Look over all of the options and make any
changes necessary for your environment. A [full description of each option][pgo-yaml-reference] is available in the documentation.

[pgo-yaml-reference]: https://access.crunchydata.com/documentation/postgres-operator/${PGO_VERSION}/configuration/pgo-yaml-configuration/

When the file is ready, upload the entire directory to the `pgo-config` ConfigMap.

```
oc -n "$PGO_OPERATOR_NAMESPACE" create configmap pgo-config \
  --from-file=./conf/postgres-operator
```

### Secrets

Configure pgBackRest for your environment. If you do not plan to use AWS S3 to store backups, you can omit
the `aws-s3` keys below.

```
oc -n "$PGO_OPERATOR_NAMESPACE" create secret generic pgo-backrest-repo-config \
  --from-file=config=./conf/pgo-backrest-repo/config \
  --from-file=sshd_config=./conf/pgo-backrest-repo/sshd_config \
  --from-file=aws-s3-ca.crt=./conf/pgo-backrest-repo/aws-s3-ca.crt \
  --from-literal=aws-s3-key="<your-aws-s3-key>" \
  --from-literal=aws-s3-key-secret="<your-aws-s3-key-secret>"
```

### Certificates (optional)

The PostgreSQL Operator has an API that uses TLS to communicate securely with clients. If you have
a certificate bundle validated by your organization, you can install it now.  If not, the API will
automatically generate and use a self-signed certificate.

```
oc -n "$PGO_OPERATOR_NAMESPACE" create secret tls pgo.tls \
  --cert=/path/to/server.crt \
  --key=/path/to/server.key
```

Once these resources are in place, the PostgreSQL Operator can be installed into the cluster.


## After You Install

Once the PostgreSQL Operator is installed in your OpenShift cluster, you will need to do a few things
to use the [PostgreSQL Operator Client][pgo-client].

[pgo-client]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/

Install the first set of client credentials and download the `pgo` binary and client certificates.

```
PGO_CMD=oc ./deploy/install-bootstrap-creds.sh
PGO_CMD=oc ./installers/kubectl/client-setup.sh
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
