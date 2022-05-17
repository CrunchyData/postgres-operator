---
title: "Upgrade Method #2: Backups"
date:
draft: false
weight: 20
---

{{% notice info %}}
Before attempting to upgrade from v4.x to v5, please familiarize yourself with the [prerequisites]({{< relref "upgrade/v4tov5/_index.md" >}}) applicable for all v4.x to v5 upgrade methods.
{{% /notice %}}

This upgrade method allows you to migrate from PGO v4 to PGO v5 by creating a new PGO v5 Postgres cluster using a backup from a PGO v4 cluster. This method allows you to preserve the data in your PGO v4 cluster while you transition to PGO v5. To fully move the data over, you will need to incur downtime and shut down your PGO v4 cluster.

### Step 1: Prepare the PGO v4 Cluster for Migration

1\. Ensure you have a recent backup of your cluster. You can do so with the `pgo backup` command, e.g.:

```
pgo backup hippo
```

Please ensure that the backup completes. You will see the latest backup appear using the `pgo show backup` command.

2\. Next, delete the cluster while keeping backups (using the `--keep-backups` flag):

```
pgo delete cluster hippo --keep-backups
```

{{% notice warning %}}

Additional steps are required to set proper file permissions when using certain storage options,
such as NFS and HostPath storage, due to a known issue with how fsGroups are applied. When
migrating from PGO v4, this will require the user to manually set the group value of the pgBackRest
repo directory, and all subdirectories, to `26` to match the `postgres` group used in PGO v5.
Please see [here](https://github.com/kubernetes/examples/issues/260) for more information.

{{% /notice %}}

### Step 2: Migrate to PGO v5

With the PGO v4 Postgres cluster's backup repository prepared, you can now create a [`PostgresCluster`]({{< relref "references/crd.md" >}}) custom resource. This migration method does not carry over any specific configurations or customizations from PGO v4: you will need to create the specific `PostgresCluster` configuration that you need.

To complete the upgrade process, your `PostgresCluster` custom resource **MUST** include the following:

1\. You will need to configure your pgBackRest repository based upon whether you are using a PVC to store your backups, or an object storage system such as S3/GCS. Please follow the directions based upon the repository type you are using as part of the migration.

#### PVC-based Backup Repository

When migrating from a PVC-based backup repository, you will need to configure a pgBackRest repo of a `spec.backups.pgbackrest.repos.volume` under the `spec.backups.pgbackrest.repos.name` of `repo1`. The `volumeClaimSpec` should match the attributes of the pgBackRest repo PVC being used as part of the migration, i.e. it must have the same `storageClassName`, `accessModes`, `resources`, etc.  Please note that you will need to perform the cluster upgrade in the same namespace as the original cluster in order for your v5 cluster to access the existing PVCs. For example:

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

3\. If you are using a PVC-based pgBackRest repository, then you will also need to specify a pgBackRestVolume data source that references the PGO v4 pgBackRest repository PVC:

```
spec:
  dataSource:
    volumes:
      pgBackRestVolume:
        pvcName: hippo-pgbr-repo
        directory: "hippo-backrest-shared-repo"
    postgresCluster:
      repoName: repo1
```


4\. If you customized other Postgres parameters, you will need to ensure they match in the PGO v5 cluster. For more information, please review the tutorial on [customizing a Postgres cluster]({{< relref "tutorial/customize-cluster.md" >}}).

5\. Once the `PostgresCluster` spec is populated according to these guidelines, you can create the `PostgresCluster` custom resource.  For example, if the `PostgresCluster` you're creating is a modified version of the [`postgres` example](https://github.com/CrunchyData/postgres-operator-examples/tree/main/kustomize/postgres) in the [PGO examples repo](https://github.com/CrunchyData/postgres-operator-examples), you can run the following command:

```
kubectl apply -k examples/postgrescluster
```

**WARNING**: Once the PostgresCluster custom resource is created, it will become the owner of the PVC.  *This means that if the PostgresCluster is then deleted (e.g. if attempting to revert back to a PGO v4 cluster), then the PVC will be deleted as well.*

If you wish to protect against this, first remove the reference to the pgBackRest PVC in the PostgresCluster spec:

```
kubectl patch postgrescluster hippo-pgbr-repo --type='json' -p='[{"op": "remove", "path": "/spec/dataSource/volumes"}]'
```

Then relabel the PVC prior to deleting the PostgresCluster custom resource. Below uses the `hippo` Postgres cluster as an example:

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
