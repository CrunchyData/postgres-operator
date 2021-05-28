# create-cluster

This is a working example of how to create a cluster via the crd workflow
using a [Helm](https://helm.sh/) chart.

## Prerequisites

### Postgres Operator

This example assumes you have the Crunchy PostgreSQL Operator installed
in a namespace called `pgo`.  

### Helm

Helm will also need to be installed for this example to run

## Documentation

Please see the documentation for more guidance using custom resources:

https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/

## Setup

If you are running Postgres Operator 4.5.3 or later, you can skip the below
step.

### Before 4.5.3

```
cd postgres-operator/examples/helm/create-cluster

mkdir certs
cd certs

# this variable is the name of the cluster being created
export pgo_cluster_name=hippo

# generate a SSH public/private keypair for use by pgBackRest
ssh-keygen -t ed25519 -N '' -f "${pgo_cluster_name}-key"
```

## Running the Example

For this example we will deploy the cluster into the `pgo` namespace where the
Postgres Operator is installed and running.

Return to the `create-cluster` directory:

```
cd postgres-operator/examples/helm/create-cluster
```

The following commands will allow you to execute a dry run first with debug
if you want to verify everything is set correctly. Then after everything looks
good run the install command with out the flags:

```
helm install --dry-run --debug postgres-operator-create-cluster . -n pgo
helm install postgres-operator-create-cluster . -n pgo
```

## Verify

Now you can your Hippo cluster has deployed into the pgo namespace by running
these few commands:

```
kubectl get all -n pgo

pgo test hippo -n pgo

pgo show cluster hippo -n pgo
```

## NOTE

As of operator version 4.5.0 when using helm uninstall you will have to manually
clean up some left over artifacts after running the uninstall.
