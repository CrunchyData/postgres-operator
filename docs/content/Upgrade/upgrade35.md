---
title: "Manual Upgrade - Operator 3.5"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---

## Upgrading the Crunchy PostgreSQL Operator from Version 3.5 to 4.3.0

This section will outline the procedure to upgrade a given cluster created using PostgreSQL Operator 3.5.x to PostgreSQL Operator version 4.3.0. This version of the PostgreSQL Operator has several fundamental changes to the existing PGCluster structure and deployment model. Most notably, all PGClusters use the new Crunchy PostgreSQL HA container in place of the previous Crunchy PostgreSQL containers. The use of this new container is a breaking change from previous versions of the Operator.

#### Crunchy PostgreSQL High Availability Containers

Using the PostgreSQL Operator 4.3.0 requires replacing your `crunchy-postgres` and `crunchy-postgres-gis` containers with the `crunchy-postgres-ha` and `crunchy-postgres-gis-ha` containers respectively. The underlying PostgreSQL installations in the container remain the same but are now optimized for Kubernetes environments to provide the new high-availability functionality.

A major change to this container is that the PostgreSQL process is now managed by Patroni. This allows a PostgreSQL cluster that is deployed by the PostgreSQL Operator to manage its own uptime and availability, to elect a new leader in the event of a downtime scenario, and to automatically heal after a failover event.

When creating your new clusters using version 4.3.0 of the PostgreSQL Operator, the `pgo create cluster` command will automatically use the new `crunchy-postgres-ha` image if the image is unspecified. If you are creating a PostGIS enabled cluster, please be sure to use the updated image name, as with the command:
```
pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis-ha
```
##### NOTE: As with any upgrade procedure, it is strongly recommended that a full logical backup is taken before any upgrade procedure is started. Please see the [Logical Backups](/pgo-client/common-tasks#logical-backups-pg_dump--pg_dumpall) section of the Common Tasks page for more information.

##### Prerequisites.
You will need the following items to complete the upgrade:

* The code for the latest PostgreSQL Operator available
* The latest client binary

##### Step 0

Create a new Linux user with the same permissions as the existing user used to install the Crunchy PostgreSQL Operator. This is necessary to avoid any issues with environment variable differences between 3.5 and 4.3.0.

##### Step 1

For the cluster(s) you wish to upgrade, record the cluster details provided by

        pgo show cluster <clustername>

so that your new clusters can be recreated with the proper settings. 

Also, you will need to note the name of the primary PVC. If it does not exactly match the cluster name, you will need to recreate your cluster using the primary PVC name as the new cluster name.

For example, given the following output:

	$ pgo show cluster mycluster

	cluster : mycluster (crunchy-postgres:centos7-11.5-2.4.2)
		pod : mycluster-7bbf54d785-pk5dq (Running) on kubernetes1 (1/1) (replica)
		pvc : mycluster
		pod : mycluster-ypvq-5b9b8d645-nvlb6 (Running) on kubernetes1 (1/1) (primary)
		pvc : mycluster-ypvq
	...

the new cluster's name will need to be "mycluster-ypvq"

##### Step2

For the cluster(s) you wish to upgrade, scale down any replicas, if necessary, then delete the cluster

	pgo delete cluster <clustername>

##### NOTE: Please record the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!

##### Step 3
Delete the 3.5.x version of the operator by executing:

	$COROOT/deploy/cleanup.sh
	$COROOT/deploy/remove-crd.sh

##### Step 4

Log in as your new Linux user and install the 4.3.0 PostgreSQL Operator.

[Bash Installation]( {{< relref "installation/other/bash.md" >}}) 

Be sure to add the existing namespace to the Operator's list of watched namespaces (see the [Namespace]( {{< relref "architecture/namespace.md" >}}) section of this document for more information) and make sure to avoid overwriting any existing data storage.


##### Step 5

Once the Operator is installed and functional, create a new 4.3.0 cluster matching the cluster details recorded in Step 1. Be sure to use the primary PVC name (also noted in Step 1) and the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs. A s
imple example is given below, but more information on cluster creation can be found [here](/pgo-client/common-tasks#creating-a-postgresql-cluster)

	pgo create cluster <clustername> -n <namespace>

##### Step 6

Manually update the old leftover Secrets to use the new label as defined in 4.3.0:

	kubectl label secret/<clustername>-postgres-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-primaryuser-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-testuser-secret pg-cluster=<clustername> -n <namespace>

##### Step 7

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
##### Step 8

Scale up to the required number of replicas, as needed.

Congratulations! Your cluster is upgraded and ready to use!
