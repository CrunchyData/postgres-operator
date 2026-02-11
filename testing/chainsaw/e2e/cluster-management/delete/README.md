## This chainsaw test will test creating a cluster and deleting it:
1. Try: Apply: Create a simple cluster using the simple-cluster.yaml template file.
   1. Assert: Check the database pod to confirm the status is running
   2. Assert: Check the database cluster to confirm the instances are running
   3. Assert: Run the resource-check.yaml file to confirm the cluster resources exist
   4. Catch: Describe pods
   5. skipDelete: true (to prevent the WARN message when the test is complete)
2. Try: Delete the cluster
   1. Delete: Delete the cluster
   2. Error: Run the resource-check.yaml file to confirm the cluster resources do NOT exist 
   3. Catch: Describe pods
3. Try: Apply: Create simple cluster with replicas
   1. Assert: Check the database pod to confirm the status is running
   2. Assert: Check the database cluster to confirm the instances are running
   3. Assert: Run the resource-check.yaml file to confirm the cluster resources exis
   4. Catch: Describe pod 
   5. skipDelete: true (to prevent the WARN message when the test is complete)
4. Try: Delete the cluster
   1. Delete: Delete the cluster
   2. Error: Run the resource-check.yaml file to confirm the cluster resources do NOT exist 
   3. Catch: Describe pods
5. Try: Create a broken cluster where the image tag is "example.com/does-not-exist"
   1. Error: Confirm the cluster is in error
   2. Delete: Delete the cluster
   3. Error: Run the resource-check.yaml file to confirm the cluster resources do NOT exist 
   4. Catch: Describe pods 

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --namespace postgres-operator --values ../../values.yaml --config ../../config.yaml
```

NOTE: You may need to increase the timeouts in the values.yaml file if you see any errors in running the test due to slowness in the cluster coming up or backups taken.
