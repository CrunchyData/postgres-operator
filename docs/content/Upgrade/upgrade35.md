---
title: "Upgrade PGO 3.5 Minor Versions"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---
## Upgrading Postgres Operator 3.5 Minor Versions

This procedure will give instructions on how to upgrade Postgres Operator 3.5 minor releases.

{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant data!

{{% / notice %}}

##### Prerequisites.
You will need the following items to complete the upgrade:

* The latest 3.5.X code for the Postgres Operator available
* The latest 3.5.X PGO client binary
* Finally, these instructions assume you are executing from $COROOT in a terminal window and that you are using the same user from your previous installation. This user must also still have admin privileges in your Kubernetes or Openshift environment.

##### Step 0
Run `pgo show config` and save this output to compare at the end to ensure you don't miss any of your current configuration changes.

##### Step 1
Update environment variables in the bashrc

    export CO_VERSION=3.5.X

If you are pulling your images from the same registry as before this should be the only update to the 3.5.X environment variables.

source the updated bash file:

    source ~/.bashrc

Check to make sure that the correct CO_IMAGE_TAG image tag is being used. With a centos7 base image and version 3.5.X of the operator your image tag will be in the format of `centos7-3.5.4`. Verify this by running echo $CO_IMAGE_TAG.

##### Step 2
Update the pgo.yaml file in `$COROOT/conf/postgres-operator/pgo.yaml`. Use the config that you saved in Step 0. to make sure that you have updated the settings to match the old config. Confirm that the yaml file includes the correct images for the version that you are upgrading to:

For Example:

```
CCPImageTag: centos7-10.9-2.3.3
COImageTag: centos7-3.5.4
```

##### Step 3  
Install the 3.5.X Operator:

    make deployoperator

Verify the Operator is running:

    kubectl get pod -n <operator namespace>


##### Step 4  
Update the PGO client binary to 3.5.X by replacing the binary file with the new one.
Run which pgo to ensure you are replacing the current binary.

##### Step 5  
Make sure that any and all configuration changes have been updated.  
Run:

    pgo show config

This will print out the current configuration that the operator is using.  Ensure you made any configuration changes required, you can compare this output with Step 0 to ensure no settings are missed.  If you happened to miss a setting, update the pgo.yaml file and rerun make deployoperator


##### Step 6
The Operator is now upgraded to 3.5.X.
Verify this by running:

    pgo version

## Postgres Operator Container Upgrade Procedure

At this point, the Operator should be running the latest minor version of 3.5, and new clusters will be built using the appropriate specifications defined in your pgo.yaml file. For the existing clusters, upgrades can be performed with the following steps.

{{% notice info %}}

Before beginning your upgrade procedure, be sure to consult the [Compatibility Requirements Page]
( {{< relref "configuration/compatibility.md" >}}) for container dependency information.

{{% / notice %}}

First, update the deployment of each replica, one at a time, with the new image version:

```
kubectl edit deployment.apps/yourcluster
```
then edit the line containing the image value, which will be similar to the following
```
image: crunchydata/crunchy-postgres:centos7-11.3-2.4.0
```

When this new deployment is written, it will kill the pod and recreate it with the new image. Do this for each replica, waiting for the previous pod to upgrade completely before moving to next.

Once the replicas have been updated, update the deployment of primary by updating the `image:` line in the same fashion, waiting for it to come back up.

Now, similar to the steps above, you will need to update the pgcluster `ccpimagetag:` to the new value:
```
kubectl edit pgcluster yourcluster
```

To check everything is now working as expected, execute
```
pgo test yourcluster
```
To validate the database connections and execute
```
pgo show cluster yourcluster
```
To check the various cluster elements are listed as expected.

There is a bug in the operator where the image version for the backrest repo deployment is not updated with a pgo upgrade. As a workaround for this you need to redeploy the backrest shared repo deployment with the correct image version.

First you will need to get a copy of the yaml file that defines the cluster:

    kubectl get deployment <cluster-name>-backrest-shared-repo -o yaml > <cluster-name>-backrest-repo.yaml

You can then edit the yaml file so that the deployment will use the correct image version:
edit `<cluster-name>-backrest-repo.yaml`

set to the image, for example:

    crunchydata/pgo-backrest-repo:centos7-3.5.4

Next you will need to delete the current backrest repo deployment and recreate it with the updated yaml:
```
kubectl delete deployment <cluster-name>-backrest-shared-repo
kubectl create -f <cluster-name>-backrest-repo.yaml
```
Verify that the correct images are being used for the cluster. Run `pgo show cluster <cluster-name>` on your cluster and check the version. Describe each of the pods in your cluster and verify that the image that is being used is correct.
```
pgo show cluster <cluster-name>
kubectl get pods
kubectl describe pod <cluster-name>-<id>
kubectl describe pod <cluster-name>-backrest-shared-repo-<id>
```
Finally, make sure that the correct version of pgbackrest is being used and verify backups are working. The versions of pgbackrest that are returned in the primary and backrest pods should match:
```
kubectl get pods
kubectl exec -it <cluster-name>-<id> -- pgbackrest version
kubectl exec -it <cluster-name>-backrest-shared-repo-<id> -- pgbackrest version
pgo backup <cluster-name> --backup-type=pgbackrest
```
You've now completed the upgrade and are running Crunchy PostgreSQL Operator v3.5.X, you can confirm this by running pgo version from the command line and running

    pgo show cluster <cluster-name>

on each cluster. For this minor upgrade, most existing settings and related services (such as pgbouncer, backup schedules and existing policies) are expected to work, but should be tested for functionality and adjusted or recreated as necessary.
