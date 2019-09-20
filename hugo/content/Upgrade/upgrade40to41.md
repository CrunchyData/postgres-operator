---
title: "Upgrade PGO 4.0.1 to 4.1.0"
Latest Release: 4.1.0 {docdate}
draft: false
weight: 8
---

## Postgres Operator Upgrade Procedure from 4.0.1 to 4.1.0

This procedure will give instructions on how to upgrade to Postgres Operator 4.1.0. 

{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant data!

{{% / notice %}}

##### Prerequisites.
You will need the following items to complete the upgrade:

* The latest 4.1.0 code for the Postgres Operator available
* The latest 4.1.0 PGO client binary
* Ensure you have a copy of the current pgouser and pgorole files for your environment and ensure you do not overwrite the current files.  

Because user and role management has changed in 4.1.0, you will need to get these files if you do not want to manually add all existing users/roles into Kubernetes secrets.

If you do not have access to the current files, you can pull the information via the configmap with the following command:

    kubectl describe configmap pgo-config -n <namespace of pgo>

With output similar to the default, shown below:
```
pgorole:
----
pgoadmin: Cat, Ls, ShowNamespace, CreateDump, RestoreDump, ScaleCluster, CreateSchedule, DeleteSchedule, ShowSchedule, DeletePgbouncer, CreatePgbouncer, DeletePgpool, CreatePgpool, Restore, RestorePgbasebackup, ShowSecrets, Reload, ShowConfig, Status, DfCluster, DeleteCluster, ShowCluster, CreateCluster, TestCluster, ShowBackup, DeleteBackup, CreateBackup, Label, Load, CreatePolicy, DeletePolicy, ShowPolicy, ApplyPolicy, ShowWorkflow, ShowPVC, CreateUpgrade, CreateUser, DeleteUser, User, Version, CreateFailover, UpdateCluster, CreateBenchmark, ShowBenchmark, DeleteBenchmark
pgoreader: Cat, Ls, ShowNamespace, Status, ShowConfig, DfCluster, ShowCluster, TestCluster, ShowBackup, ShowPolicy, ShowWorkflow, ShowPVC, Version, ShowSchedule, ShowBenchmark

pgouser:
----
username:password:pgoadmin:
readonlyuser:password:pgoreader:
```
Finally, these instructions assume you are executing from $PGOROOT in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.

##### Step 0
You will most likely want to run:
    
    pgo show config -n <any watched namespace>

Save this output to compare once the procedure has been completed to ensure none of the current configuration changes are missing.

##### Step 1
Update environment variables in the bashrc:

    export PGO_VERSION=4.1.0

If you are pulling your images from the same registry as before this should be the only update to the existing 4.0.1 environment variables.

You will need the following new environment variables:
```
# PGO_INSTALLATION_NAME is the unique name given to this Operator install
# this supports multi-deployments of the Operator on the same Kube cluster
export PGO_INSTALLATION_NAME=devtest

# for setting the pgo apiserver port, disabling TLS or not verifying TLS
# if TLS is disabled, ensure setip() function port is updated and http is used in place of https
export PGO_APISERVER_PORT=8443		# Defaults: 8443 for TLS enabled, 8080 for TLS disabled
export DISABLE_TLS=false
export TLS_NO_VERIFY=false

# for disabling the Operator eventing
export DISABLE_EVENTING=false
```
There is a new eventing feature in 4.1.0, so if you want an alias to look at the eventing logs you can add the following:
```
elog () {
$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" logs `$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c event
}
```
Finally source the updated bash file:

    source ~/.bashrc

##### Step 2
Update the pgo.yaml file in `$PGOROOT/conf/postgres-operator/pgo.yaml`

You will want to use the 4.1.0 pgo.yaml file and update custom settings such as image locations, storage, and resource configs.

##### Step 3
Create an initial Operator Admin user account.
You will need to edit the `$PGOROOT/deploy/install-bootstrap-creds.sh` file to configure the username and password that you want for the Admin account. The default values are:
```
export PGOADMIN_USERNAME=pgoadmin
export PGOADMIN_PASSWORD=examplepassword
```
You will need to update the `$HOME/.pgouser`file to match the values you set in order to use the Operator. Once you create more accounts in a later step you can change this file to login in via the PGO CLIi as that user.

You can verify the secrets are created by running:

    Kubectl get secrets -n <pgo namespace>

```
pgorole-pgoadmin             Opaque                             2      19s
pgouser-pgoadmin             Opaque                             3      18s
```

##### Step 4
Install the 4.1.0 Operator:

    make deployoperator

Verify the Operator is running:

    kubectl get pod -n <operator namespace>


##### Step 5
Update the PGO client binary to 4.1.0 by replacing the existing 4.0 binary with the latest 4.1.0 binary available.

You can run: 

    which pgo

to ensure you are replacing the current binary.


##### Step 6
You will want to make sure that any and all configuration changes have been updated.  You can run:

    pgo show config -n <any watched namespace>

This will print out the current configuration that the Operator will be using.  

To ensure that you made any required configuration changes, you can compare with Step 0 to make sure you did not miss anything.  If you happened to miss a setting, update the pgo.yaml file and rerun:

    make deployoperator

##### Step 7
Add the previous users and roles into Kubernetes as Secrets so that Operator 4.1 can use them using the following steps.

We will be running `$PGOROOT/deploy/upgrade-creds.sh`, there are 2 ways to pass in the user and roles as described next.

###### Option 1
The original location in 4.0 for the pgouser and pgorole files was in the `$PGOROOT/conf/postgres-operator` folder.  If you put the backed up files into that folder, the upgrade-creds script will find them automatically.  

###### Option 2
However, if you prefer not to mix the 4.0 files into the 4.1 files you can pass in the files on the command line using the command below: 

    $PGOROOT/deploy/upgrade-creds.sh /path/to/pgorole /path/to/pgouser

After running the upgrade-creds script you can verify the users and roles have been created as Secrets by running:

    kubectl get secrets -n $PGO_OPERATOR_NAMESPACE

##### Step 8
The Operator is now upgraded to 4.1.0 and all users and roles have been recreated.
Verify this by running:

    pgo version


## Postgres Operator Container Upgrade Procedure

At this point, the Operator will be running version 4.1.0, and new clusters will be built using the appropriate specifications defined in your pgo.yaml file. For the existing clusters, upgrades can be performed with the following steps.

To bring your clusters up to the latest versions of Postgres and Containers, for each of your clusters you will want to run the following:
```
pgo scaledown <clustername> --query
pgo scaledown <clustername> --target
```

Now that your cluster only has one pod you can run the minor upgrade:

    pgo upgrade cluster <clustername>

By default this command updates the cluster with the values in the pgo.yaml.  If however you are running more than one version of Postgres clusters you can run the following to upgrade any clusters that do not match what is in your current configuration.

    pgo upgrade <clustername> --ccp-image-tag=<imagetag>

Once the minor upgrade is done you can scale your cluster back to the previous number of replicas, for example:

    pgo scale <clustername> --replica-count=2

There is a bug in the operator where the image version for the backrest repo deployment is not updated with a pgo upgrade. As a workaround for this you need to redeploy the backrest shared repo deployment with the correct image version.

    kubectl get deployment <cluster-name>-backrest-shared-repo -o yaml > <cluster-name>-backrest-repo.yaml

Edit the file

    <cluster-name>-backrest-repo.yaml

And set to the image (example below, please update to match your image repository details):

    crunchydata/container-suite/pgo-backrest-repo:centos7-4.1.0

Next you will need to delete the current backrest repo deployment and recreate it with the updated yaml:
```
kubectl delete deployment <cluster-name>-backrest-shared-repo
kubectl create -f <cluster-name>-backrest-repo.yaml
```

Verify that the correct images are being used for the cluster. Run 

    pgo show cluster <cluster-name>
on your cluster and check the version. Describe each of the pods in your cluster and verify that the image that is being used is correct.

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

You've now completed the upgrade and are running Crunchy PostgreSQL Operator v4.1.0, you can confirm this by running 

    pgo version

from the command line and running 

    pgo show cluster <cluster-name>

on each cluster. For this minor upgrade, most existing settings and related services (such as pgbouncer, backup schedules and existing policies) are expected to work, but should be tested for functionality and adjusted or recreated as necessary.
