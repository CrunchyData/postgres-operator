## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator
```

## This test has the following steps:
1. Try: Apply: Create the cluster with local pgbackrest repo1 configured
2. Try: Assert: Confirm the database is running
3. Try: Assert: Confirm the pgbackrest stanza was created and replicaCreateBackupComplete: true
4. Catch: Describe pods
