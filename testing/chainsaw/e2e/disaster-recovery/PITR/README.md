## Test Steps: Test a PITR
1. Test name: Create a cluster 
   1. Apply: Create a cluster with local backups with a manual diff backup annotated for repo1
   2. Assert: Describe the cluster pod to see if the database status is Running
   3. Catch: Describe the cluster pod
2. Test name: Create table and add some data
   1. Script: Get the cluster pod name
   2. Script: Exec into the primary cluster pod and add 1000 rows of data to date_test table; check to confirm 1000 rows were inserted by checking stdout
   3. Catch: Describe the cluster pod
3. Test name: Take a manual backup before PITR
   1. Script: Annotate the postgres-operator.crunchydata.com/pgbackrest-backup with the date to kick off the manual backup
   2. Assert: Check that postgres-operator.crunchydata.com/pgbackrest-backup: manual completes with state of Terminated due to reason: Completed.
   3. Catch: Describe the cluster pod
4. Test name: Store the recovery point and add more data 
   1. Script: Get the Primary pod name and exec into the pod, capturing the current time and annotating the cluster under testing/objective; check that the cluster was annotated.
   2. Script: Add 1000 more rows to the date_test table; check to confirm the table now has 2000 rows.
   3. Catch: Describe the cluster pod 
5. Test name: Restore the cluster
   1. Script: Patch the cluster with the testing/objective time; annotate the cluster with pgbackrest-restore=pitr_one
   2. Sleep: Duration 60s to allow the restore to complete
   3. Assert: Describe cluster to confirm pgbackrest for restore is `finished: true` for `id: pitr_one`
   4. Script: Get the primary pod name
   5. Script: Exec into the primary pod and select count(*) from the date_test table; check that the output is 1000
   6. Catch: Describe the cluster pod

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator
```
