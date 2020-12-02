# create cluster
This is a working example creating multiple clusters via the crd workflow using
kustomize.

## Prerequisites

### Postgres Operator

This example assumes you have the Crunchy PostgreSQL Operator installed
in a namespace called `pgo`.

### Kustomize

Install the latest [kustomize](https://kubectl.docs.kubernetes.io/installation/kustomize/) version available.  Kustomise is availble in kubectl but it will not be the latest version.

## Documenation
Please see the [documentation](https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/) for more guidance using custom resources.

## Example set up and execution
Navigate to the createcluster directory under the examples/kustomize directory
```
cd ./examples/kustomize/createcluster/
```


