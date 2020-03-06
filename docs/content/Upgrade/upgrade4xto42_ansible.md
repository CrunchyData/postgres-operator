---
title: "Upgrade PGO 4.X to 4.3.0 (Ansible)"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---

## Postgres Operator Ansible Upgrade Procedure from 4.X to 4.3.0

This procedure will give instructions on how to upgrade to version 4.3.0 of the Crunchy Postgres Operator using the Ansible installation method. This version of the Postgres Operator has several fundamental changes to the existing PGCluster structure and deployment model. Most notably, all PGClusters use the new Crunchy Postgres HA container in place of the previous Crunchy Postgres containers. The use of this new container is a breaking change from previous versions of the Operator.

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

* The latest 4.3.0 code for the Postgres Operator available

These instructions assume you are executing in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.


##### Step 0
For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

	pgo delete cluster <clustername>

{{% notice warning %}}

Please note the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!

{{% /notice %}}


##### Step 1

Save a copy of your current inventory file with a new name (such as `inventory.backup)` and checkout the latest 4.3.0 tag of the Postgres Operator.


##### Step 2
Update the new inventory file with the appropriate values for your new Operator installation, as described in the [Ansible Install Prerequisites] ( {{< relref "installation/install-with-ansible/prerequisites.md" >}}) and the [Compatibility Requirements Guide] ( {{< relref "configuration/compatibility.md" >}}).


##### Step 3

Now you can upgrade your Operator installation and configure your connection settings as described in the [Ansible Update Page] ( {{< relref "installation/install-with-ansible/updating-operator.md" >}}).


##### Step 4
Verify the Operator is running:

    kubectl get pod -n <operator namespace>

And that it is upgraded to the appropriate version

    pgo version

##### Step 5
Once the Operator is installed and functional, create a new 4.3.0 cluster with the same name and using the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs.

	pgo create cluster <clustername> -n <namespace>

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
