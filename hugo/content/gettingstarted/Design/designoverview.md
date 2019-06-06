---
title: "Design"
date:
draft: false
weight: 3
---

## Provisioning

So, what does the Postgres Operator actually deploy
when you create a cluster?

![Reference](/OperatorReferenceDiagram.png)

On this diagram, objects with dashed lines are components
that are optionally deployed as part of a PostgreSQL Cluster
by the operator. Objects with solid lines are the fundamental and
required components.

For example, within the Primary Deployment, the *metrics* container
is completely optional. That component can be deployed using
either the operator configuration or command line arguments if you
want to cause metrics to be collected from the Postgres container.

Replica deployments are similar to the primary deployment but
are optional. A replica is not required to be created unless the
capability for one is necessary. As you scale up the Postgres
cluster, the standard set of components gets deployed and
replication to the primary is started.

Notice that each cluster deployment gets its own unique
Persistent Volumes. Each volume can use different storage
configurations which provides fined grained placement of
the database data files.

There is a Service created for the primary Postgres deployment
and a Service created for any replica Postgres deployments within
a given Postgres cluster.  Primary services match Postgres deployments
using a label *service-name* of the following format:

    service-name=mycluster
    service-name=mycluster-replica


## Custom Resource Definitions

Kubernetes Custom Resource Definitions are used in the design
of the PostgreSQL Operator to define the following:

 * Cluster - *pgclusters*
 * Backup - *pgbackups*
 * Policy - *pgpolicies*
 * Tasks - *pgtasks*

Metadata about the Postgres cluster deployments are stored within
these CRD resources which act as the source of truth for the
Operator.

The *postgres-operator* design incorporates the following concepts:

## Event Listeners

Kubernetes events are created for the Operator CRD resources when
new resources are made, deleted, or updated.  These events are
processed by the Operator to perform asynchronous actions.

As events are captured, controller logic is executed within the Operator
to perform the bulk of operator logic.

## REST API

A feature of the Operator is to provide a REST API upon which users
or custom applications can inspect and cause actions within the Operator
such as provisioning resources or viewing status of resources.

This API is secured by a RBAC (role based access control) security
model whereby each API call has a permission assigned to it.  API
roles are defined to provide granular authorization to Operator
services.

## Command Line Interface

One of the unique features of the Operator is
the pgo command line interface (CLI).  This tool is used by a normal end-user
to create databases or clusters, or make changes to existing databases.

The CLI interacts with the REST API deployed within the *postgres-operator* deployment.


## Node Affinity

You can have the Operator add a node affinity section to
a new Cluster Deployment if you want to cause Kubernetes to
attempt to schedule a primary cluster to a specific Kubernetes node.

You can see the nodes on your Kube cluster by running the following:
```
kubectl get nodes
```

You can then specify one of those names (e.g. kubeadm-node2)  when creating a cluster;
```
pgo create cluster thatcluster --node-label=kubeadm-node2
```

The affinity rule inserted in the Deployment uses a *preferred*
strategy so that if the node were down or not available, Kubernetes will
go ahead and schedule the Pod on another node.

When you scale up a Cluster and add a replica, the scaling will
take into account the use of `--node-label`.  If it sees that a
cluster was created with a specific node name, then the replica
Deployment will add an affinity rule to attempt to schedule

## Fail-over

Manual and automated fail-over are supported in the Operator
within a single Kubernetes cluster.

Manual failover is performed by API actions involving a *query*
and then a *target* being specified to pick the fail-over replica
target.

Automatic fail-over is performed by the Operator by evaluating
the readiness of a primary.  Automated fail-over can be globally
specified for all clusters or specific clusters.

Users can configure the Operator to replace a failed primary with
a new replica if they want that behavior.

The fail-over logic includes:

 * deletion of the failed primary Deployment
 * pick the best replica to become the new primary
 * label change of the targeted Replica to match the primary Service
 * execute the PostgreSQL promote command on the targeted replica

## pgbackrest Integration

The Operator integrates various features of the [pgbackrest backup and restore project](https://pgbackrest.org).  A key component added to the Operator
is the *pgo-backrest-repo* container, this container acts as a pgBackRest
remote repository for the Postgres cluster to use for storing archive
files and backups.

The following diagrams depicts some of the integration features:

![alt text](/operator-backrest-integration.png "Operator Backrest Integration")

In this diagram, starting from left to right we see the following:

 * a user when they enter *pgo backup mycluster --backup-type=pgbackrest* will cause a pgo-backrest container to be run as a Job, that container will execute a   *pgbackrest backup* command in the pgBackRest repository container to perform the backup function.

 * a user when they enter *pgo show backup mycluster --backup-type=pgbackrest* will cause a *pgbackrest info* command to be executed on the pgBackRest repository container, the *info* output is sent directly back to the user to view

 * the Postgres container itself will use an archive command, *pgbackrest archive-push* to send archives to the pgBackRest repository container

 * the user entering *pgo create cluster mycluster --pgbackrest* will cause
a pgBackRest repository container deployment to be created, that repository
is exclusively used for this Postgres cluster

 * lastly, a user entering *pgo restore mycluster* will cause a *pgo-backrest-restore* container to be created as a Job, that container executes the *pgbackrest restore* command

### pgbackrest Restore

The pgbackrest restore command is implemented as the *pgo restore* command.  This command is destructive in the sense that it is meant to *restore* a PG cluster meaning it will revert the PG cluster to a restore point that is kept in the pgbackrest repository.   The prior primary data is not deleted but left in a PVC to be manually cleaned up by a DBA.  The restored PG cluster will work against a new PVC created from the restore workflow.  

When doing a *pgo restore*, here is the workflow the Operator executes:

 * turn off autofail if it is enabled for this PG cluster
 * allocate a new PVC to hold the restored PG data
 * delete the the current primary database deployment
 * update the pgbackrest repo for this PG cluster with a new data path of the new PVC
 * create a pgo-backrest-restore job, this job executes the *pgbackrest restore* command from the pgo-backrest-restore container, this Job mounts the newly created PVC
 * once the restore job completes, a new primary Deployment is created which mounts the restored PVC volume

At this point the PG database is back in a working state.  DBAs are still responsible to re-enable autofail using *pgo update cluster* and also perform a pgBackRest backup after the new primary is ready.  This version of the Operator also does not handle any errors in the PG replicas after a restore, that is left for the DBA to handle.

Other things to take into account before you do a restore:

 * if a schedule has been created for this PG cluster, delete that schedule prior to performing a restore
 * If your database has been paused after the target restore was completed, then you would need to run the psql command select pg_wal_replay_resume() to complete the recovery, on PG 9.6⁄9.5 systems, the command you will use is select pg_xlog_replay_resume(). You can confirm the status of your database by using the built in postgres admin functions found [here:] (https://www.postgresql.org/docs/current/functions-admin.html#FUNCTIONS-RECOVERY-CONTROL-TABLE)
 * a pgBackRest restore is destructive in the sense that it deletes the existing primary deployment for the cluster prior to creating a new deployment containing the restored primary database.  However, in the event that the pgBackRest restore job fails, the `pgo restore` command be can be run again, and instead of first deleting the primary deployment (since one no longer exists), a new primary will simply be created according to any options specified.  Additionally, even though the original primary deployment will be deleted, the original primary PVC will remain.
 * there is currently no Operator validation of user entered pgBackRest command options, you will need to make sure to enter these correctly, if not the pgBackRest restore command can fail.
 * the restore workflow does not perform a backup after the restore nor does it verify that any replicas are in a working status after the restore, it is possible you might have to take actions on the replica to get them back to replicating with the new restored primary.
 * pgbackrest.org suggests running a pgbackrest backup after a restore, this needs to be done by the DBA as part of a restore
 * when performing a pgBackRest restore, the **node-label** flag can be utilized to target a specific node for both the pgBackRest restore job and the new (i.e. restored) primary deployment that is then created for the cluster.  If a node label is not specified, the restore job will not target any specific node, and the restored primary deployment will inherit any node labels defined for the original primary deployment.

### pgbackrest AWS S3 Support

The Operator supports the use AWS S3 storage buckets for the pgbackrest repository in any pgbackrest-enabled cluster.  When S3 support is enabled for a cluster, all archives will automatically be pushed to a pre-configured S3 storage bucket, and that same bucket can then be utilized for the creation of any backups as well as when performing restores.  Please note that once a storage type has been selected for a cluster during cluster creation (specifically `local`, `s3`, or _both_, as described in detail below), it cannot be changed.    

The Operator allows for the configuration of a single storage bucket, which can then be utilized across multiple clusters.  Once S3 support has been enabled for a cluster, pgbackrest will create a `backrestrepo` directory in the root of the configured S3 storage bucket (if it does not already exist), and subdirectories will then be created under the `backrestrepo` directory for each cluster created with S3 storage enabled.

#### S3 Configuration

In order to enable S3 storage, you must provide the required AWS S3 configuration information prior to deploying the Operator.  First, you will need to add the proper S3 bucket name, AWS S3 endpoint and AWS S3 region to the `Cluster` section of the `pgo.yaml` configuration file (additional information regarding the configuration of the `pgo.yaml` file can be found [here](/configuration/pgo-yaml-configuration/))  :

```yaml
Cluster:
  BackrestS3Bucket: containers-dev-pgbackrest
  BackrestS3Endpoint: s3.amazonaws.com
  BackrestS3Region: us-east-1
```

You will then need to specify the proper credentials for authenticating into the S3 bucket specified by adding a **key** and **key secret** to the `$PGOROOT/pgo-backrest-repo/aws-s3-credentials.yaml` configuration file:

```yaml
---
aws-s3-key: ABCDEFGHIJKLMNOPQRST
aws-s3-key-secret: ABCDEFG/HIJKLMNOPQSTU/VWXYZABCDEFGHIJKLM
```

Once the above configuration details have been provided, you can deploy the Operator per the [PGO installation instructions](/installation/operator-install/).  

#### Enabling S3 Storage in a Cluster

With S3 storage properly configured within your PGO installation, you can now select either local storage, S3 storage, or _both_ when creating a new cluster.  The type of storage selected upon creation of the cluster will determine the type of storage that can subsequently be used when performing pgbackrest backups and restores.  A storage type is specified using the `--pgbackrest-storage-type` flag, and can be one of the following values:

* `local` - pgbackrest will use volumes local to the container (e.g. Persistent Volumes) for storing archives, creating backups and locating backups for restores.  This is the default value for the `--pgbackrest-storage-type` flag.
* `s3` - pgbackrest will use the pre-configured AWS S3 storage bucket for storing archives, creating backups and locating backups for restores
* `local,s3` (both) - pgbackrest will use both volumes local to the container (e.g. persistent volumes), as well as the pre-configured AWS S3 storage bucket, for storing archives.  Also allows the use of local and/or S3 storage when performing backups and restores.

For instance, the following command enables both `local` and `s3` storage in a new cluster:

```bash
pgo create cluster mycluster --pgbackrest --pgbackrest-storage-type=local,s3 -n pgouser1
```

As described above, this will result in pgbackrest pushing archives to both local and S3 storage, while also allowing both local and S3 storage to be utilized for backups and restores.  However, you could also enable S3 storage only when creating the cluster:

```bash
pgo create cluster mycluster --pgbackrest --pgbackrest-storage-type=s3 -n pgouser1
```

Now all archives for the cluster will be pushed to S3 storage only, and local storage will not be utilized for storing archives (nor can local storage be utilized for backups and restores).

#### Using S3 to Backup & Restore

As described above, once S3 storage has been enabled for a cluster, it can also be used when backing up or restoring a cluster.  Here a both local and S3 storage is selected when performing a backup:

```bash
pgo backup mycluster --backup-type=pgbackrest --pgbackrest-storage-type=local,s3 -n pgouser1
```

This results in pgbackrest creating a backup in a local volume (e.g. a persistent volume), while also creating an additional backup in the configured S3 storage bucket.  However, a backup can be created using S3 storage only:

```bash
pgo backup mycluster --backup-type=pgbackrest --pgbackrest-storage-type=s3 -n pgouser1
```

Now pgbackrest will only create a backup in the S3 storage bucket only.

When performing a restore, either `local` or `s3` must be selected (selecting both for a restore will result in an error).  For instance, the following command specifies S3 storage for the restore:

```bash
pgo restore mycluster --pgbackrest-storage-type=s3 -n pgouser1
```

This will result in a full restore utilizing the backups and archives stored in the configured S3 storage bucket.

_Please note that because `local` is the default storage type for the `backup` and `restore` commands, `s3` must be explicitly set using the `--pgbackrest-storage-type` flag when performing backups and restores on clusters where only S3 storage is enabled._

#### AWS Certificate Authority
The Operator installation includes a default certificate bundle that is utilized by default to establish trust between pgbackrest and the AWS S3 endpoint used for S3 storage.  Please modify or replace this certificate bundle as needed prior to deploying the Operator if another certificate authority is needed to properly establish trust between pgbackrest and your S3 endpoint.  

The certificate bundle can be found here: `$PGOROOT/pgo-backrest-repo/aws-s3-ca.crt`.  

When modifying or replacing the certificate bundle, please be sure to maintain the same path and filename.

## PGO Scheduler

The Operator includes a cronlike scheduler application called `pgo-scheduler`.  Its purpose
is to run automated tasks such as PostgreSQL backups or SQL policies against PostgreSQL clusters
created by the Operator.

PGO Scheduler watches Kubernetes for configmaps with the label `crunchy-scheduler=true` in the
same namespace the Operator is deployed.  The configmaps are json objects that describe the schedule
such as:

* Cron like schedule such as: * * * * *
* Type of task: `pgbackrest`, `pgbasebackup` or `policy`

Schedules are removed automatically when the configmaps are deleted.

PGO Scheduler uses the `UTC` timezone for all schedules.

### Schedule Expression Format

Schedules are expressed using the following rules:

```
Field name   | Mandatory? | Allowed values  | Allowed special characters
----------   | ---------- | --------------  | --------------------------
Seconds      | Yes        | 0-59            | * / , -
Minutes      | Yes        | 0-59            | * / , -
Hours        | Yes        | 0-23            | * / , -
Day of month | Yes        | 1-31            | * / , - ?
Month        | Yes        | 1-12 or JAN-DEC | * / , -
Day of week  | Yes        | 0-6 or SUN-SAT  | * / , - ?
```

### pgBackRest Schedules

pgBackRest schedules require pgBackRest enabled on the cluster to backup.  The scheduler
will not do this on its own.

### pgBaseBackup Schedules

pgBaseBackup schedules require a backup PVC to already be created.  The operator will make
this PVC using the backup commands:

    pgo backup mycluster

### Policy Schedules

Policy schedules require a SQL policy already created using the Operator.  Additionally users
can supply both the database in which the policy should run and a secret that contains the username
and password of the PostgreSQL role that will run the SQL.  If no user is specified the scheduler will default to the replication user provided during cluster creation.


## Custom Resource Definitions

The Operator makes use of custom resource definitions to maintain state
and resource definitions as offered by the Operator. 

![Reference](/operator-crd-architecture.png)

In this above diagram, the CRDs heavily used by the Operator include:

 * pgcluster - defines the Postgres cluster definition, new cluster requests
are captured in a unique pgcluster resource for that Postgres cluster
 * pgtask - workflow and other related administration tasks are captured within a set of pgtasks for a given pgcluster
 * pgbackup - when you run a pgbasebackup, a pgbackup is created to hold the workflow and status of the last backup job, this CRD will eventually be deprecated in favor of a more general pgtask resource
 * pgreplica - when you create a Postgres replica, a pgreplica CRD is created to define that replica


## Considerations for Multi-zone Cloud Environments

#### Overview
When using the Operator in a Kubernetes cluster consisting of nodes that span multiple zones, special consideration must be
taken to ensure all pods and the volumes they require are scheduled and provisioned within the same zone.  Specifically,
being that a pod is unable mount a volume that is located in another zone, any volumes that are dynamically provisioned must
be provisioned in a topology-aware manner according to the specific scheduling requirements for the pod. For instance, this
means ensuring that the volume containing the database files for the primary DB in a new PG cluster is provisioned in the
same zone as the node containing the PG primary pod that will be using it.

#### Default Behavior
By default, the Kubernetes scheduler will ensure any pods created that claim a specific volume via a PVC are scheduled on a
node in the same zone as that volume.  This is part of the
[multi-zone support](https://kubernetes.io/docs/setup/multiple-zones/) that is included in Kubernetes by default. However,
when using dynamic provisioning, volumes are not provisioned in a topology-aware manner by default, which means a volume
will not be provisioned according to the same scheduling requirements that will be placed on the pod that will be using it
(e.g. it will not consider node selectors, resource requirements, pod affinity/anti-affinity, and various other scheduling
requirements).  Rather, PVCs are immediately bound as soon as they are requested, which means volumes are provisioned 
without knowledge of these scheduling requirements. This behavior is the result of the `volumeBindingMode` defined on the
Storage Class being utilized to dynamically provision the volume, which is set to `Immediate` by default.  This can be seen
in the following Storage Class definition, which defines a Storage Class for a Google Cloud Engine Persistent Disk (GCE PD)
that uses the default value of `Immediate` for its `volumeBindingMode`:

```bash
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: example-sc
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
volumeBindingMode: Immediate
```

Unfortunately, using `Immediate` for the `volumeBindingMode` in a multi-zone cluster can result in undesired behavior when
using the Operator, being that the scheduler will ignore any requested _(but not mandatory)_ scheduling requirements if
necessary to ensure the pod can be scheduled. Specifically, the scheduler will ultimately schedule the pod on a node in the
same zone as the volume, even if another node was requested for scheduling that pod. For instance, a node label might be
specified using the `--node-label` option when creating a cluster using the `pgo create cluster` command in order target a 
specific node (or nodes) for the deployment of that cluster. Within the Operator, a **node label** is implemented as a 
`preferredDuringSchedulingIgnoredDuringExecution` node affinity rule, which is an affinity rule that Kubernetes will attempt
to adhere to when scheduling any pods for the cluster, but _will not guarantee_ (more information on node affinity rules can
be found [here](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity)). Therefore,
if the volume ends up in a zone other than the zone containing the node (or nodes) defined by the node label, the node label
will be ignored, and the pod will be scheduled according to the zone containing the volume.  

#### Topology Aware Volumes
In order to overcome the behavior described above in a multi-zone cluster, volumes must be made topology aware.  This is
accomplished by setting the `volumeBindingMode` for the storage class to `WaitForFirstConsumer`, which delays the dynamic
provisioning of a volume until a pod using it is created. In other words, the PVC is no longer bound as soon as it is 
requested, but rather waits for a pod utilizing it to be creating prior to binding.  This change ensures that volume can
take into account the scheduling requirements for the pod, which in the case of a multi-zone cluster means ensuring the
volume is provisioned in the same zone containing the node where the pod has be scheduled.  This also means the scheduler
should no longer ignore a node label in order to follow a volume to another zone when scheduling a pod, since the volume
will now follow the pod according to the pods specificscheduling requirements.  The following is an example of the the same
Storage Class defined above, only with `volumeBindingMode` now set to `WaitForFirstConsumer`:

```bash
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: example-sc
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
volumeBindingMode: WaitForFirstConsumer
```

#### Additional Solutions
If you are using a version of Kubernetes that does not support `WaitForFirstConsumer`, an alternate _(and now deprecated)_
solution exists in the form of parameters that can be defined on the Storage Class definition to ensure volumes are
provisioned in a specific zone (or zones).  For instance, when defining a Storage Class for a GCE PD for use in Google 
Kubernetes Engine (GKE) cluster, the **zone** parameter can be used to ensure any volumes dynamically provisioned using that
Storage Class are located in that specific zone.  The following is an example of a Storage Class for a GKE cluster that will
provision volumes in the **us-east1** zone:

```bash
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: example-sc
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
  replication-type: none
  zone: us-east1
```

Once storage classes have been defined for one or more zones, they can then be defined as one or more storage configurations
within the pgo.yaml configuration file (as described in the 
[PGO YAML configuration guide](/configuration/pgo-yaml-configuration)).  From there those storage configurations can then be
selected when creating a new cluster, as shown in the following example:

```bash
pgo create cluster mycluster --storage-config=example-sc
```

With this approach, the pod will once again be scheduled according to the zone in which the volume was provisioned. However,
the zone parameters defined on the Storage Class bring consistency to scheduling by guaranteeing that the volume, and
therefore also the pod using that volume, are scheduled in a specific zone as defined by the user, bringing consistency
and predictability to volume provisioning and pod scheduling in multi-zone clusters.

For more information regarding the specific parameters available for the Storage Classes being utilizing in your cloud 
environment, please see the
[Kubernetes documentation for Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/).

Lastly, while the above applies to the dynamic provisioning of volumes, it should be noted that volumes can also be manually
provisioned in desired zones in order to achieve the desired topology requirements for any pods and their volumes.
