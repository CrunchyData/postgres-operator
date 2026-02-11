## This chainsaw test will test multiple backup repos:
1. Try: Apply: Create cluster with 2 pgbackrest repos
   1. Sleep for 120s while the initial backups complete
   2. Assert: Confirm the repo host is ready and the repos are created.
   3. Assert: Confirm the initial backup to repo1 is complete.
   4. Catch: Describe the backup job
2. Try: Script: Run the pgbackrest_initialization.sh script
3. Try: Apply: Annotate the cluster to take a manual backup to repo2
   1. Assert: Confirm the job for the manual backup has completed
   2. Catch: Describe the backup job 
4. Try: Script: Run the pgbackrest_initialization.sh script
   1. Catch: Describe the backup job 


## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator --config ../../config.yaml
```
