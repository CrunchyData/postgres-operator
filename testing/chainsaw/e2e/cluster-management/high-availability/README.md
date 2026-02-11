## This Chainsaw test will test cluster HA by adding replicas
1. Test name: Create cluster with 2 instances
   1. Apply: Create cluster with 2 instances (instance1 and instance2) with local backups
   2. Assert: Describe pod and check for status:phase:Running and database read: true
   3. Assert: Describe cluster to check the status of the instances.
   4. Sleep: sleep because the backup takes about 2 mins to complete
   5. Assert: Describe cluster to confirm replicaCreateBackupComplete: true and the repo has been created.
   Catch: Describe the pods upon failure
2. Test name: Create table and add some data
   1. Script: Connect to the primary database and create a table and add 1000 rows to that table
   2. Script: Query the replica instance to validate the data has been replicated
   Catch: Describe the pods upon failure
3. Test name: Perform a failover/switchover
   1. Script: Connect to the primary database pod to confirm pg_is_in_recovery is set to f.
   2. Script: Patch the cluster with switchover.enabled: "true" and annotate the cluster
   3. Sleep: Because it takes a while for the switchover to complete.
   4. Script: Confirm the old primary is now a replica and in recovery
   Catch: Describe the pods upon failure.
4. Test name: Add additional replicas
   1. Apply: Add additional replicas to both instances (instance1 = 2 and instance2 = 3)
   2. Assert: Describe the cluster and check that the instance statuses are correct.
   Catch: Describe the pods upon failure.

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator --config ../../config.yaml
```
