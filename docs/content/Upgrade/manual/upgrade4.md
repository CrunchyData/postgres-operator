---
title: "Manual Upgrade - Operator 4"
draft: false
weight: 8
---

## Manual PostgreSQL Operator 4 Upgrade Procedure

Below are the procedures for upgrading to version {{< param operatorVersion >}} of the Crunchy PostgreSQL Operator using the Bash or Ansible installation methods. This version of the PostgreSQL Operator has several fundamental changes to the existing PGCluster structure and deployment model. Most notably for those upgrading from 4.1 and below, all PGClusters use the new Crunchy PostgreSQL HA container in place of the previous Crunchy PostgreSQL containers. The use of this new container is a breaking change from previous versions of the Operator did not use the HA containers.

NOTE: If you are upgrading from Crunchy PostgreSQL Operator version 4.1.0 or later, the [Automated Upgrade Procedure](/upgrade/automatedupgrade) is recommended. If you are upgrading PostgreSQL 12 clusters, you MUST use the [Automated Upgrade Procedure](/upgrade/automatedupgrade).

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

The Ansible installation upgrade procedure is below. Please click [here](/upgrade/upgrade4#bash-installation-upgrade-procedure) for the Bash installation upgrade procedure.

### Ansible Installation Upgrade Procedure

Below are the procedures for upgrading the PostgreSQL Operator and PostgreSQL clusters using the Ansible installation method.

##### Prerequisites.

You will need the following items to complete the upgrade:

* The latest {{< param operatorVersion >}} code for the Postgres Operator available

These instructions assume you are executing in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.

##### Step 1

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


##### Step 2

NOTE: Skip this step if your primary PVC still matches your original cluster name, or if you do not have pgBackrestBackups you wish to preserve for use in the upgraded cluster.

Otherwise, noting the primary PVC name mentioned in Step 2, run

```
kubectl exec mycluster-backrest-shared-repo-<id> -- bash -c "mv /backrestrepo/mycluster-backrest-shared-repo /backrestrepo/mycluster-ypvq-backrest-shared-repo"
```

where "mycluster" is the original cluster name, "mycluster-ypvq" is the primary PVC name and "mycluster-backrest-shared-repo-<id>" is the pgBackRest shared repo pod name.


##### Step 3

For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

For 4.2:

```
pgo delete cluster <clustername> --keep-backups --keep-data
```

For 4.0 and 4.1:

```
pgo delete cluster <clustername>
```

and then, for all versions, delete the "backrest-repo-config" secret, if it exists:

```
kubectl delete secret <clustername>-backrest-repo-config
```

If there are any remaining jobs for this deleted cluster, use

```
kubectl -n <namespace> delete job <jobname>
```

to remove the job and any associated "Completed" pods.


NOTE: Please note the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!


##### Step 4

Save a copy of your current inventory file with a new name (such as `inventory.backup)` and checkout the latest {{< param operatorVersion >}} tag of the Postgres Operator.


##### Step 5

Update the new inventory file with the appropriate values for your new Operator installation, as described in the [Ansible Install Prerequisites]( {{< relref "installation/other/ansible/prerequisites.md" >}}) and the [Compatibility Requirements Guide]( {{< relref "configuration/compatibility.md" >}}).


##### Step 6

Now you can upgrade your Operator installation and configure your connection settings as described in the [Ansible Update Page]( {{< relref "installation/other/ansible/updating-operator.md" >}}).


##### Step 7

Verify the Operator is running:

```
kubectl get pod -n <operator namespace>
```

And that it is upgraded to the appropriate version

```
pgo version
```

We strongly recommend that you create a test cluster before proceeding to the next step.

##### Step 8

Once the Operator is installed and functional, create a new {{< param operatorVersion >}} cluster matching the cluster details recorded in Step 1. Be sure to use the primary PVC name (also noted in Step 1) and the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs.

NOTE: If you have existing pgBackRest backups stored that you would like to have available in the upgraded cluster, you will need to follow the [PVC Renaming Procedure](#pgbackrest-repo-pvc-renaming).

A simple example is given below, but more information on cluster creation can be found [here](/pgo-client/common-tasks#creating-a-postgresql-cluster)

```
pgo create cluster <clustername> -n <namespace>
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

### Bash Installation Upgrade Procedure

Below are the procedures for upgrading the PostgreSQL Operator and PostgreSQL clusters using the Bash installation method.

##### Prerequisites.

You will need the following items to complete the upgrade:

* The code for the latest release of the PostgreSQL Operator
* The latest PGO client binary

Finally, these instructions assume you are executing from $PGOROOT in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.

##### Step 1

You will most likely want to run:

```
pgo show config -n <any watched namespace>
```

Save this output to compare once the procedure has been completed to ensure none of the current configuration changes are missing.

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

For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

For 4.2:

```
pgo delete cluster <clustername> --keep-backups --keep-data
```

For 4.0 and 4.1:

```
pgo delete cluster <clustername>
```

and then, for all versions, delete the "backrest-repo-config" secret, if it exists:

```
kubectl delete secret <clustername>-backrest-repo-config
```

NOTE: Please record the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!


##### Step 5

Delete the 4.X version of the Operator by executing:

```
$PGOROOT/deploy/cleanup.sh
$PGOROOT/deploy/remove-crd.sh
$PGOROOT/deploy/cleanup-rbac.sh
```

##### Step 6

For versions 4.0, 4.1 and 4.2, update environment variables in the bashrc:

```
export PGO_VERSION={{< param operatorVersion >}}
```

NOTE: This will be the only update to the bashrc file for 4.2.

If you are pulling your images from the same registry as before this should be the only update to the existing 4.X environment variables.

###### Operator 4.0

If you are upgrading from PostgreSQL Operator 4.0, you will need the following new environment variables:

```
# PGO_INSTALLATION_NAME is the unique name given to this Operator install
# this supports multi-deployments of the Operator on the same Kubernetes cluster
export PGO_INSTALLATION_NAME=devtest

# for setting the pgo apiserver port, disabling TLS or not verifying TLS
# if TLS is disabled, ensure setip() function port is updated and http is used in place of https
export PGO_APISERVER_PORT=8443          # Defaults: 8443 for TLS enabled, 8080 for TLS disabled
export DISABLE_TLS=false
export TLS_NO_VERIFY=false
export TLS_CA_TRUST=""
export ADD_OS_TRUSTSTORE=false
export NOAUTH_ROUTES=""

# for disabling the Operator eventing
export DISABLE_EVENTING=false
```

There is a new eventing feature, so if you want an alias to look at the eventing logs you can add the following:

```
elog () {
$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" logs `$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c event
}
```

###### Operator 4.1

If you are upgrading from PostgreSQL Operator 4.1.0 or 4.1.1, you will only need the following subset of the environment variables listed above:

```
export TLS_CA_TRUST=""
export ADD_OS_TRUSTSTORE=false
export NOAUTH_ROUTES=""
```

##### Step 7

Source the updated bash file:

```
source ~/.bashrc
```

##### Step 8

Ensure you have checked out the latest {{< param operatorVersion >}} version of the source code and update the pgo.yaml file in `$PGOROOT/conf/postgres-operator/pgo.yaml`

You will want to use the {{< param operatorVersion >}} pgo.yaml file and update custom settings such as image locations, storage, and resource configs.

##### Step 9

Create an initial Operator Admin user account.
You will need to edit the `$PGOROOT/deploy/install-bootstrap-creds.sh` file to configure the username and password that you want for the Admin account. The default values are:

```
PGOADMIN_USERNAME=admin
PGOADMIN_PASSWORD=examplepassword
```

You will need to update the `$HOME/.pgouser`file to match the values you set in order to use the Operator. Additional accounts can be created later following the steps described in the 'Operator Security' section of the main [Bash Installation Guide]( {{< relref "installation/other/bash.md" >}}). Once these accounts are created, you can change this file to login in via the PGO CLI as that user.

##### Step 10

Install the {{< param operatorVersion >}} Operator:

Setup the configured namespaces:

```
make setupnamespaces
```

Install the RBAC configurations:

```
make installrbac
```

Deploy the PostgreSQL Operator:

```
make deployoperator
```

Verify the Operator is running:

```
kubectl get pod -n <operator namespace>
```

##### Step 11

Next, update the PGO client binary to {{< param operatorVersion >}} by replacing the existing 4.X binary with the latest {{< param operatorVersion >}} binary available.

You can run:

```
which pgo
```

to ensure you are replacing the current binary.


##### Step 12

You will want to make sure that any and all configuration changes have been updated.  You can run:

```
pgo show config -n <any watched namespace>
```

This will print out the current configuration that the Operator will be using.

To ensure that you made any required configuration changes, you can compare with Step 0 to make sure you did not miss anything.  If you happened to miss a setting, update the pgo.yaml file and rerun:

```
make deployoperator
```

##### Step 13

The Operator is now upgraded to {{< param operatorVersion >}} and all users and roles have been recreated.
Verify this by running:

```
pgo version
```

We strongly recommend that you create a test cluster before proceeding to the next step.

##### Step 14

Once the Operator is installed and functional, create a new {{< param operatorVersion >}} cluster matching the cluster details recorded in Step 1. Be sure to use the same name and the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs. A simple example is given below, but more information on cluster creation can be found [here](/pgo-client/common-tasks#creating-a-postgresql-cluster)

NOTE: If you have existing pgBackRest backups stored that you would like to have available in the upgraded cluster, you will need to follow the [PVC Renaming Procedure](#pgbackrest-repo-pvc-renaming).

```
pgo create cluster <clustername> -n <namespace>
```

##### Step 15

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

##### Step 16

Scale up to the required number of replicas, as needed.

Congratulations! Your cluster is upgraded and ready to use!

### pgBackRest Repo PVC Renaming

If the pgcluster you are upgrading has an existing pgBackRest repo PVC that you would like to continue to use (which is required for existing pgBackRest backups to be accessible by your new cluster), the following renaming procedure will be needed.

##### Step 1

To start, if your current cluster is "mycluster", the pgBackRest PVC created by version 3.5 of the Postgres Operator will be named "mycluster-backrest-shared-repo". This will need to be renamed to "mycluster-pgbr-repo" to be used in your new cluster.

To begin, save the output description from the pgBackRest PVC:

In 4.0:
```
kubectl -n <namespace> describe pvc mycluster-backrest-shared-repo
```

In 4.1 and later:
```
kubectl -n <namespace> describe pvc mycluster-pgbr-repo
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

In 4.0:
```
kubectl -n <namespace> delete pvc mycluster-backrest-shared-repo
```

In 4.1 and later:
```
kubectl -n <namespace> delete pvc mycluster-pgbr-repo
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
