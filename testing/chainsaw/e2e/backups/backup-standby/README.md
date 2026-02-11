## This chainsaw test will test standby backups:
1. Try: Apply: Create cluster with standby backups, but replicas set to 1
   1. Assert: Confirm the backup pod failed
   2. Assert: Confirm the cluster has been created.
   3. Catch: Describe pods
2. Try: Script: Check backup logs to confirm the existence of "unable to find standby cluster - cannot proceed"
   1. Catch: Describe pods
3. Try: Apply: Add a replica instance to allow backups to standby
   1. Assert: Confirm the backup job has completed.
   2. Assert: Confirm the cluster is running.
   3. Catch: Describe pod 

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator --config ../../config.yaml
```

NOTE: You may need to increase the timeouts in the values.yaml file if you see any errors in running the test due to slowness in the cluster coming up or backups taken.
