---
title: "PGO v4 to PGO v5"
date:
draft: false
weight: 100
---

You can upgrade from PGO v4 to PGO v5 through a variety of methods by following this guide. There are several methods that can be used to upgrade: we present these methods based upon a variety of factors, including:

- Redundancy / ability to roll back
- Available resources
- Downtime preferences

and others.

These methods include:

- [*Migrating Using Data Volumes*](#upgrade-method-1-data-volumes). This allows you to migrate from v4 to v5 using the existing data volumes that you created in v4. This is the simplest method for upgrade and is the most resource efficient, but you will have a greater potential for downtime using this method.
- [*Migrate From Backups*](#upgrade-method-2-backups). This allows you to create a Postgres cluster with v5 from the backups taken with v4. This provides a way for you to create a preview of your Postgres cluster through v5, but you would need to take your applications offline to ensure all the data is migrated.
- [*Migrate Using a Standby Cluster*](#upgrade-method-3-standby-cluster). This allows you to run a v4 and a v5 Postgres cluster in parallel, with data replicating from the v4 cluster to the v5 cluster. This method minimizes downtime and lets you preview your v5 environment, but is the most resource intensive.

You should choose the method that makes the most sense for your environment. Each method is described in detail below.

## Prerequisites

There are several prerequisites for using any of these upgrade methods.

- PGO v4 is currently installed within the Kubernetes cluster, and is actively managing any existing v4 PostgreSQL clusters.
- Any PGO v4 clusters being upgraded have been properly initialized using PGO v4, which means the v4 `pgcluster` custom resource should be in a `pgcluster Initialized` status:

```
$ kubectl get pgcluster hippo -o jsonpath='{ .status }'
{"message":"Cluster has been initialized","state":"pgcluster Initialized"}
```

- The PGO v4 `pgo` client is properly configured and available for use.
- PGO v5 is currently [installed]({{< relref "installation/_index.md" >}}) within the Kubernetes cluster.

For these examples, we will use a Postgres cluster named `hippo`.

## Upgrade Method #1: Data Volumes

This upgrade method allows you to migrate from PGO v4 to PGO v5 using the existing data volumes that were created in PGO v4. Note that this is an "in place" migration method: this will immediately move your Postgres clusters from being managed by PGO v4 and PGO v5. If you wish to have some failsafes in place, please use one of the other migration methods.

### Step 1: Prepare the PGO v4 Cluster for Migration

You will need to set up your PGO v4 Postgres cluster so that it can be migrated to a PGO v5 cluster. The following describes how to set up a PGO v4 cluster for using this migration method.

1. Scale down any existing replicas within the cluster.  This will ensure that the primary PVC does not change again prior to the upgrade.

You can get a list of replicas using the `pgo scaledown --query` command, e.g.:
```
pgo scaledown hippo --query  
```

If there are any replicas, you will see something similar to:

```
Cluster: hippo
REPLICA                 STATUS          NODE ...   
hippo                   running         node01 ...
```

Scaledown any replicas that are running in this cluser, e.g.:

```
pgo scaledown hippo --target=hippo
```

2\. Once all replicas are removed and only the primary remains, proceed with deleting the cluster while retaining the data and backups. You can do this `--keep-data` and  `--keep-backups` flags:

**You MUST run this command with the `--keep-data` and `--keep-backups` flag otherwise you risk deleting ALL of your data.**

```
pgo delete cluster hippo --keep-data --keep-backups
```

3\. The PVC for the primary Postgres instance and the pgBackRest repository should still remain. You can verify this with the command below:

```
kubectl get pvc --selector=pg-cluster=hippo
```

This should yield something similar to:

```
NAME              STATUS   VOLUME ...
hippo-jgut        Bound    pvc-a0b89bdb- ...
hippo-pgbr-repo   Bound    pvc-25501671- …
```

A third PVC used to store write-ahead logs (WAL) may also be present if external WAL volumes were enabled for the cluster.

### Step 2: Migrate to PGO v5

With the PGO v4 cluster's volumes prepared for the move to PGO v5, you can now create a [`PostgresCluster`]({{< relref "references/crd.md" >}}) custom resource using these volumes. This migration method does not carry over any specific configurations or customizations from PGO v4: you will need to create the specific `PostgresCluster` configuration that you need.

To complete the upgrade process, your `PostgresCluster` custom resource **MUST** include the following:

1\. An `existingVolumes` data source that points to the PostgreSQL data, PostgreSQL WAL (if applicable) and pgBackRest repository PVCs identified in the `spec.dataSource.existingVolumes` section.

For example, using the `hippo` cluster:

```
spec:
  dataSource:
    existingVolumes:
      existingPGDataVolume:
        pvcName: hippo-jgut
        directory: "hippo-jgut"
      existingPGBackRestVolume:
        pvcName: hippo-pgbr-repo
        directory: "hippo-backrest-shared-repo"
      # only specify external WAL PVC if enabled in v4 cluster
      # existingPGWALVolume:
      #  pvcName: hippo-wal
```

Please see the [Data Migration]({{< relref "guides/data-migration.md" >}}) section of the [tutorial]({{< relref "tutorial/_index.md" >}}) for more details on how to properly populate this section of the spec when migrating from a PGO v4 cluster.

2\. If you are using the default setup in your PGO v4 cluster, you will need to provide custom setup parameters to include the [`pgAudit`](https://github.com/pgaudit/pgaudit) extension extension. This looks similar to the following:

```
patroni:
  dynamicConfiguration:
    postgresql:
      parameters:
        shared_preload_libraries: pgaudit.so
```

If you customized other Postgres parameters, you will need to ensure they match in the PGO v5 cluster. For more information, please review the tutorial on [customizing a Postgres cluster]({{< relref "tutorial/customize-cluster.md" >}}).

3\. Once the `PostgresCluster` spec is populated according to these guidelines, you can create the `PostgresCluster` custom resource.  For example, if the `PostgresCluster` you're creating is a modified version of the [`postgres` example](https://github.com/CrunchyData/postgres-operator-examples/tree/main/kustomize/postgres) in the [PGO examples repo](https://github.com/CrunchyData/postgres-operator-examples), you can run the following command:

```
kubectl apply -k examples/postgrescluster
```

Your upgrade is now complete! For more information on how to use PGO v5, we recommend reading through the [PGO v5 tutorial]({{< relref "tutorial/_index.md" >}}).

## Upgrade Method #2: Backups

This upgrade method allows you to migrate from PGO v4 to PGO v5 by creating a new PGO v5 Postgres cluster using a backup from a PGO v4 cluster. This method allows you to preserve the data in your PGO v4 cluster while you transition to PGO v5. To fully move the data over, you will need to incur downtime and shut down your PGO v4 cluster.

*NOTE*: External WAL volumes **MUST** be enabled for the PGO v4 cluster being upgraded.  Additionally, the backup that will be used to initialize the PGO v5 cluster **MUST** be created with external WAL volumes.

If you did not create your cluster with an external WAL volume (`pgo create cluster --wal-storage-config`), you can do so using the following command. Note that this involves a cluster deletion with the `-keep-data` flag::

```
pgo delete cluster hippo --keep-data
# wait for deletion to complete...
pgo create cluster hippo --wal-storage-config= ...
```

### Step 1: Prepare the PGO v4 Cluster for Migration

1. Ensure you have a recent backup of your cluster. You can do so with the `pgo backup` command, e.g.:

```
pgo backup hippo
```

Please ensure that the backup completes. You will see the latest backup appear using the `pgo show backup` command.

2. If are using a pgBackRest repository that is using S3 (or a S3-like storage system) or GCS, you can delete the cluster while keeping the backups (using the `--keep-backups` flag) and skip ahead to the [Migrate to PGO v5](#step-2-migrate-to-pgo-v5-1) section:

```
pgo delete cluster hippo --keep-backups
```

Otherwise, if you are using a PVC-based pgBackRest repository for your PGO v4 cluster to create the PGO v5 cluster, shut down and continue following the directions in this section:

```
pgo update cluster hippo --shutdown
```

Wait for the shutdown to complete.

3. At this point, the pgBackRest dedicated repository host should no longer be running. Scale the dedicated pgBackRest repo host Deployment back up in order to adjust the repository permissions that are required for the PGO v5 migration:

```
kubectl scale deployment hippo-backrest-shared-repo --replicas=1
```

The Deployment is named following the pattern `<clusterName>-backrest-shared-repo`.

4\. Identify the name of the pgBackRest repo Pod. You can do so with the following command:

```
kubectl get pod --selector=pg-cluster=hippo,pgo-backrest-repo=true -o name
```

For convenience, you can store this value to an environmental variable:

```
BACKREST_POD_NAME=($kubectl get pod --selector=pg-cluster=hippo,pgo-backrest-repo=true -o name)
```

5\. The PGO v5 Postgres cluster will need to be able to access the pgBackRest repository data. Exec into the pgBackRest repository host and grant group ownership for the pgBackRest repository to the `postgres` group and group read/write access to the repository:

```
kubectl exec -it "${BACKREST_POD_NAME}" -- \
  chown -R pgbackrest:postgres /backrestrepo/hippo-backrest-shared-repo
kubectl exec -it "${BACKREST_POD_NAME}" -- \
  chmod -R g+rw /backrestrepo/hippo-backrest-shared-repo
```

Note that the directory name should match the Deployment name.

6\. You can delete the cluster while keeping the backups (using the `--keep-backups` flag):

```
pgo delete cluster hippo --keep-backups
```

7\. At this point, only the PVC for the pgBackRest repository should remain. You can verify this with the following command, e.g.:

```
kubectl get pvc --selector=pg-cluster=hippo
```

which should yield something similar to:

```
NAME              STATUS   VOLUME ...
	hippo-pgbr-repo   Bound    pvc-25501671- ...
```

You will need to relabel this PVC to match the expected labels in PGO v5. First, remove the PGO v4 labels:

```
kubectl label pvc --selector=pg-cluster=hippo vendor- pg-cluster-
```

Add the PGO v5 labels. Substitute "hippo" with the name of your cluster:

```
kubectl label pvc hippo-pgbr-repo \
  postgres-operator.crunchydata.com/cluster=hippo \
  postgres-operator.crunchydata.com/pgbackrest-repo=repo1 \
  postgres-operator.crunchydata.com/pgbackrest-volume= \
  postgres-operator.crunchydata.com/pgbackrest=
```

### Step 2: Migrate to PGO v5

With the PGO v4 Postgres cluster's backup repository prepared, you can now create a [`PostgresCluster`]({{< relref "references/crd.md" >}}) custom resource. This migration method does not carry over any specific configurations or customizations from PGO v4: you will need to create the specific `PostgresCluster` configuration that you need.

To complete the upgrade process, your `PostgresCluster` custom resource **MUST** include the following:

1\. You will need to configure your pgBackRest repository based upon whether you are using a PVC to store your backups, or an object storage system such as S3/GCS. Please follow the directions based upon the repository type you are using as part of the migration.

#### PVC-based Backup Repository

When migrating from a PVC-based backup repository, you will need to configure a pgBackRest repo of a `spec.backups.pgbackrest.repos.volume` under the `spec.backups.pgbackrest.repos.name` of `repo1`. The `volumeClaimSpec` should match the attributes of the pgBackRest repo PVC being used as part of the migration, i.e. it must have the same `storageClassName`, `accessModes`, `resources`, etc. For example:

```
spec:
  backups:
    pgbackrest:
      repos:
        - name: repo1
          volume:
            volumeClaimSpec:
              storageClassName: standard-wffc
              accessModes:
              - "ReadWriteOnce"
              resources:
                requests:
                  storage: 1Gi
```

Please ensure that the `pgbackrest-repo-path` configured for this repository includes the name of the repository used by the PGO v4 cluster. The default name for a PGO v4 repository follows the pattern `<clusterName>-backrest-shared-repo`. This is then is mounted to a path that follows the format `/pgbackrest/repo1/<clusterName>-backrest-shared-repo`.

Using the `hippo` Postgres cluster as an example, you would set the following in the `spec.backups.pgbackrest.global` section:

```
spec:
  backups:
    pgbackrest:
      global:
        repo1-path: /pgbackrest/repo1/hippo-backrest-shared-repo
```

(This can also be set via ConfigMaps or Secrets as well. Please see the [Backup Configuration]({{< relref "tutorial/backups.md" >}}) for more information).

#### S3 / GCS Backup Repository

When migrating from a S3 or GCS based backup repository, you will need to configure your `spec.backups.pgbackrest.repos.volume` to point to the backup storage system. For instance, if AWS S3 storage is being utilized, the repo would be defined similar to the following:

```
spec:
  backups:
    pgbackrest:
      repos:
        - name: repo1
          s3:
            bucket: hippo
            endpoint: s3.amazonaws.com
            region: us-east-1
```

Any required secrets or desired custom pgBackRest configuration should be created and configured as described in the [backup tutorial]({{< relref "tutorial/backups.md" >}}).

You will also need to ensure that the “pgbackrest-repo-path” configured for the repository matches the path used by the PGO v4 cluster. The default repository path follows the pattern `/backrestrepo/<clusterName>-backrest-shared-repo`. Note that the path name here is different than migrating from a PVC-based repository.

Using the `hippo` Postgres cluster as an example, you would set the following in the `spec.backups.pgbackrest.global` section:

```
spec:
  backups:
    pgbackrest:
      global:
        repo1-path: /backrestrepo/hippo-backrest-shared-repo
```

2\. Set the `spec.dataSource` section to restore from the backups used for this migration. For example:

```
spec:
  dataSource:
    postgresCluster:
      repoName: repo1
```

You can also provide other pgBackRest restore options, e.g. if you wish to restore to a specific point-in-time (PITR).

3\.  If you are using the default setup in your PGO v4 cluster, you will need to provide custom setup parameters to include the [`pgAudit`](https://github.com/pgaudit/pgaudit) extension extension. This looks similar to the following:
```
patroni:
  dynamicConfiguration:
    postgresql:
      parameters:
        shared_preload_libraries: pgaudit.so
```

Once the `PostgresCluster` spec is populated according to these guidelines, you can create the `PostgresCluster` custom resource.  For example, if the `PostgresCluster` you're creating is a modified version of the [`postgres` example](https://github.com/CrunchyData/postgres-operator-examples/tree/main/kustomize/postgres) in the [PGO examples repo](https://github.com/CrunchyData/postgres-operator-examples), you can run the following command:

```
kubectl apply -k examples/postgrescluster
```

3\. If you are using the default setup in your PGO v4 cluster, you will need to provide custom setup parameters to include the pgAudit extension. This looks similar to the following: v4):

```
spec:
	patroni:
    dynamicConfiguration:
      postgresql:
        parameters:
          shared_preload_libraries: pgaudit.so
```

If you customized other Postgres parameters, you will need to ensure they match in the PGO v5 cluster. For more information, please review the tutorial on [customizing a Postgres cluster]({{< relref "tutorial/customize-cluster.md" >}}).

4\. Once the `PostgresCluster` spec is populated according to these guidelines, you can create the `PostgresCluster` custom resource.  For example, if the `PostgresCluster` you're creating is a modified version of the [`postgres` example](https://github.com/CrunchyData/postgres-operator-examples/tree/main/kustomize/postgres) in the [PGO examples repo](https://github.com/CrunchyData/postgres-operator-examples), you can run the following command:

```
kubectl apply -k examples/postgrescluster
```

**WARNING**: Once the PostgresCluster custom resource is created, it will become the owner of the PVC.  *This means that if the PostgresCluster is then deleted (e.g. if attempting to revert back to a PGO v4 cluster), then the PVC will be deleted as well.*

If you wish to protect against this, relabel the PVC prior to deleting the PostgresCluster custom resource. Below uses the `hippo` Postgres cluster as an example:

```
kubectl label pvc hippo-pgbr-repo \
  postgres-operator.crunchydata.com/cluster- \
  postgres-operator.crunchydata.com/pgbackrest-repo- \
  postgres-operator.crunchydata.com/pgbackrest-volume- \
  postgres-operator.crunchydata.com/pgbackrest-
```

You will also need to remove all ownership references from the PVC:

```
kubectl patch pvc hippo-pgbr-repo --type='json' -p='[{"op": "remove", "path": "/metadata/ownerReferences"}]'
```

It is recommended to set the reclaim policy for any PV’s bound to existing PVC’s to `Retain` to ensure data is retained in the event a PVC is accidentally deleted during the upgrade.

Your upgrade is now complete! For more information on how to use PGO v5, we recommend reading through the [PGO v5 tutorial]({{< relref "tutorial/_index.md" >}}).

## Upgrade Method #3: Standby Cluster

This upgrade method allows you to migrate from PGO v4 to PGO v5 by creating a new PGO v5 Postgres cluster in a "standby" mode, allowing it to mirror the PGO v4 cluster and continue to receive data updates in real time. This has the advantage of being able to fully inspect your PGO v5 Postgres cluster while leaving your PGO v4 cluster up and running, thus minimizing downtime when you cut over. The tradeoff is that you will temporarily use more resources while this migration is occurring.

This method only works if your PGO v4 cluster uses S3 or an S3-compatible storage system, or GCS. For more information on [standby clusters]({{< relref "tutorial/disaster-recovery.md" >}}#standby-cluster), please refer to the [tutorial](({{< relref "tutorial/disaster-recovery.md" >}}#standby-cluster)).

*NOTE*: External WAL volumes **MUST** be enabled for the PGO v4 cluster being upgraded.  Additionally, the backup that will be used to initialize the PGO v5 cluster **MUST** be created with external WAL volumes.

If you did not create your cluster with an external WAL volume (`pgo create cluster --wal-storage-config`), you can do so using the following command. Note that this involves a cluster deletion with the `-keep-data` flag::

```
pgo delete cluster hippo --keep-data
# wait for deletion to complete...
pgo create cluster hippo --wal-storage-config= ...
```

### Step 1: Migrate to PGO v5

Create a [`PostgresCluster`]({{< relref "references/crd.md" >}}) custom resource. This migration method does not carry over any specific configurations or customizations from PGO v4: you will need to create the specific `PostgresCluster` configuration that you need.

To complete the upgrade process, your `PostgresCluster` custom resource **MUST** include the following:

1\. Configure your pgBackRest to use an object storage system such as S3/GCS. You will need to configure your `spec.backups.pgbackrest.repos.volume` to point to the backup storage system. For instance, if AWS S3 storage is being utilized, the repo would be defined similar to the following:

```
spec:
  backups:
    pgbackrest:
      repos:
        - name: repo1
          s3:
            bucket: hippo
            endpoint: s3.amazonaws.com
            region: us-east-1
```

Any required secrets or desired custom pgBackRest configuration should be created and configured as described in the [backup tutorial]({{< relref "tutorial/backups.md" >}}).

You will also need to ensure that the “pgbackrest-repo-path” configured for the repository matches the path used by the PGO v4 cluster. The default repository path follows the pattern `/backrestrepo/<clusterName>-backrest-shared-repo`. Note that the path name here is different than migrating from a PVC-based repository.

Using the `hippo` Postgres cluster as an example, you would set the following in the `spec.backups.pgbackrest.global` section:

```
spec:
  backups:
    pgbackrest:
      global:
        repo1-path: /backrestrepo/hippo-backrest-shared-repo
```

2\. A `spec.standby` cluster configuration within the spec that is populated according to the name of pgBackRest repo configured in the spec. For example:

```
spec:
  standby:
    enabled: true
    repoName: repo1
```

3\. If you are using the default setup in your PGO v4 cluster, you will need to provide custom setup parameters to include the pgAudit extension. This looks similar to the following: v4):

```
spec:
	patroni:
    dynamicConfiguration:
      postgresql:
        parameters:
          shared_preload_libraries: pgaudit.so
```

If you customized other Postgres parameters, you will need to ensure they match in the PGO v5 cluster. For more information, please review the tutorial on [customizing a Postgres cluster]({{< relref "tutorial/customize-cluster.md" >}}).

4\. Once the `PostgresCluster` spec is populated according to these guidelines, you can create the `PostgresCluster` custom resource.  For example, if the `PostgresCluster` you're creating is a modified version of the [`postgres` example](https://github.com/CrunchyData/postgres-operator-examples/tree/main/kustomize/postgres) in the [PGO examples repo](https://github.com/CrunchyData/postgres-operator-examples), you can run the following command:

```
kubectl apply -k examples/postgrescluster
```

5\. Once the standby cluster is up and running and you are satisfied with your set up, you can promote it.

First, you will need to shut down your PGO v4 cluster. You can do so with the following command, e.g.:

```
pgo update cluster hippo --shutdown
```

You can then update your PGO v5 cluster spec to promote your standby cluster:

```
spec:
  standby:
    enabled: false
```

Your upgrade is now complete! For more information on how to use PGO v5, we recommend reading through the [PGO v5 tutorial]({{< relref "tutorial/_index.md" >}}).

## Additional Considerations

Upgrading to PGO v5 may result in a base image upgrade from EL-7 (UBI / CentOS) to EL-8
(UBI / CentOS). Based on the contents of your Postgres database, you may need to perform
additional steps.

Due to changes in the GNU C library (`glibc`) in EL-8, you may need to reindex certain indexes in
your Postgres cluster. For more information, please read the
[PostgreSQL Wiki on Locale Data Changes](https://wiki.postgresql.org/wiki/Locale_data_changes), how
you can determine if your indexes are affected, and how to fix them.
