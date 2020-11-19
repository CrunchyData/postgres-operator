---
title: "Automated PostgreSQL Operator Upgrade - Operator 4.1+"
Latest Release: 4.3.4 {docdate}
draft: false
weight: 80
---

## Automated PostgreSQL Operator Upgrade Procedure

The automated upgrade to a new release of the PostgreSQL Operator comprises two main steps:

* Upgrading the PostgreSQL Operator itself
* Upgrading the existing PostgreSQL Clusters to the new release 

The first step will result in an upgraded PostgreSQL Operator that is able to create and manage new clusters as expected, but will be unable to manage existing clusters until they have been upgraded. The second step upgrades the clusters to the current Operator version, allowing them to once again be fully managed by the Operator.

The automated upgrade procedure is designed to facilate the quickest and most efficient method to the current release of the PostgreSQL Operator. However, as with any upgrade, there are several considerations before beginning.

### Considerations

1. Versions Supported - This upgrade currently supports cluster upgrades from PostgreSQL Operator version 4.1.0 and later.

2. PostgreSQL Major Version Requirements - The underlying PostgreSQL major version must match between the old and new clusters. For example, if you are upgrading a 4.1.0 version of the PostgreSQL Operator and the cluster is using PostgreSQL 11.5, your upgraded clusters will need to use container images with a later minor version of PostgreSQL 11. Note that this is not a requirement for new clusters, which may use any currently supported version. For more information, please see the [Compatibility Requirements]({{< relref "configuration/compatibility.md" >}}).

3. Cluster Downtime - The re-creation of clusters will take some time, generally on the order of minutes but potentially longer depending on the operating environment. As such, the timing of the upgrade will be an important consideration. It should be noted that the upgrade of the PostgreSQL Operator itself will leave any existing cluster resources in place until individual pgcluster upgrades are performed.

4. Destruction and Re-creation of Certain Resources - As this upgrade process does destroy and recreate most elements of the cluster, unhealthy Kubernetes or Openshift environments may have difficulty recreating the necessary elements. Node availability, necessary PVC storage allocations and processing requirements are a few of the resource considerations to make before proceeding.

5. Compatibility with Custom Configurations -  Given the nearly endless potential for custom configuration settings, it is important to consider any resource or implemenation that might be uniquely tied to the current PostgreSQL Operator version.

6. Storage Requirements - An essential part of both the automated and manual upgrade procedures is the reuse of existing PVCs. As such, it is essential that the existing storage settings are maintained for any upgraded clusters.

7. As opposed to the manual upgrade procedures, the automated upgrade is designed to leave existing resources (such as CRDs, config maps, secrets, etc) in place whenever possible to minimize the need for resource re-creation.

8. Metrics - While the PostgreSQL Operator upgrade process will not delete an existing Metrics Stack, it does not currently support the upgrade of existing metrics infrastructure.

##### NOTE: As with any upgrade procedure, it is strongly recommended that a full logical backup is taken before any upgrade procedure is started. Please see the [Logical Backups](/pgo-client/common-tasks#logical-backups-pg_dump--pg_dumpall) section of the Common Tasks page for more information.
 
### Automated Upgrade when using an Ansible installation of the PostgreSQL Operator

For existing PostgreSQL Operator deployments that were installed using Ansible, the upgrade process is straightforward. 

First, you will copy your existing inventory file as a backup for your existing settings. You will reference these settings, but you will need to use the updated version of the inventory file for the current version of PostgreSQL Operator.

Once you've checked out the appropriate release tag, please follow the [Update Instructions]({{< relref "installation/other/ansible/updating-operator.md" >}}), being sure to update the new inventory file with your required settings. Please keep the above [Considerations](/upgrade/automatedupgrade#considerations) in mind, particularly with regard to the version and storage requirements listed.

Once the update is complete, you should now see the PostgreSQL Operator pods are up and ready. It is strongly recommended that you create a test cluster to validate proper functionality before moving on to the [Automated Cluster Upgrade](/upgrade/automatedupgrade#postgresql-operator-automated-cluster-upgrade) section below.

### Automated Upgrade when using a Bash installation of the PostgreSQL Operator

Like the Ansible procedure given above, the Bash upgrade procedure for upgrading the PostgreSQL Operator will require some manual configuration steps before the upgrade can take place. These updates will be made to your user's environment variables and the pgo.yaml configuration file.

#### PostgreSQL Operator Configuration Updates

To begin, you will need to make the following updates to your existing configuration.

##### Bashrc File Updates

First, you will make the following updates to your $HOME/.bashrc file.

When upgrading from version 4.1.X, in `$HOME/.bashrc`

Add the following variables:

```
export TLS_CA_TRUST=""
export ADD_OS_TRUSTSTORE=false
export NOAUTH_ROUTES=""

# Disable default inclusion of OS trust in PGO clients
export EXCLUDE_OS_TRUST=false
```

Then, for either 4.1.X or 4.2.X, 

Update the `PGO_VERSION` variable to `4.3.4`

Finally, source this file with 
```
source $HOME/.bashrc
```

##### PostgreSQL Operator Configuration File updates

Next, you will and save a copy of your existing pgo.yaml file (`$PGOROOT/conf/postgres-operator/pgo.yaml`) as pgo_old.yaml or similar. 

Once this is saved, you will checkout the current release of the PostgreSQL Operator and update the pgo.yaml for the current version, making sure to make updates to the CCPImageTag and storage settings in line with the [Considerations](/upgrade/automatedupgrade#considerations) given above.

#### Upgrading the Operator

Once the above configuration updates are completed, the PostgreSQL Operator can be upgraded.
To help ensure that needed resources are not inadvertently deleted during an upgrade of the PostgreSQL Operator, a helper script is provided. This script provides a similar function to the Ansible installation method's 'update' tag, where the Operator is undeployed, and the designated namespaces, RBAC rules, pods, etc are redeployed or recreated as appropriate, but required CRDs and other resources are left in place.

To use the script, execute:
```
$PGOROOT/deploy/upgrade-pgo.sh
```
This script will undeploy the current PostgreSQL Operator, configure the desired namespaces, install the RBAC configuration, deploy the new Operator, and, attempt to install a new PGO client, assuming default location settings are being used.

After this script completes, it is strongly recommended that you create a test cluster to validate the Operator is functioning as expected before moving on to the individual cluster upgrades.

## PostgreSQL Operator Automated Cluster Upgrade
    
Previously, the existing cluster upgrade focused on updating a cluster's underlying container images. However, due to the various changes in the PostgreSQL Operator's operation between the various versions (including numerous updates to the relevant CRDs, integration of Patroni for HA and other significant changes), updates between PostgreSQL Operator releases required the manual deletion of the existing clusters while preserving the underlying PVC storage. After installing the new PostgreSQL Operator version, the clusters could be recreated manually with the name of the new cluster matching the existing PVC's name.

The automated upgrade process provides a mechanism where, instead of being deleted, the existing PostgreSQL clusters will be left in place during the PostgreSQL Operator upgrade. While normal Operator functionality will be restricted on these existing clusters until they are upgraded to the currently installed PostgreSQL Operator version, the pods, services, etc will still be in place and accessible via other methods (e.g. kubectl, service IP, etc).
    
To upgrade a particular cluster, use
```    
pgo upgrade mycluster
``` 
This will follow a similar process to the documented manual process, where the pods, deployments, replicasets, pgtasks and jobs are deleted, the cluster's replicas are scaled down and replica PVCs deleted, but the primary PVC and backrest-repo PVC are left in place. Existing services for the primary, replica and backrest-shared-repo are also kept and will be updated to the requirements of the current version. Configmaps and secrets are kept except where deletion is required. For a cluster 'mycluster', the following configmaps will be deleted (if they exist) and recreated:
```    
mycluster-leader
mycluster-pgha-default-config
```
along with the following secret:
```    
mycluster-backrest-repo-config
```    

The pgcluster CRD will be read, updated automatically and replaced, at which point the normal cluster creation process will take over. The end result of the upgrade should be an identical numer of pods, deployments, replicas, etc with a new pgbackrest backup taken, but existing backups left in place.

Finally, to disable PostgreSQL version checking during the upgrade, such as for when container images are re-tagged and no longer follow the standard version tagging format, use the "ignore-validation" flag:

```
pgo upgrade mycluster --ignore-validation
```

That will allow the upgrade to proceed, regardless of the tag values. Please note, the underlying image must still be chosen in accordance with the [Considerations](/upgrade/automatedupgrade#considerations) listed above.
