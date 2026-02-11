## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../values.yaml --namespace postgres-operator
```

NOTE: Removed the backups from this cluster so that the creation is quicker.
