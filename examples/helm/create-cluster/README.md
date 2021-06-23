# create-cluster

This is a working example of how to create a cluster via the crd workflow
using a helm chart

## Assumptions
This example assumes you have the Crunchy PostgreSQL Operator installed
in a namespace called postgres-operator.  

## Helm
Helm will also need to be installed for this example to run.

## Documenation
Please see the documentation for more guidance using custom resources:

https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/



For this example we will deploy the cluster into the 
postgres-operator namespace where the operator is installed 
and running.

Return to the helm directory: 
```
cd postgres-operator/examples/helm
```

The following commands will allow you to execute a dry run first with debug 
if you want to verify everthing is set correctly. Then after everything looks good 
run the install command with out the flags
```
helm install --dry-run --debug -n postgres-operator postgres-cluster create-cluster

helm install -n postgres-operator postgres-cluster create-cluster
```
## Verify
Now you can verify your Hippo cluster has deployed into the postgres-operator
namespace by running these few commands

```
helm list
kubectl get all -n postgres-operator
```

## delete the hippo cluster
run the following helm command to delete the cluster 

```
helm uninstall postgres-cluster -n postgres-operator
```