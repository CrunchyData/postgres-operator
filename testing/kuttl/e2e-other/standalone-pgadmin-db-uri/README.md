# pgAdmin external database tests

Notes: 
- Due to the (random) namespace being part of the host, we cannot check the configmap using the usual assert/file pattern.
- These tests will only work with pgAdmin version v8 and higher

## create postgrescluster and add user schema
* 00:
  * create a postgrescluster with a label;
  * check that the cluster has the label and that the expected user secret is created.
* 01: 
  * create the user schema for pgAdmin to use

 ## create pgadmin and verify connection to database
* 02:
  * create a pgadmin with a selector for the existing cluster's label;
  * check the correct existence of the secret, configmap, and pod.
* 03: 
  * check that pgAdmin only has one user

 ## add a pgadmin user and verify it in the database
* 04:
  * update pgadmin with a new user;
  * check that the pod is still running as expected.
* 05:
  * check that pgAdmin now has two users and that the defined user is present.
