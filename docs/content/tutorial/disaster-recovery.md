---
title: "Disaster Recovery and Cloning"
date:
draft: false
weight: 85
---

Perhaps someone accidentally dropped the `users` table. Perhaps you want to clone your production database to a step-down environment. Perhaps you want to exercise your disaster recovery system (and it is important that you do!).

Regardless of scenario, it's important to know how you can perform a "restore" operation with PGO to be able to recovery your data from a particular point in time, or clone a database for other purposes.

Let's look at how we can perform different types of restore operations. First, let's understand the core restore properties on the custom resource.

## Restore Properties

{{% notice info %}}

As of v5.0.5, PGO offers the ability to restore from an existing PostgresCluster or a remote
cloud-based data source, such as S3, GCS, etc. For more on that, see the [Clone From Backups Stored in S3 / GCS / Azure Blob Storage](#cloud-based-data-source) section.

Note that you cannot use both a local PostgresCluster data source and a remote cloud-based data
source at one time; if both the `dataSource.postgresCluster` and `dataSource.pgbackrest` fields
are filled in, the local PostgresCluster data source will take precedence.

{{% /notice %}}

There are several attributes on the custom resource that are important to understand as part of the restore process. All of these attributes are grouped together in the `spec.dataSource.postgresCluster` section of the custom resource.

Please review the table below to understand how each of these attributes work in the context of setting up a restore operation.

- `spec.dataSource.postgresCluster.clusterName`: The name of the cluster that you are restoring from. This corresponds to the `metadata.name` attribute on a different `postgrescluster` custom resource.
- `spec.dataSource.postgresCluster.clusterNamespace`: The namespace of the cluster that you are restoring from. Used when the cluster exists in a different namespace.
- `spec.dataSource.postgresCluster.repoName`: The name of the pgBackRest repository from the `spec.dataSource.postgresCluster.clusterName` to use for the restore. Can be one of `repo1`, `repo2`, `repo3`, or `repo4`. The repository must exist in the other cluster.
- `spec.dataSource.postgresCluster.options`: Any additional [pgBackRest restore options](https://pgbackrest.org/command.html#command-restore) or general options that PGO allows. For example, you may want to set `--process-max` to help improve performance on larger databases; but you will not be able to set`--target-action`, since that option is currently disallowed. (PGO always sets it to `promote` if a `--target` is present, and otherwise leaves it blank.)
- `spec.dataSource.postgresCluster.resources`: Setting [resource limits and requests](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#requests-and-limits) of the restore job can ensure that it runs efficiently.
- `spec.dataSource.postgresCluster.affinity`: Custom [Kubernetes affinity](https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/) rules constrain the restore job so that it only runs on certain nodes.
- `spec.dataSource.postgresCluster.tolerations`: Custom [Kubernetes tolerations](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) allow the restore job to run on [tainted](https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/) nodes.

Let's walk through some examples for how we can clone and restore our databases.

## Clone a Postgres Cluster

Let's create a clone of our [`hippo`]({{< relref "./create-cluster.md" >}}) cluster that we created previously. We know that our cluster is named `hippo` (based on its `metadata.name`) and that we only have a single backup repository called `repo1`.

Let's call our new cluster `elephant`. We can create a clone of the `hippo` cluster using a manifest like this:

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: elephant
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
```

Note this section of the spec:

```
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
```

This is the part that tells PGO to create the `elephant` cluster as an independent copy of the `hippo` cluster.

The above is all you need to do to clone a Postgres cluster! PGO will work on creating a copy of your data on a new persistent volume claim (PVC) and work on initializing your cluster to spec. Easy!

## Perform a Point-in-time-Recovery (PITR)

Did someone drop the user table? You may want to perform a point-in-time-recovery (PITR)
to revert your database back to a state before a change occurred. Fortunately, PGO can help you do that.

You can set up a PITR using the [restore](https://pgbackrest.org/command.html#command-restore)
command of [pgBackRest](https://www.pgbackrest.org), the backup management tool that powers
the disaster recovery capabilities of PGO. You will need to set a few options on
`spec.dataSource.postgresCluster.options` to perform a PITR. These options include:

- `--type=time`: This tells pgBackRest to perform a PITR.
- `--target`: Where to perform the PITR to. An example recovery target is `2021-06-09 14:15:11-04`.
  The timezone specified here as -04 for EDT. Please see the [pgBackRest documentation for other timezone options](https://pgbackrest.org/user-guide.html#pitr).
- `--set` (optional): Choose which backup to start the PITR from.

A few quick notes before we begin:

- To perform a PITR, you must have a backup that finished before your PITR time.
  In other words, you can't perform a PITR back to a time where you do not have a backup!
- All relevant WAL files must be successfully pushed for the restore to complete correctly.
- Be sure to select the correct repository name containing the desired backup!

With that in mind, let's use the `elephant` example above. Let's say we want to perform a point-in-time-recovery (PITR) to `2021-06-09 14:15:11-04`, we can use the following manifest:

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: elephant
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
      options:
      - --type=time
      - --target="2021-06-09 14:15:11-04"
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
```

The section to pay attention to is this:

```
spec:
  dataSource:
    postgresCluster:
      clusterName: hippo
      repoName: repo1
      options:
      - --type=time
      - --target="2021-06-09 14:15:11-04"
```

Notice how we put in the options to specify where to make the PITR.

Using the above manifest, PGO will go ahead and create a new Postgres cluster that recovers
its data up until `2021-06-09 14:15:11-04`. At that point, the cluster is promoted and
you can start accessing your database from that specific point in time!

## Perform an In-Place Point-in-time-Recovery (PITR)

Similar to the PITR restore described above, you may want to perform a similar reversion
back to a state before a change occurred, but without creating another PostgreSQL cluster.
Fortunately, PGO can help you do this as well.

You can set up a PITR using the [restore](https://pgbackrest.org/command.html#command-restore)
command of [pgBackRest](https://www.pgbackrest.org), the backup management tool that powers
the disaster recovery capabilities of PGO. You will need to set a few options on
`spec.backups.pgbackrest.restore.options` to perform a PITR. These options include:

- `--type=time`: This tells pgBackRest to perform a PITR.
- `--target`: Where to perform the PITR to. An example recovery target is `2021-06-09 14:15:11-04`.
- `--set` (optional): Choose which backup to start the PITR from.

A few quick notes before we begin:

- To perform a PITR, you must have a backup that finished before your PITR time.
  In other words, you can't perform a PITR back to a time where you do not have a backup!
- All relevant WAL files must be successfully pushed for the restore to complete correctly.
- Be sure to select the correct repository name containing the desired backup!

To perform an in-place restore, users will first fill out the restore section of the spec as follows:

```
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
        repoName: repo1
        options:
        - --type=time
        - --target="2021-06-09 14:15:11-04"
```

And to trigger the restore, you will then annotate the PostgresCluster as follows:

```
kubectl annotate -n postgres-operator postgrescluster hippo --overwrite \
  postgres-operator.crunchydata.com/pgbackrest-restore=id1
```

And once the restore is complete, in-place restores can be disabled:

```
spec:
  backups:
    pgbackrest:
      restore:
        enabled: false
```

Notice how we put in the options to specify where to make the PITR.

Using the above manifest, PGO will go ahead and re-create your Postgres cluster to recover
its data up until `2021-06-09 14:15:11-04`. At that point, the cluster is promoted and
you can start accessing your database from that specific point in time!

## Restore Individual Databases

You can restore individual databases using a spec similar to the following:

```yaml
spec:
  backups:
    pgbackrest:
      restore:
        enabled: true
        repoName: repo1
        options:
        - --db-include=hippo
```

where `--db-include=hippo` would restore only the contents of the `hippo` database.

Please review the pgBackRest documentation on the [limitations on restoring individual databases](https://pgbackrest.org/user-guide.html#restore/option-db-include).

## Standby Cluster

Advanced high-availability and disaster recovery strategies involve spreading your database clusters
across data centers to help maximize uptime. PGO provides ways to deploy postgresclusters that can
span multiple Kubernetes clusters using an external storage system or PostgreSQL streaming replication.
The [disaster recovery architecture]({{< relref "architecture/disaster-recovery.md" >}}) documentation
provides a high-level overview of standby clusters with PGO can be found in the [disaster recovery
architecture] documentation.

### Creating a standby Cluster

This tutorial section will describe how to create three different types of standby clusters, one
using an external storage system, one that is streaming data directly from the primary, and one that
takes advantage of both external storage and streaming. These example clusters can be created in the
same Kubernetes cluster, using a single PGO instance, or spread across different Kubernetes clusters
and PGO instances with the correct storage and networking configurations.

#### Repo-based Standby

A repo-based standby will recover from WAL files a pgBackRest repo stored in external storage. The
primary cluster should be created with a cloud-based [backup configuration]({{< relref "tutorial/backups.md" >}}).
The following manifest defines a Postgrescluster with `standby.enabled` set to true and `repoName`
configured to point to the `s3` repo configured in the primary:

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo-standby
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec: { accessModes: [ReadWriteOnce], resources: { requests: { storage: 1Gi } } }
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        s3:
          bucket: "my-bucket"
          endpoint: "s3.ca-central-1.amazonaws.com"
          region: "ca-central-1"
  standby:
    enabled: true
    repoName: repo1
```

#### Streaming Standby

A streaming standby relies on an authenticated connection to the primary over the network. The primary
cluster should be accessible via the network and allow TLS authentication (TLS is enabled by default).
In the following manifest, we have `standby.enabled` set to `true` and have provided both the `host`
and `port` that point to the primary cluster. We have also defined `customTLSSecret` and
`customReplicationTLSSecret` to provide certs that allow the standby to authenticate to the primary.
For this type of standby, you must use [custom TLS]({{< relref "tutorial/customize-cluster.md" >}}#customize-tls):

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo-standby
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec: { accessModes: [ReadWriteOnce], resources: { requests: { storage: 1Gi } } }
  backups:
    pgbackrest:
      repos:
      - name: repo1
        volume:
          volumeClaimSpec: { accessModes: [ReadWriteOnce], resources: { requests: { storage: 1Gi } } }
  customTLSSecret:
    name: cluster-cert
  customReplicationTLSSecret:
    name: replication-cert
  standby:
    enabled: true
    host: "<primary-ip>"
    port: "<primary-port>"
```

#### Streaming Standby with an External Repo

Another option is to create a standby cluster using an external pgBackRest repo that streams from the
primary. With this setup, the standby cluster will continue recovering from the pgBackRest repo if
streaming replication falls behind. In this manifest, we have enabled the settings from both previous
examples:

```
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo-standby
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec: { accessModes: [ReadWriteOnce], resources: { requests: { storage: 1Gi } } }
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        s3:
          bucket: "my-bucket"
          endpoint: "s3.ca-central-1.amazonaws.com"
          region: "ca-central-1"
  customTLSSecret:
    name: cluster-cert
  customReplicationTLSSecret:
    name: replication-cert
  standby:
    enabled: true
    repoName: repo1
    host: "<primary-ip>"
    port: "<primary-port>"
```

## Promoting a Standby Cluster

At some point, you will want to promote the standby to start accepting both reads and writes.
This has the net effect of pushing WAL (transaction archives) to the pgBackRest repository, so we
need to ensure we don't accidentally create a split-brain scenario. Split-brain can happen if two
primary instances attempt to write to the same repository. If the primary cluster is still active,
make sure you [shutdown]({{< relref "tutorial/administrative-tasks.md" >}}#shutdown) the primary
before trying to promote the standby cluster.

Once the primary is inactive, we can promote the standby cluster by removing or disabling its
`spec.standby` section:

```
spec:
  standby:
    enabled: false
```

This change triggers the promotion of the standby leader to a primary PostgreSQL
instance and the cluster begins accepting writes.

## Clone From Backups Stored in S3 / GCS / Azure Blob Storage {#cloud-based-data-source}

You can clone a Postgres cluster from backups that are stored in AWS S3 (or a storage system
that uses the S3 protocol), GCS, or Azure Blob Storage without needing an active Postgres cluster!
The method to do so is similar to how you clone from an existing PostgresCluster. This is useful
if you want to have a data set for people to use but keep it compressed on cheaper storage.

For the purposes of this example, let's say that you created a Postgres cluster named `hippo` that
has its backups stored in S3 that looks similar to this:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      configuration:
      - secret:
          name: pgo-s3-creds
      global:
        repo1-path: /pgbackrest/postgres-operator/hippo/repo1
      manual:
        repoName: repo1
        options:
         - --type=full
      repos:
      - name: repo1
        s3:
          bucket: "my-bucket"
          endpoint: "s3.ca-central-1.amazonaws.com"
          region: "ca-central-1"
```

Ensure that the credentials in `pgo-s3-creds` match your S3 credentials. For more details on
[deploying a Postgres cluster using S3 for backups]({{< relref "./backups.md" >}}#using-s3),
please see the [Backups]({{< relref "./backups.md" >}}#using-s3) section of the tutorial.

For optimal performance when creating a new cluster from an active cluster, ensure that you take a
recent full backup of the previous cluster. The above manifest is set up to take a full backup.
Assuming `hippo` is created in the `postgres-operator` namespace, you can trigger a full backup
with the following command:

```shell
kubectl annotate -n postgres-operator postgrescluster hippo --overwrite \
  postgres-operator.crunchydata.com/pgbackrest-backup="$( date '+%F_%H:%M:%S' )"
```

Wait for the backup to complete. Once this is done, you can delete the Postgres cluster.

Now, let's clone the data from the `hippo` backup into a new cluster called `elephant`. You can use a manifest similar to this:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: elephant
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  dataSource:
    pgbackrest:
      stanza: db
      configuration:
      - secret:
          name: pgo-s3-creds
      global:
        repo1-path: /pgbackrest/postgres-operator/hippo/repo1
      repo:
        name: repo1
        s3:
          bucket: "my-bucket"
          endpoint: "s3.ca-central-1.amazonaws.com"
          region: "ca-central-1"
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      configuration:
      - secret:
          name: pgo-s3-creds
      global:
        repo1-path: /pgbackrest/postgres-operator/elephant/repo1
      repos:
      - name: repo1
        s3:
          bucket: "my-bucket"
          endpoint: "s3.ca-central-1.amazonaws.com"
          region: "ca-central-1"
```

There are a few things to note in this manifest. First, note that the `spec.dataSource.pgbackrest`
object in our new PostgresCluster is very similar but slightly different from the old
PostgresCluster's `spec.backups.pgbackrest` object. The key differences are:

1. No image is necessary when restoring from a cloud-based data source
2. `stanza` is a required field when restoring from a cloud-based data source
3. `backups.pgbackrest` has a `repos` field, which is an array
4. `dataSource.pgbackrest` has a `repo` field, which is a single object

Note also the similarities:

1. We are reusing the secret for both (because the new restore pod needs to have the same credentials as the original backup pod)
2. The `repo` object is the same
3. The `global` object is the same

This is because the new restore pod for the `elephant` PostgresCluster will need to reuse the
configuration and credentials that were originally used in setting up the `hippo` PostgresCluster.

In this example, we are creating a new cluster which is also backing up to the same S3 bucket;
only the `spec.backups.pgbackrest.global` field has changed to point to a different path. This
will ensure that the new `elephant` cluster will be pre-populated with the data from `hippo`'s
backups, but will backup to its own folders, ensuring that the original backup repository is
appropriately preserved.

Deploy this manifest to create the `elephant` Postgres cluster. Observe that it comes up and running:

```
kubectl -n postgres-operator describe postgrescluster elephant
```

When it is ready, you will see that the number of expected instances matches the number of ready
instances, e.g.:

```
Instances:
  Name:               00
  Ready Replicas:     1
  Replicas:           1
  Updated Replicas:   1
```

The previous example shows how to use an existing S3 repository to pre-populate a PostgresCluster
while using a new S3 repository for backing up. But PostgresClusters that use cloud-based data
sources can also use local repositories.

For example, assuming a PostgresCluster called `rhino` that was meant to pre-populate from the
original `hippo` PostgresCluster, the manifest would look like this:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: rhino
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  dataSource:
    pgbackrest:
      stanza: db
      configuration:
      - secret:
          name: pgo-s3-creds
      global:
        repo1-path: /pgbackrest/postgres-operator/hippo/repo1
      repo:
        name: repo1
        s3:
          bucket: "my-bucket"
          endpoint: "s3.ca-central-1.amazonaws.com"
          region: "ca-central-1"
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi

```

## Next Steps

Now we've seen how to clone a cluster and perform a point-in-time-recovery, let's see how we can [monitor]({{< relref "./monitoring.md" >}}) our Postgres cluster to detect and prevent issues from occurring.
