---
title: "Upgrade PGO 4.0.1 to 4.1.0 (Bash)"
Latest Release: 4.1.0 {docdate}
draft: false
weight: 8
---

## Postgres Operator Bash Upgrade Procedure from 4.0.1 to 4.1.0

This procedure will give instructions on how to upgrade to Postgres Operator 4.1.0 when using the Bash installation method.

{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant data!

{{% / notice %}}

##### Prerequisites.
You will need the following items to complete the upgrade:

* The latest 4.1.0 code for the Postgres Operator available
* The latest 4.1.0 PGO client binary

Finally, these instructions assume you are executing from $PGOROOT in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.

##### Step 0
You will most likely want to run:

    pgo show config -n <any watched namespace>

Save this output to compare once the procedure has been completed to ensure none of the current configuration changes are missing.


##### Step 1
For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

	pgo delete cluster <clustername>

{{% notice warning %}}

Please note the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!

{{% /notice %}}


##### Step 2
Delete the 4.0.1 version of the Operator by executing:

	$PGOROOT/deploy/cleanup.sh
	$PGOROOT/deploy/remove-crd.sh
	$PGOROOT/deploy/cleanup-rbac.sh


##### Step 3
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
export PGO_APISERVER_PORT=8443          # Defaults: 8443 for TLS enabled, 8080 for TLS disabled
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

##### Step 4

Ensure you have checked out the latest 4.1.0 version of the source code and update the pgo.yaml file in `$PGOROOT/conf/postgres-operator/pgo.yaml`

You will want to use the 4.1.0 pgo.yaml file and update custom settings such as image locations, storage, and resource configs.

##### Step 5
Create an initial Operator Admin user account.
You will need to edit the `$PGOROOT/deploy/install-bootstrap-creds.sh` file to configure the username and password that you want for the Admin account. The default values are:
```
export PGOADMIN_USERNAME=pgoadmin
export PGOADMIN_PASSWORD=examplepassword
```
You will need to update the `$HOME/.pgouser`file to match the values you set in order to use the Operator. Additional accounts can be created later following the steps described in the 'Operator Security' section of the main [Bash Installation Guide] ( {{< relref "installation/operator-install.md" >}}). Once these accounts are created, you can change this file to login in via the PGO CLI as that user.

##### Step 6

Install the 4.1.0 Operator:

Setup the configured namespaces:

    make setupnamespaces

Install the RBAC configurations:

    make installrbac

Deploy the Postgres Operator:

    make deployoperator

Verify the Operator is running:

    kubectl get pod -n <operator namespace>

##### Step 7
Next, update the PGO client binary to 4.1.0 by replacing the existing 4.0 binary with the latest 4.1.0 binary available.

You can run: 

    which pgo

to ensure you are replacing the current binary.


##### Step 8
You will want to make sure that any and all configuration changes have been updated.  You can run:

    pgo show config -n <any watched namespace>

This will print out the current configuration that the Operator will be using.

To ensure that you made any required configuration changes, you can compare with Step 0 to make sure you did not miss anything.  If you happened to miss a setting, update the pgo.yaml file and rerun:

    make deployoperator

##### Step 9
The Operator is now upgraded to 4.1.0 and all users and roles have been recreated.
Verify this by running:

    pgo version

##### Step 10
Once the Operator is installed and functional, create new 4.1 clusters with the same name as was used previously. This will allow the new clusters to utilize the existing PVCs.

	pgo create cluster <clustername> -n <namespace>

##### Step 11
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
##### Step 12
Scale up to the required number of replicas, as needed.

