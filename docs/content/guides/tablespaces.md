---
title: "Tablespaces in PGO"
date:
draft: false
weight: 160
---

{{% notice warning %}}
PGO tablespaces currently requires enabling the `TablespaceVolumes` feature gate
and may interfere with other features. (See below for more details.)
{{% /notice %}}

A [Tablespace](https://www.postgresql.org/docs/current/manage-ag-tablespaces.html)
is a Postgres feature that is used to store data on a different volume than the
primary data directory. While most workloads do not require tablespaces, they can
be helpful for larger data sets or utilizing particular hardware to optimize
performance on a particular Postgres object (a table, index, etc.). Some examples
of use cases for tablespaces include:

- Partitioning larger data sets across different volumes
- Putting data onto archival systems
- Utilizing faster/more performant hardware (or a storage class) for a particular database
- Storing sensitive data on a volume that supports transparent data-encryption (TDE)

and others.

In order to use Postgres tablespaces properly in a highly-available,
distributed system, there are several considerations to ensure proper operations:

- Each tablespace must have its own volume; this means that every tablespace for
every replica in a system must have its own volume;
- The available filesystem paths must be consistent on each Postgres pod in a Postgres cluster;
- The backup & disaster recovery management system must be able to safely backup
and restore data to tablespaces.

Additionally, a tablespace is a critical piece of a Postgres instance: if
Postgres expects a tablespace to exist and the tablespace volume is unavailable,
this could trigger a downtime scenario.

While there are certain challenges with creating a Postgres cluster with
high-availability along with tablespaces in a Kubernetes-based environment, the
Postgres Operator adds many conveniences to make it easier to use tablespaces.

## Enabling TablespaceVolumes in PGO v5

In PGO v5, tablespace support is currently feature-gated. If you want to use this
experimental feature, you will need to enable the feature via the PGO `TablespaceVolumes`
[feature gate](https://kubernetes.io/docs/reference/command-line-tools-reference/feature-gates/).

PGO feature gates are enabled by setting the `PGO_FEATURE_GATES` environment
variable on the PGO Deployment. To enable tablespaces, you would want to set

```
PGO_FEATURE_GATES="TablespaceVolumes=true"
```

Please note that it is possible to enable more than one feature at a time as
this variable accepts a comma delimited list. For example, to enable multiple features,
you would set `PGO_FEATURE_GATES` like so:

```
PGO_FEATURE_GATES="FeatureName=true,FeatureName2=true,FeatureName3=true..."
```

## Adding TablespaceVolumes to a postgrescluster in PGO v5

Once you have enabled `TablespaceVolumes` on your PGO deployment, you can add volumes to
a new or existing cluster by adding volumes to the `spec.instances.tablespaceVolumes` field.

A `TablespaceVolume` object has two fields: a name (which is required and used to set the path)
and a `dataVolumeClaimSpec`, which describes the storage that your Postgres instance will use
for this volume. This field behaves identically to the `dataVolumeClaimSpec` in the `instances`
list. For example, you could use the following to create a `postgrescluster`:

```yaml
spec:
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
      tablespaceVolumes:
      - name: user
        dataVolumeClaimSpec:
          accessModes:
          - "ReadWriteOnce"
          resources:
            requests:
              storage: 1Gi
```

In this case, the `postgrescluster` will have 1Gi for the database volume and 1Gi for the tablespace
volume, and both will be provisioned by PGO.

But if you were attempting to migrate data from one `postgrescluster` to another, you could re-use
pre-existing volumes by passing in some label selector or the `volumeName` into the
`tablespaceVolumes.dataVolumeClaimSpec` the same way you would pass that information into the
`instances.dataVolumeClaimSpec` field:

```yaml
spec:
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        volumeName: pvc-1001c17d-c137-4f78-8505-be4b26136924 # A preexisting volume you want to reuse for PGDATA
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
      tablespaceVolumes:
      - name: user
        dataVolumeClaimSpec:
          accessModes:
          - "ReadWriteOnce"
          resources:
            requests:
              storage: 1Gi
          volumeName: pvc-3fea1531-617a-4fff-9032-6487206ce644 # A preexisting volume you want to use for this tablespace
```

Note: the `name` of the `tablespaceVolume` needs to be

* unique in the instance since that name becomes part of the mount path for that volume;
* valid as part of a path name, label, and part of a volume name.

There is validation on the CRD for these requirements.

Once you request those `tablespaceVolumes`, PGO takes care of creating (or reusing) those volumes,
including mounting them to the pod at a known path (`/tablespaces/NAME`) and adding them to the
necessary containers.

### How to use Postgres Tablespaces in PGO v5

After PGO has mounted the volumes at the requested locations, the startup container makes sure
that those locations have the appropriate owner and permissions. This behavior mimics the startup
behavior behind the `PGDATA` directory, so that when you connect to your cluster, you should be
able to start using those tablespaces.

In order to use those tablespaces in Postgres, you will first need to create the tablespace,
including the location. As noted above, PGO mounts the requested volumes at `/tablespaces/NAME`.
So if you request tablespaces with the names `books` and `authors`, the two volumes will be
mounted at `/tablespaces/books` and `/tablespaces/authors`.

However, in order to make sure that the directory has the appropriate ownership so that Postgres
can use it, we create a subdirectory called `data` in each volume.

To create a tablespace in Postgres, you will issue a command of the form

```
CREATE TABLESPACE name LOCATION '/path/to/dir';
```

So to create a tablespace called `books` in the new `books` volume, your command might look like

```
CREATE TABLESPACE books LOCATION '/tablespaces/books/data';
```

To break that path down: `tablespaces` is the mount point for all tablespace volumes; `books`
is the name of the volume in the spec; and `data` is a directory created with the appropriate
ownership by the startup script.

Once you have

* enabled the `TablespaceVolumes` feature gate,
* added `tablespaceVolumes` to your cluster spec,
* and created the tablespace in Postgres,

then you are ready to use tablespaces in your cluster. For example, if you wanted to create a
table called `books` on the `books` tablespace, you could execute the following SQL:

```sql
CREATE TABLE books (
   book_id VARCHAR2(20),
   title VARCHAR2(50)
   author_last_name VARCHAR2(30)
)
TABLESPACE books;
```

## Considerations

### Only one pod per volume

As stated above, it is important to ensure that every tablespace has its own volume
(i.e. its own [persistent volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)).
This is especially true for any replicas in a cluster: you don't want multiple Postgres instances
writing to the same volume.

So if you have a single named volume in your spec (for either the main PGDATA directory or
for tablespaces), you should not raise the `spec.instances.replicas` field above 1, because if you
did, multiple pods would try to use the same volume.

### Too-long names?

Different Kubernetes objects have different limits about the length of their names. For example,
services follow the DNS label conventions: 63 characters or less, lowercase, and alphanumeric with
hyphens U+002D allowed in between.

Occasionally some PGO-managed objects will go over the limit set for that object type because of
the user-set cluster or instance name.

We do not anticipate this being a problem with the `PersistentVolumeClaim` created for a tablespace.
The name for a `PersistentVolumeClaim` created by PGO for a tablespace will potentially be long since
the name is a combination of the cluster, the instance, the tablespace, and the `-tablespace` suffix.
However, a `PersistentVolumeClaim` name can be up to 253 characters in length.

### Same tablespace volume names across replicas

We want to make sure that every pod has a consistent filesystem because Postgres expects
the same path on each replica.

For instance, imagine on your primary Postgres, you add a tablespace with the location
`/tablespaces/kafka/data`. If you have a replica attached to that primary, it will likewise
try to add a tablespace at the location `/tablespaces/kafka/data`; and if that location doesn't
exist on the replica's filesystem, Postgres will rightly complain.

Therefore, if you expand your `postgrescluster` with multiple instances, you will need to make
sure that the multiple instances have `tablespaceVolumes` with the *same names*, like so:

```yaml
spec:
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
      tablespaceVolumes:
      - name: user
        dataVolumeClaimSpec:
          accessModes:
          - "ReadWriteOnce"
          resources:
            requests:
              storage: 1Gi
    - name: instance2
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
      tablespaceVolumes:
      - name: user
        dataVolumeClaimSpec:
          accessModes:
          - "ReadWriteOnce"
          resources:
            requests:
              storage: 1Gi
```

### Tablespace backups

PGO uses `pgBackRest` as our backup solution, and `pgBackRest` is built to work with tablespaces
natively. That is, `pgBackRest` should back up the entire database, including tablespaces, without
any additional work on your part.

**Note**: `pgBackRest` does not itself use tablespaces, so all the backups will go to a single volume.
One of the primary uses of tablespaces is to relieve disk pressure by separating the database among
multiple volumes, but if you are running out of room on your `pgBackRest` persistent volume,
tablespaces will not help, and you should first solve your backup space problem.

### Adding tablespaces to existing clusters

As with other changes made to the definition of a Postgres pod, adding `tablespaceVolumes` to an
existing cluster may cause downtime. The act of mounting a new PVC to a Kubernetes Deployment
causes the Pods in the deployment to restart.

### Restoring from a cluster with tablespaces

This functionality has not been fully tested. Enjoy!

### Removing tablespaces

Removing a tablespace is a nontrivial operation. Postgres does not provide a
`DROP TABLESPACE .. CASCADE` command that would drop any associated objects with a tablespace.
Additionally, the Postgres documentation covering the
[`DROP TABLESPACE`](https://www.postgresql.org/docs/current/sql-droptablespace.html)
command goes on to note:

> A tablespace can only be dropped by its owner or a superuser. The tablespace
> must be empty of all database objects before it can be dropped. It is possible
> that objects in other databases might still reside in the tablespace even if
> no objects in the current database are using the tablespace. Also, if the
> tablespace is listed in the temp_tablespaces setting of any active session,
> the DROP might fail due to temporary files residing in the tablespace.

Because of this, and to avoid a situation where a Postgres cluster is left in an inconsistent
state due to trying to remove a tablespace, PGO does not provide any means to remove tablespaces
automatically. If you need to remove a tablespace from a Postgres deployment, we recommend
following this procedure:

1. As a database administrator:
  1. Log into the primary instance of your cluster.
  1. Drop any objects (tables, indexes, etc) that reside within the tablespace you wish to delete.
  1. Delete this tablespace from the Postgres cluster using the `DROP TABLESPACE` command.
1. As a Kubernetes user who can modify `postgrescluster` specs
  1. Remove the `tablespaceVolumes` entries for the tablespaces you wish to remove.

## More Information

For more information on how tablespaces work in Postgres please refer to the
[Postgres manual](https://www.postgresql.org/docs/current/manage-ag-tablespaces.html).