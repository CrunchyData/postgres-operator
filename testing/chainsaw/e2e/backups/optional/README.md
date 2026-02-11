## This chainsaw test will test optional backups:
1. Try: Apply: Create the cluster with optional backups
   1. Assert: Confirm the database is running
   2. Assert: Confirm the PVC for the pgdata directory has been created.
   3. Script: Confirm the archive_command is true
   4. Errors: Confirm no repo opjects are created.
   5. Catch: Describe pods
2. Try: Apply: Create table and add some data
   1. Apply: Add replica instance
   2. Assert: Confirm the replica is up and running
   3. Script: Query the replica instance to confirm replication
   4. Catch: Describe pods
3. Try: Apply: Add backups and confirm everything
   1. Assert: Confirm the backup stanza has been created and if the backup has completed.
   2. Assert: Confirm the status of the PVC for pgdata
   3. Assert: Confirm the statefulset for the cluster
   4. Assert: Confirm the PVC for the backrest repo
   5. Script: Confirm the archive_command has changed to archive-push.
   6. Catch: Describe pod 
4. Try: Script: Patch to remove backups, but do not annotate
   1. Assert: Confirm the PVC for the pgdata directory has been created.
   2. Assert: Confirm the StatefulSet for the cluster
   3. Assert: Confirm the PVC for the pgbackrest repo
   4. Script: Confirm the archive_command has not changed
   5. Catch: Describe pod 
5. Try: Script: Annotate the cluster and confirm backups are removed
   1. Sleep for 30s to allow the backups to drop
   2. Assert: Confirm the status of the PVC for pgdat
   3. Script: Confirm the archive_command is true
   4. Errors: Confirm the backrest repo and it's components have been removed
   5. Catch: Describe pod 

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator --config ../../config.yaml
```
