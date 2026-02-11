## Test Steps: Test a Streaming Standby
1. Test name: Create the Secrets
   1. Apply: Using the template file ../tls_secret.yaml, create the CustomTLS Secrets
2. Test name: Create a cluster using the CustomTLS Secret
   1. Apply: Create a cluster with no backups
   2. Assert: Describe the primary database PostgresCluster resource to confirm the instance is running
   3. Assert: Describe the Service for the primary to confirm it exists
   4. Catch: Describe the primary cluster pod
3. Test name: Create table and add some data
   1. Script: Get the cluster pod name
   2. Script: Exec into the primary cluster pod and add 1000 rows of data to date_test table; check to confirm 1000 rows were inserted by checking stdout
   3. Catch: Describe the primary cluster pod
4. Test name: Create the Standby cluster
   1. Apply: Create a Standby cluster with 1 instance using the CustomTLS Secrets
   2. Sleep: for 2ms because it took a little while for things to come up
   3. Assert: Describe Standby cluster PostgresCluster resource to confirm the instance is running
   4. Assert: Confirm the Standby cluster Service exists
   5. Script: Query the Standby cluster to confirm the data has been replicated.
   6. Catch: Describe the Standby cluster pod

## Setup: Create the namespace and the Operator
```
kubectl apply -k kustomize/install/namespace
kubectl apply --server-side -k kustomize/install/default
```

## To run the test:
```
chainsaw test --values ../../values.yaml --namespace postgres-operator --config ../../config.yaml
```

This test references ../tls_secret.yaml that contain the TLS certificates for the Secrets. These certs are valid until 2033.
