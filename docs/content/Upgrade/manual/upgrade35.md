---
title: "Manual Upgrade - Operator 3.5"
draft: false
weight: 8
---

## Upgrading the Crunchy PostgreSQL Operator from Version 3.5 to {{< param operatorVersion >}}

This section will outline the procedure to upgrade a given cluster created using PostgreSQL Operator 3.5.x to PostgreSQL Operator version {{< param operatorVersion >}}. This version of the PostgreSQL Operator has several fundamental changes to the existing PGCluster structure and deployment model. Most notably, all PGClusters use the new Crunchy PostgreSQL HA container in place of the previous Crunchy PostgreSQL containers. The use of this new container is a breaking change from previous versions of the Operator.

#### Crunchy PostgreSQL High Availability Containers

Using the PostgreSQL Operator {{< param operatorVersion >}} requires replacing your `crunchy-postgres` and `crunchy-postgres-gis` containers with the `crunchy-postgres-ha` and `crunchy-postgres-gis-ha` containers respectively. The underlying PostgreSQL installations in the container remain the same but are now optimized for Kubernetes environments to provide the new high-availability functionality.

A major change to this container is that the PostgreSQL process is now managed by Patroni. This allows a PostgreSQL cluster that is deployed by the PostgreSQL Operator to manage its own uptime and availability, to elect a new leader in the event of a downtime scenario, and to automatically heal after a failover event.

When creating your new clusters using version {{< param operatorVersion >}} of the PostgreSQL Operator, the `pgo create cluster` command will automatically use the new `crunchy-postgres-ha` image if the image is unspecified. If you are creating a PostGIS enabled cluster, please be sure to use the updated image name and image tag, as with the command:

```
pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis-ha --ccp-image-tag={{< param centosBase >}}-{{< param postgresVersion >}}-{{< param postgisVersion >}}-{{< param operatorVersion >}}
```
Where `{{< param postgresVersion >}}` is the PostgreSQL version, `{{< param postgisVersion >}}` is the PostGIS version and `{{< param operatorVersion >}}` is the PostgreSQL Operator version.
Please note, no tag validation will be performed and additional steps may be required to upgrade your PostGIS extension implementation. For more information on PostGIS upgrade considerations, please see
[PostGIS Upgrade Documentation](https://access.crunchydata.com/documentation/postgis/latest/postgis_installation.html#upgrading).

NOTE: As with any upgrade procedure, it is strongly recommended that a full logical backup is taken before any upgrade procedure is started. Please see the [Logical Backups](/pgo-client/common-tasks#logical-backups-pg_dump--pg_dumpall) section of the Common Tasks page for more information.

##### Prerequisites.
You will need the following items to complete the upgrade:

* The code for the latest PostgreSQL Operator available
* The latest client binary

##### Step 1

Create a new Linux user with the same permissions as the existing user used to install the Crunchy PostgreSQL Operator. This is necessary to avoid any issues with environment variable differences between 3.5 and {{< param operatorVersion >}}.

##### Step 2

For the cluster(s) you wish to upgrade, record the cluster details provided by

```
pgo show cluster <clustername>
```

so that your new clusters can be recreated with the proper settings.

Also, you will need to note the name of the primary PVC. If it does not exactly match the cluster name, you will need to recreate your cluster using the primary PVC name as the new cluster name.

For example, given the following output:

```
$ pgo show cluster mycluster

cluster : mycluster (crunchy-postgres:centos7-11.5-2.4.2)
	pod : mycluster-7bbf54d785-pk5dq (Running) on kubernetes1 (1/1) (replica)
	pvc : mycluster
	pod : mycluster-ypvq-5b9b8d645-nvlb6 (Running) on kubernetes1 (1/1) (primary)
	pvc : mycluster-ypvq
...
```

the new cluster's name will need to be "mycluster-ypvq"

##### Step 3

NOTE: Skip this step if your primary PVC still matches your original cluster name, or if you do not have pgBackrestBackups you wish to preserve for use in the upgraded cluster.

Otherwise, noting the primary PVC name mentioned in Step 2, run

```
kubectl exec mycluster-backrest-shared-repo-<id> -- bash -c "mv /backrestrepo/mycluster-backrest-shared-repo /backrestrepo/mycluster-ypvq-backrest-shared-repo"
```

where "mycluster" is the original cluster name, "mycluster-ypvq" is the primary PVC name and "mycluster-backrest-shared-repo-<id>" is the pgBackRest shared repo pod name.

##### Step 4

For the cluster(s) you wish to upgrade, scale down any replicas, if necessary, then delete the cluster

```
pgo delete cluster <clustername>
```

If there are any remaining jobs for this deleted cluster, use

```
kubectl -n <namespace> delete job <jobname>
```

to remove the job and any associated "Completed" pods.

NOTE: Please record the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!

##### Step 5

Delete the 3.5.x version of the operator by executing:

```
$COROOT/deploy/cleanup.sh
$COROOT/deploy/remove-crd.sh
```

##### Step 6

Log in as your new Linux user and install the {{< param operatorVersion >}} PostgreSQL Operator as described in the [Bash Installation Procedure]( {{< relref "installation/other/bash.md" >}}).

Be sure to add the existing namespace to the Operator's list of watched namespaces (see the [Namespace]( {{< relref "architecture/namespace.md" >}}) section of this document for more information) and make sure to avoid overwriting any existing data storage.

We strongly recommend that you create a test cluster before proceeding to the next step.


##### Step 7

Once the Operator is installed and functional, create a new {{< param operatorVersion >}} cluster matching the cluster details recorded in Step 1. Be sure to use the primary PVC name (also noted in Step 1) and the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs.

NOTE: If you have existing pgBackRest backups stored that you would like to have available in the upgraded cluster, you will need to follow the [PVC Renaming Procedure](#pgbackrest-repo-pvc-renaming).

A simple example is given below, but more information on cluster creation can be found [here](/pgo-client/common-tasks#creating-a-postgresql-cluster)

```
pgo create cluster <clustername> -n <namespace>
```

##### Step 8

Manually update the old leftover Secrets to use the new label as defined in {{< param operatorVersion >}}:

```
kubectl -n <namespace> label secret/<clustername>-postgres-secret pg-cluster=<clustername> -n <namespace>
kubectl -n <namespace> label secret/<clustername>-primaryuser-secret pg-cluster=<clustername> -n <namespace>
kubectl -n <namespace> label secret/<clustername>-testuser-secret pg-cluster=<clustername> -n <namespace>
```

##### Step 9

To verify cluster status, run

```
pgo test <clustername> -n <namespace>
```

Output should be similar to:

```
cluster : mycluster
        Services
                primary (10.106.70.238:5432): UP
        Instances
                primary (mycluster-7d49d98665-7zxzd): UP
```

##### Step 10

Scale up to the required number of replicas, as needed.

Congratulations! Your cluster is upgraded and ready to use!


### pgBackRest Repo PVC Renaming

If the pgcluster you are upgrading has an existing pgBackRest repo PVC that you would like to continue to use (which is required for existing pgBackRest backups to be accessible by your new cluster), the following renaming procedure will be needed.

##### Step 1

To start, if your current cluster is "mycluster", the pgBackRest PVC created by version 3.5 of the Postgres Operator will be named "mycluster-backrest-shared-repo". This will need to be renamed to "mycluster-pgbr-repo" to be used in your new cluster.

To begin, save the output from

```
kubectl -n <namespace> describe pvc mycluster-backrest-shared-repo
```

for later use when recreating this PVC with the new name. In this output, note the "Volume" name, which is the name of the underlying PV.

##### Step 2

Now use

```
kubectl -n <namespace> get pv <PV name>
```

to check the "RECLAIM POLICY". If this is not set to "Retain", edit the "persistentVolumeReclaimPolicy" value so that it is set to "Retain" using

```
kubectl -n <namespace> patch pv <PV name> --type='json' -p='[{"op": "replace", "path": "/spec/persistentVolumeReclaimPolicy", "value":"Retain"}]'
```

##### Step 3

Now, delete the PVC:

```
kubectl -n <namespace> delete pvc mycluster-backrest-shared-repo
```

##### Step 4

You will remove the "claimRef" section of the PV with

```
kubectl -n <namespace> patch pv <PV name> --type=json -p='[{"op": "remove", "path": "/spec/claimRef"}]'
```

which will make the PV "Available" so it may be reused by the new PVC.

##### Step 5

Now, create a file with contents similar to the following:

```
kind: PersistentVolumeClaim
apiVersion: v1
metadata:
  name: mycluster-pgbr-repo
  namespace: demo
spec:
  storageClassName: ""
  accessModes:
    - ReadWriteMany
  resources:
    requests:
      storage: 2Gi
  volumeMode: Filesystem
  volumeName: "crunchy-pv156"
```

where name matches your new cluster (Remember that this will need to match the "primary PVC" name identified in [Step 2](#step-2) of the upgrade procedure!) and namespace, storageClassName, accessModes, storage, volumeMode and volumeName match your original PVC.

##### Step 6

Now you can use the new file to recreate your PVC using

```
kubectl -n <namespace> create -f <filename>
```

To check that your PVC is "Bound", run

```
kubectl -n <namespace> get pvc mycluster-pgbr-repo
```
Congratulations, you have renamed your PVC! Once the PVC Status is "Bound", your cluster can be recreated. If you altered the Reclaim Policy on your PV in Step 1, you will want to reset it now.
