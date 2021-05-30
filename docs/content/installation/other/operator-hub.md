---
title: "OperatorHub.io"
date:
draft: false
weight: 200
---

If your Kubernetes cluster is already running the [Operator Lifecycle Manager][OLM],
then PGO, the Postgres Operator from Crunchy Data, can be installed as part of [Crunchy PostgreSQL for Kubernetes][hub-listing]
that is available in OperatorHub.io.

[hub-listing]: https://operatorhub.io/operator/postgresql
[OLM]: https://olm.operatorframework.io/


## Before You Begin

There are a few manual steps that the cluster administrator must perform prior to installing PGO.
At the very least, it must be provided with an initial configuration.

First, make sure OLM and the OperatorHub.io catalog are installed by running
`kubectl get CatalogSources --all-namespaces`.  You should see something similar to the following:

```
NAMESPACE   NAME                    DISPLAY               TYPE   PUBLISHER
olm         operatorhubio-catalog   Community Operators   grpc   OperatorHub.io
```

Take note of the name and namespace above, you will need them later on.

Next, select a namespace in which to install the PostgreSQL Operator. PostgreSQL clusters will also be deployed here.
If it does not exist, create it now.

```
export PGO_OPERATOR_NAMESPACE=pgo
kubectl create namespace "$PGO_OPERATOR_NAMESPACE"
```

Next, clone the PostgreSQL Operator repository locally.

```
git clone -b v{{< param operatorVersion >}} https://github.com/CrunchyData/postgres-operator.git
cd postgres-operator
```

### Secrets

Configure pgBackRest for your environment. If you do not plan to use AWS S3 to store backups, you can omit
the `aws-s3` keys below.

```
kubectl -n "$PGO_OPERATOR_NAMESPACE" create secret generic pgo-backrest-repo-config \
  --from-file=./installers/ansible/roles/pgo-operator/files/pgo-backrest-repo/config \
  --from-file=./installers/ansible/roles/pgo-operator/files/pgo-backrest-repo/sshd_config \
  --from-file=./installers/ansible/roles/pgo-operator/files/pgo-backrest-repo/aws-s3-ca.crt \
  --from-literal=aws-s3-key="<your-aws-s3-key>" \
  --from-literal=aws-s3-key-secret="<your-aws-s3-key-secret>"
kubectl -n "$PGO_OPERATOR_NAMESPACE" label secret pgo-backrest-repo-config \
  vendor=crunchydata
```

### Certificates (optional)

PGO has an API that uses TLS to communicate securely with clients. If you have
a certificate bundle validated by your organization, you can install it now.  If not, the API will
automatically generate and use a self-signed certificate.

```
kubectl -n "$PGO_OPERATOR_NAMESPACE" create secret tls pgo.tls \
  --cert=/path/to/server.crt \
  --key=/path/to/server.key
```

Once these resources are in place, the PostgreSQL Operator can be installed into the cluster.


## Installation

Create an `OperatorGroup` and a `Subscription` in your chosen namespace.
Make sure the `source` and `sourceNamespace` match the CatalogSource from earlier.

```
kubectl -n "$PGO_OPERATOR_NAMESPACE" create -f- <<YAML
---
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: postgresql
spec:
  targetNamespaces: ["$PGO_OPERATOR_NAMESPACE"]

---
apiVersion: operators.coreos.com/v1alpha1
kind: Subscription
metadata:
  name: postgresql
spec:
  name: postgresql
  channel: stable
  source: operatorhubio-catalog
  sourceNamespace: olm
  startingCSV: postgresoperator.v{{< param operatorVersion >}}
YAML
```


## After You Install

Once PGO is installed in your Kubernetes cluster, you will need to do a few things
to use the [PostgreSQL Operator Client]({{< relref "/pgo-client/_index.md" >}}).

Install the first set of client credentials and download the `pgo` binary and client certificates.

```
PGO_CMD=kubectl ./deploy/install-bootstrap-creds.sh
PGO_CMD=kubectl ./installers/kubectl/client-setup.sh
```

The client needs to be able to reach the PGO API from outside the Kubernetes cluster.
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
# pgo client version {{< param operatorVersion >}}
# pgo-apiserver version {{< param operatorVersion >}}
```
