## This chainsaw test will test creating a cluster with tablespaces:
Note: This series of tests depends on PGO being deployed with the TablespaceVolume feature gate enabled.
1. Try: Apply: Create a cluster with tablespace volumes and a configmap `databaseInitSQL` to create tablespaces named trial and castl with the non-superuser as owner.
   1. Assert: Check the database pod to confirm the status is running
   2. Catch: Describe pods
2. Try: Connect to the cluster and create tables in the new tablespaces by creating a psql job.
   1. Assert: Confirm the psql job succeeded. 
   2. Catch: Describe pods

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --namespace postgres-operator --values ../values.yaml --config ../config.yaml
```

NOTE: You may need to increase the timeouts in the values.yaml file if you see any errors in running the test due to slowness in the cluster coming up or backups taken.
