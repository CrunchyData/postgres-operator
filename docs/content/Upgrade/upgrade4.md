---
title: "Manual Upgrade - Operator 4"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---

## Manual PostgreSQL Operator 4 Upgrade Procedure

Below are the procedures for upgrading to version 4.3.0 of the Crunchy PostgreSQL Operator using the Bash or Ansible installation methods. This version of the PostgreSQL Operator has several fundamental changes to the existing PGCluster structure and deployment model. Most notably for those upgrading from 4.1 and below, all PGClusters use the new Crunchy PostgreSQL HA container in place of the previous Crunchy PostgreSQL containers. The use of this new container is a breaking change from previous versions of the Operator did not use the HA containers.

#### Crunchy PostgreSQL High Availability Containers

Using the PostgreSQL Operator 4.3.0 requires replacing your `crunchy-postgres` and `crunchy-postgres-gis` containers with the `crunchy-postgres-ha` and `crunchy-postgres-gis-ha` containers respectively. The underlying PostgreSQL installations in the container remain the same but are now optimized for Kubernetes environments to provide the new high-availability functionality.

A major change to this container is that the PostgreSQL process is now managed by Patroni. This allows a PostgreSQL cluster that is deployed by the PostgreSQL Operator to manage its own uptime and availability, to elect a new leader in the event of a downtime scenario, and to automatically heal after a failover event.

When creating your new clusters using version 4.3.0 of the PostgreSQL Operator, the `pgo create cluster` command will automatically use the new `crunchy-postgres-ha` image if the image is unspecified. If you are creating a PostGIS enabled cluster, please be sure to use the updated image name, as with the command:
```
pgo create cluster mygiscluster --ccp-image=crunchy-postgres-gis-ha
```

##### NOTE: As with any upgrade procedure, it is strongly recommended that a full logical backup is taken before any upgrade procedure is started. Please see the [Logical Backups](/pgo-client/common-tasks#logical-backups-pg_dump--pg_dumpall) section of the Common Tasks page for more information.

The Ansible installation upgrade procedure is below. Please click [here](/upgrade/upgrade4#bash-installation-upgrade-procedure) for the Bash installation upgrade procedure.

### Ansible Installation Upgrade Procedure

Below are the procedures for upgrading the PostgreSQL Operator and PostgreSQL clusters using the Ansible installation method.

##### Prerequisites.

You will need the following items to complete the upgrade:

* The latest 4.3.0 code for the Postgres Operator available

These instructions assume you are executing in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.

##### Step 0

For the cluster(s) you wish to upgrade, record the cluster details provided by

        pgo show cluster <clustername>

so that your new clusters can be recreated with the proper settings.

##### Step 1

For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

        pgo delete cluster <clustername>

##### Please note the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!


##### Step 2

Save a copy of your current inventory file with a new name (such as `inventory.backup)` and checkout the latest 4.3.0 tag of the Postgres Operator.


##### Step 3

Update the new inventory file with the appropriate values for your new Operator installation, as described in the [Ansible Install Prerequisites]( {{< relref "installation/install-with-ansible/prerequisites.md" >}}) and the [Compatibility Requirements Guide]( {{< relref "configuration/compatibility.md" >}}).


##### Step 4

Now you can upgrade your Operator installation and configure your connection settings as described in the [Ansible Update Page]( {{< relref "installation/install-with-ansible/updating-operator.md" >}}).


##### Step 5

Verify the Operator is running:

    kubectl get pod -n <operator namespace>

And that it is upgraded to the appropriate version

    pgo version

##### Step 6

Once the Operator is installed and functional, create a new 4.3.0 cluster matching the cluster details recorded in Step 0. Be sure to use the same name and the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs. A simple example is given below, but more information on cluster creation can be found [here](/pgo-client/common-tasks#creating-a-postgresql-cluster)

        pgo create cluster <clustername> -n <namespace>

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

### Bash Installation Upgrade Procedure 

Below are the procedures for upgrading the PostgreSQL Operator and PostgreSQL clusters using the Bash installation method.

##### Prerequisites.

You will need the following items to complete the upgrade:

* The code for the latest release of the PostgreSQL Operator
* The latest PGO client binary

Finally, these instructions assume you are executing from $PGOROOT in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.

##### Step 0

You will most likely want to run:

    pgo show config -n <any watched namespace>

Save this output to compare once the procedure has been completed to ensure none of the current configuration changes are missing.

##### Step 1

For the cluster(s) you wish to upgrade, record the cluster details provided by

        pgo show cluster <clustername>

so that your new clusters can be recreated with the proper settings.

##### Step 2

For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

	pgo delete cluster <clustername>

##### NOTE: Please record the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!


##### Step 3

Delete the 4.X version of the Operator by executing:

	$PGOROOT/deploy/cleanup.sh
	$PGOROOT/deploy/remove-crd.sh
	$PGOROOT/deploy/cleanup-rbac.sh


##### Step 4

For versions 4.0, 4.1 and 4.2, update environment variables in the bashrc:

    export PGO_VERSION=4.3.0

Note: This will be the only update to the bashrc file for 4.2.

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

##### Step 5

Source the updated bash file:

    source ~/.bashrc

##### Step 6

Ensure you have checked out the latest 4.3.0 version of the source code and update the pgo.yaml file in `$PGOROOT/conf/postgres-operator/pgo.yaml`

You will want to use the 4.3.0 pgo.yaml file and update custom settings such as image locations, storage, and resource configs.

##### Step 7

Create an initial Operator Admin user account.
You will need to edit the `$PGOROOT/deploy/install-bootstrap-creds.sh` file to configure the username and password that you want for the Admin account. The default values are:
```
export PGOADMIN_USERNAME=pgoadmin
export PGOADMIN_PASSWORD=examplepassword
```
You will need to update the `$HOME/.pgouser`file to match the values you set in order to use the Operator. Additional accounts can be created later following the steps described in the 'Operator Security' section of the main [Bash Installation Guide] ( {{< relref "installation/operator-install.md" >}}). Once these accounts are created, you can change this file to login in via the PGO CLI as that user.

##### Step 8

Install the 4.3.0 Operator:

Setup the configured namespaces:

    make setupnamespaces

Install the RBAC configurations:

    make installrbac

Deploy the PostgreSQL Operator:

    make deployoperator

Verify the Operator is running:

    kubectl get pod -n <operator namespace>

##### Step 9

Next, update the PGO client binary to 4.3.0 by replacing the existing 4.X binary with the latest 4.3.0 binary available.

You can run:

    which pgo

to ensure you are replacing the current binary.


##### Step 10

You will want to make sure that any and all configuration changes have been updated.  You can run:

    pgo show config -n <any watched namespace>

This will print out the current configuration that the Operator will be using.

To ensure that you made any required configuration changes, you can compare with Step 0 to make sure you did not miss anything.  If you happened to miss a setting, update the pgo.yaml file and rerun:

    make deployoperator

##### Step 11

The Operator is now upgraded to 4.3.0 and all users and roles have been recreated.
Verify this by running:

    pgo version

##### Step 12

Once the Operator is installed and functional, create a new 4.3.0 cluster matching the cluster details recorded in Step 1. Be sure to use the same name and the same major PostgreSQL version as was used previously. This will allow the new clusters to utilize the existing PVCs. A simple example is given below, but more information on cluster creation can be found [here](/pgo-client/common-tasks#creating-a-postgresql-cluster)

        pgo create cluster <clustername> -n <namespace>

##### Step 13

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
##### Step 14

Scale up to the required number of replicas, as needed.

Congratulations! Your cluster is upgraded and ready to use!

