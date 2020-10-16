# crunchycrdcluster

This is a working example of how to create a cluster via the crd workflow
using a helm chart

## Assumptions
This example assumes you have the Crunchy PostgreSQL Operator installed
in namespace called pgo.  

## Helm
Helm will also need to be installed for this example to run

## Documenation
Please see the documentation for more guidance using custom resources:

https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/


## Example set up and execution
create a cert directy and generate certs
```
cd postgres-operator/examples/helm/crunchycrdcluster

mkdir cert

# this variable is the name of the cluster being created
export pgo_cluster_name=hippo

# generate a SSH public/private keypair for use by pgBackRest
ssh-keygen -t ed25519 -N '' -f "${pgo_cluster_name}-key"

```
The cluster can bedeployed in the pgo namespace but for this example
we are going to create a Crunchy PostgreSql cluster in a namespace 
called test1
```
pgo create namespace test1
```

run the following command to install via helm from the 
crunchycrdcluster directory
```
cd postgres-operator/examples/helm/crunchycrdcluster
```
you can run a dry run first with debug if you like to verify everthing
is set correctly. Then after everything looks good run the install command
with out the flags
```
helm install --dry-run --debug crunchycrdcluster . -n test1

helm install crunchycrdcluster . -n test1
```
## Verify
Now you can your Hippo cluster has deployed into the test1
namespace by running these few commands

```
kubectl get all -n test1

pgo test hippo -n test1

pgo show cluster hippo -n test1
```


