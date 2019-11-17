---
title: "Upgrade PGO 4.0.1 to 4.1.0 (Ansible)"
Latest Release: 4.1.1 {docdate}
draft: false
weight: 8
---

## Postgres Operator Ansible Upgrade Procedure from 4.0.1 to 4.1.0

This procedure will give instructions on how to upgrade to Postgres Operator 4.1.0 when using the Ansible installation method.

{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant data!

{{% / notice %}}

##### Prerequisites.
You will need the following items to complete the upgrade:

* The latest 4.1.0 code for the Postgres Operator available

These instructions assume you are executing in a terminal window and that your user has admin privileges in your Kubernetes or Openshift environment.


##### Step 0
For the cluster(s) you wish to upgrade, scale down any replicas, if necessary (see `pgo scaledown --help` for more information on command usage) page for more information), then delete the cluster

	pgo delete cluster <clustername>

{{% notice warning %}}

Please note the name of each cluster, the namespace used, and be sure not to delete the associated PVCs or CRDs!

{{% /notice %}}


##### Step 1

Save a copy of your current inventory file with a new name (such as `inventory.backup)` and checkout the latest 4.1 tag of the Postgres Operator.


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
Once the Operator is installed and functional, create new 4.1 clusters with the same name as was used previously. This will allow the new clusters to utilize the existing PVCs.

	pgo create cluster <clustername> -n <namespace>

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
