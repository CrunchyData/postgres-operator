
---
title: "PGO YAML"
Latest Release: 4.0.1 {docdate}
draft: false
weight: 3
---

# pgo.yaml Configuration
The *pgo.yaml* file contains many different configuration settings as described in this section of the documentation.

The *pgo.yaml* file is broken into major sections as described below:
## Cluster 

| Setting |Definition  |
|---|---|
|BasicAuth        | if set to *true* will enable Basic Authentication
|PrimaryNodeLabel        |newly created primary deployments will specify this node label if specified, unless you override it using the --node-label command line flag, if not set, no node label is specifed
|ReplicaNodeLabel        |newly created replica deployments will specify this node label if specified, unless you override it using the --node-label command line flag, if not set, no node label is specifed
|CCPImagePrefix        |newly created containers will be based on this image prefix (e.g. crunchydata), update this if you require a custom image prefix
|CCPImageTag        |newly created containers will be based on this image version (e.g. centos7-11.4-2.4.1), unless you override it using the --ccp-image-tag command line flag
|Port        | the PostgreSQL port to use for new containers (e.g. 5432)
|LogStatement        | postgresql.conf log_statement value (required field)
|LogMinDurationStatement        | postgresql.conf log_min_duration_statement value (required field)
|User        | the PostgreSQL normal user name
|Database        | the PostgreSQL normal user database
|Replicas        | the number of cluster replicas to create for newly created clusters, typically users will scale up replicas on the pgo CLI command line but this global value can be set as well
|PgmonitorPassword        | the password to use for pgmonitor metrics collection if you specify --metrics when creating a PG cluster
|Metrics        | boolean, if set to true will cause each new cluster to include crunchy-collect as a sidecar container for metrics collection, if set to false (default), users can still add metrics on a cluster-by-cluster basis using the pgo command flag --metrics
|Badger        | boolean, if set to true will cause each new cluster to include crunchy-pgbadger as a sidecar container for static log analysis, if set to false (default), users can still add pgbadger on a cluster-by-cluster basis using the pgo create cluster command flag --pgbadger
|Policies        | optional, list of policies to apply to a newly created cluster, comma separated, must be valid policies in the catalog
|PasswordAgeDays        | optional, if set, will set the VALID UNTIL date on passwords to this many days in the future when creating users or setting passwords, defaults to 60 days
|PasswordLength        | optional, if set, will determine the password length used when creating passwords, defaults to 8
|ServiceType        | optional, if set, will determine the service type used when creating primary or replica services, defaults to ClusterIP if not set, can be overridden by the user on the command line as well
|Backrest        | optional, if set, will cause clusters to have the pgbackrest volume PVC provisioned during cluster creation
|BackrestPort        | currently required to be port 2022
|Autofail        | optional, if set, will cause clusters to be checked for auto failover in the event of a non-Ready status
|AutofailReplaceReplica        | optional, default is false, if set, will determine whether a replica is created as part of a failover to replace the promoted replica, the AutofailReplaceReplica setting in *pgo.yaml* is overrode with this command line flag if specified by a user.

## Storage
| Setting|Definition  |
|---|---|
|PrimaryStorage    |required, the value of the storage configuration to use for the primary PostgreSQL deployment
|BackupStorage    |required, the value of the storage configuration to use for backups, including the storage for pgbackrest repo volumes
|ReplicaStorage    |required, the value of the storage configuration to use for the replica PostgreSQL deployments
|ReplicaStorage    |required, the value of the storage configuration to use for the replica PostgreSQL deployments
|BackrestStorage    |required, the value of the storage configuration to use for the pgbackrest shared repository deployment created when a user specifies pgbackrest to be enabled on a cluster
|StorageClass        |for a dynamic storage type, you can specify the storage class used for storage provisioning(e.g. standard, gold, fast)
|AccessMode        |the access mode for new PVCs (e.g. ReadWriteMany, ReadWriteOnce, ReadOnlyMany). See below for descriptions of these.
|Size        |the size to use when creating new PVCs (e.g. 100M, 1Gi)
|Storage.storage1.StorageType        |supported values are either *dynamic*,  *create*,  if not supplied, *create* is used
|Fsgroup        | optional, if set, will cause a *SecurityContext* and *fsGroup* attributes to be added to generated Pod and Deployment definitions
|SupplementalGroups        | optional, if set, will cause a SecurityContext to be added to generated Pod and Deployment definitions
|MatchLabels        | optional, if set, will cause the PVC to add a *matchlabels* selector in order to match a PV, only useful when the StorageType is *create*, when specified a label of *key=value* is added to the PVC as a match criteria

## Storage Configuration Examples
In *pgo.yaml*, you will need to configure your storage configurations
depending on which storage you are wanting to use for
Operator provisioning of Persistent Volume Claims.  The examples
below are provided as a sample.  In all the examples you are
free to change the *Size* to meet your requirements of Persistent
Volume Claim size.

### HostPath Example

HostPath is provided for simple testing and use
cases where you only intend to run on a single
Linux host for your Kubernetes cluster.

```
  hostpathstorage:
    AccessMode:  ReadWriteMany
    Size:  1G
    StorageType:  create
```

### NFS Example

In the following NFS example, notice that the
*SupplementalGroups* setting is set, this can
be whatever GID you have your NFS mount set
to, typically we set this *nfsnobody* as below.
NFS file systems offer a *ReadWriteMany* access
mode.

```
  nfsstorage:
    AccessMode:  ReadWriteMany
    Size:  1G
    StorageType:  create
    SupplementalGroups:  65534
```

### Storage Class Example

In the following example, the important attribute to
set for a typical Storage Class is the  *Fsgroup* setting.
This value is almost always set to *26* which represents
the Postgres user ID that the Crunchy Postgres container
runs as.  Most Storage Class providers offer *ReadWriteOnce*
access modes, but refer to your provider documentation
for other access modes it might support.

```
  storageos:
    AccessMode:  ReadWriteOnce
    Size:  1G
    StorageType:  dynamic
    StorageClass:  fast
    Fsgroup:  26
```


## Container Resources
| Setting |Definition  |
|---|---|
|DefaultContainerResource    |optional, the value of the container resources configuration to use for all database containers, if not set, no resource limits or requests are added on the database container
|DefaultLoadResource    |optional, the value of the container resources configuration to use for pgo-load containers, if not set, no resource limits or requests are added on the database container
|DefaultLspvcResource    |optional, the value of the container resources configuration to use for pgo-lspvc containers, if not set, no resource limits or requests are added on the database container
|DefaultRmdataResource    |optional, the value of the container resources configuration to use for pgo-rmdata containers, if not set, no resource limits or requests are added on the database container
|DefaultBackupResource    |optional, the value of the container resources configuration to use for crunchy-backup containers, if not set, no resource limits or requests are added on the database container
|DefaultPgbouncerResource    |optional, the value of the container resources configuration to use for crunchy-pgbouncer containers, if not set, no resource limits or requests are added on the database container
|DefaultPgpoolResource    |optional, the value of the container resources configuration to use for crunchy-pgpool containers, if not set, no resource limits or requests are added on the database container
|RequestsMemory        | request size of memory in bytes
|RequestsCPU        | request size of CPU cores
|LimitsMemory        | request size of memory in bytes
|LimitsCPU        | request size of CPU cores

## Miscellaneous (Pgo)
| Setting |Definition  |
|---|---|
|PreferredFailoverNode        | optional, a label selector (e.g. hosttype=offsite) that if set, will be used to pick the failover target which is running on a host that matches this label if multiple targets are equal in replication status
|COImagePrefix        | image tag prefix to use for the Operator containers
|COImageTag        | image tag to use for the Operator containers
|Audit        | boolean, if set to true will cause each apiserver call to be logged with an *audit* marking

## Storage Configuration Details

You can define n-number of Storage configurations within the *pgo.yaml* file. Those Storage configurations follow these conventions -

 * they must have lowercase name (e.g. storage1)
 * they must be unique names (e.g. mydrstorage, faststorage, slowstorage)

These Storage configurations are referenced in the BackupStorage, ReplicaStorage, and PrimaryStorage configuration values. However, there are command line
options in the *pgo* client that will let a user override these default global values to offer you the user a way to specify very targeted storage configurations when needed (e.g. disaster recovery storage for certain backups).

You can set the storage AccessMode values to the following:

* *ReadWriteMany* - mounts the volume as read-write by many nodes
* *ReadWriteOnce* - mounts the PVC as read-write by a single node
* *ReadOnlyMany* - mounts the PVC as read-only by many nodes

These Storage configurations are validated when the *pgo-apiserver* starts, if a
non-valid configuration is found, the apiserver will abort.  These Storage values are only read at *apiserver* start time.

The following StorageType values are possible -

 * *dynamic* - this will allow for dynamic provisioning of storage using a StorageClass.
 * *create* - This setting allows for the creation of a new PVC for each PostgreSQL cluster using a naming convention of *clustername*.  When set, the *Size*, *AccessMode* settings are used in constructing the new PVC.

The operator will create new PVCs using this naming convention: *dbname* where *dbname* is the database name you have specified.  For example, if you run:

    pgo create cluster example1 -n pgouser1

It will result in a PVC being created named *example1* and in the case of a backup job, the pvc is named *example1-backup*

Note, when Storage Type is *create*, you can specify a storage configuration setting of *MatchLabels*, when set, this will cause a *selector* of *key=value* to be added into the PVC, this will let you target specific PV(s) to be matched for this cluster. Note, if a PV does not match the claim request, then the cluster will not start.  Users
that want to use this feature have to place labels on their PV resources as part of PG cluster creation before creating the PG cluster.  For example, users would add a label like this to their PV before they create the PG cluster:

    kubectl label pv somepv myzone=somezone -n pgouser1

If you do not specify *MatchLabels* in the storage configuration, then no match filter is added and any available PV will be used to satisfy the PVC request.  This option does not apply to *dynamic* storage types.

Example PV creation scripts are provided that add labels to a set of PVs and can be used for testing:  `$COROOT/pv/create-pv-nfs-labels.sh`
 in that example, a label of **crunchyzone=red** is set on a set of PVs to test with.

The *pgo.yaml* includes a storage config named **nfsstoragered** that when used will demonstrate the label matching.  This feature allows you to support
n-number of NFS storage configurations and supports spreading a PG cluster across different NFS storage configurations.

## Container Resources Details
In the *pgo.yaml* configuration file you have the option to configure a default container resources configuration that when set will add CPU and memory resource limits and requests values into each database container when the container is created.

You can also override the default value using the `--resources-config` command flag when creating a new cluster:

    pgo create cluster testcluster --resources-config=large -n pgouser1

Note, if you try to allocate more resources than your
host or Kube cluster has available then you will see your
pods wait in a *Pending* status. The output from a `kubectl describe pod`
command will show output like this in this event:
Events:

      Type     Reason            Age               From               Message
      ----     ------            ----              ----               -------
      Warning  FailedScheduling  49s (x8 over 1m)  default-scheduler  No nodes are available that match all of the predicates: Insufficient memory (1).

## Overriding Storage Configuration Defaults
 

    pgo create cluster testcluster --storage-config=bigdisk -n pgouser1

 

That example will create a cluster and specify a storage configuration of *bigdisk* to be used for the primary database storage. The replica storage will default to the value of ReplicaStorage as specified in *pgo.yaml*.

    pgo create cluster testcluster2 --storage-config=fastdisk --replica-storage-config=slowdisk -n pgouser1

That example will create a cluster and specify a storage configuration of *fastdisk* to be used for the primary database storage, while the replica storage will use the storage configuration *slowdisk*.

    pgo backup testcluster --storage-config=offsitestorage -n pgouser1

That example will create a backup and use the *offsitestorage* storage configuration for persisting the backup.

## Using Storage Configurations for Disaster Recovery
A simple mechanism for partial disaster recovery can be obtained by leveraging network storage, Kubernetes storage classes, and the storage configuration options within the Operator.

For example, if you define a Kubernetes storage class that refers to a storage backend that is running within your disaster recovery site, and then use that storage class as
a storage configuration for your backups, you essentially have moved your backup files automatically to your disaster recovery site thanks to network storage.

