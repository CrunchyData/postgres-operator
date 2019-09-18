---
title: "Upgrade PGO 3.5 to 4"
Latest Release: 4.1.0 {docdate}
draft: false
weight: 8
---

## Upgrading a Cluster from Version 3.5.x to PGO 4

This section will outline the procedure to upgrade a given cluster created using Postgres Operator 3.5.x to PGO version 4.1.

##### Prerequisites.
You will need the following items to complete the upgrade:

* The latest PGO 4.1 code for the Postgres Operator available
* The latest PGO 4.1 client binary

##### Step 0
Create a new Centos/Redhat user with the same permissions are the existing user used to install the Crunchy Postgres Operator. This is necessary to avoid any issues with environment variable differences between 3.5 and 4.1.

##### Step 1
For the cluster(s) you wish to upgrade, scale down any replicas, if necessary, then delete the cluster

	pgo delete cluster <clustername>

{{% notice warning %}}

Please note the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!

{{% /notice %}}

##### Step 2
Delete the 3.5.x version of the operator by executing:

	$COROOT/deploy/cleanup.sh
	$COROOT/deploy/remove-crd.sh

##### Step 3
Log in as your new Linux user and install the 4.1 Postgres Operator. 

Be sure to add the existing namespace to the Operator's list of watched namespaces (see the [Namespace] ( {{< relref "gettingstarted/Design/namespace.md" >}}) section of this document for more information) and make sure to avoid overwriting any existing data storage.


##### Step 4
Once the Operator is installed and functional, create a new 4.1 cluster with the same name as was used previously. This will allow the new cluster to utilize the existing PVCs.

	pgo create cluster <clustername> -n <namespace>

##### Step 5
Manually update the old leftover Secrets to use the new label as defined in 4.1:

	kubectl label secret/<clustername>-postgres-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-primaryuser-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-testuser-secret pg-cluster=<clustername> -n <namespace>

##### Step 6
To verify cluster status, run 
	pgo test <clustername> -n <namespace>
Output should be similar to:
```
psql -p 5432 -h 10.104.74.189 -U postgres postgres is Working
psql -p 5432 -h 10.104.74.189 -U postgres userdb is Working
psql -p 5432 -h 10.104.74.189 -U primaryuser postgres is Working
psql -p 5432 -h 10.104.74.189 -U primaryuser userdb is Working
psql -p 5432 -h 10.104.74.189 -U testuser postgres is Working
psql -p 5432 -h 10.104.74.189 -U testuser userdb is Working
``` 
##### Step 7
Scale up to the required number of replicas, as needed.
