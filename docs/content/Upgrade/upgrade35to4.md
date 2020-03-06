---
title: "Upgrade PGO 3.5 to 4.3.0"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---

## Upgrading a Cluster from Version 3.5.x to PGO 4.3.0

This section will outline the procedure to upgrade a given cluster created using Postgres Operator 3.5.x to PGO version 4.3.0. This version of the Postgres Operator has several fundamental changes to the existing PGCluster structure and deployment model. Most notably, all PGClusters use the new Crunchy Postgres HA container in place of the previous Crunchy Postgres containers. The use of this new container is a breaking change from previous versions of the Operator.

#### Crunchy Postgres High Availability Containers

Using the PostgreSQL Operator 4.3.0 requires replacing your `crunchy-postgres` and `crunchy-postgres-gis` containers with the `crunchy-postgres-ha` and `crunchy-postgres-gis-ha` containers respectively. The underlying PostgreSQL installations in the container remain the same but are now optimized for Kubernetes environments to provide the new high-availability functionality.

A major change to this container is that the PostgreSQL process is now managed by Patroni. This allows a PostgreSQL cluster that is deployed by the PostgreSQL Operator to manage its own uptime and availability, to elect a new leader in the event of a downtime scenario, and to automatically heal after a failover event.

When creating your new clusters using version 4.3.0 of the Postgres Operator, the `pgo create cluster` command will automatically use the new `crunchy-postgres-ha` image if the image is unspecified. If you are creating a PostGIS enabled cluster, please be sure to use the updated image name, as with the command:
```
pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis-ha
```
{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant data!

{{% / notice %}}

##### Prerequisites.
You will need the following items to complete the upgrade:

* The latest PGO 4.3.0 code for the Postgres Operator available
* The latest PGO 4.3.0 client binary

##### Step 0
Create a new Linux user with the same permissions are the existing user used to install the Crunchy Postgres Operator. This is necessary to avoid any issues with environment variable differences between 3.5 and 4.3.0.

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
Log in as your new Linux user and install the 4.3.0 Postgres Operator.

[Bash Installation] ( {{< relref "installation/operator-install.md" >}}) 

Be sure to add the existing namespace to the Operator's list of watched namespaces (see the [Namespace] ( {{< relref "architecture/namespace.md" >}}) section of this document for more information) and make sure to avoid overwriting any existing data storage.


##### Step 4
Once the Operator is installed and functional, create a new 4.3.0 cluster with the same name and using the same major PostgreSQL version as was used previously. This will allow the new cluster to utilize the existing PVCs.

	pgo create cluster <clustername> -n <namespace>

##### Step 5
Manually update the old leftover Secrets to use the new label as defined in 4.3.0:

	kubectl label secret/<clustername>-postgres-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-primaryuser-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-testuser-secret pg-cluster=<clustername> -n <namespace>

##### Step 6
To verify cluster status, run

	pgo test <clustername> -n <namespace>

Output should be similar to:
```
cluster : mycluster
        Services
                primary (10.106.70.238:5432): UP
        Instances
                primary (mycluster-7d49d98665-7zxzd): UP
```
##### Step 7
Scale up to the required number of replicas, as needed.

It is also recommended to take full backups of each pgcluster once the upgrade is completed due to version differences between the old and new clusters.
