---
title: "Storage Retention"
date:
draft: false
weight: 125
---

PGO uses [persistent volumes](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) to store Postgres data and, based on your configuration, data for backups, archives, etc. There are cases where you may want to retain your volumes for [later use]({{< relref "./data-migration.md" >}}).

The below guide shows how to configure your persistent volumes (PVs) to remain after a Postgres cluster managed by PGO is deleted and to deploy the retained PVs to a new Postgres cluster.

For the purposes of this exercise, we will use a Postgres cluster named `hippo`.

## Modify Persistent Volume Retention

Retention of persistent volumes is set using a [reclaim policy](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming). By default, more persistent volumes have a policy of `Delete`, which removes any data on a persistent volume once there are no more persistent volume claims (PVCs) associated with it.

To retain a persistent volume you will need to set the reclaim policy to `Retain`. Note that persistent volumes are cluster-wide objects, so you will need to appropriate permissions to be able to modify a persistent volume.

To retain the persistent volume associated with your Postgres database, you must first determine which persistent volume is associated with the persistent volume claim for your database. First, local the persistent volume claim. For example, with the `hippo` cluster, you can do so with the following command:

```
kubectl get pvc -n postgres-operator --selector=postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/data=postgres
```

This will yield something similar to the below, which are the PVCs associated with any Postgres instance:

```
NAME                          STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
hippo-instance1-x9vq-pgdata   Bound    pvc-aef7ee64-4495-4813-b896-8a67edc53e58   1Gi        RWO            standard       6m53s
```

The `VOLUME` column contains the name of the persistent volume. You can inspect it using `kubectl get pv`, e.g.:

```
kubectl get pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58
```

which should yield:

```
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                           STORAGECLASS   REASON   AGE
pvc-aef7ee64-4495-4813-b896-8a67edc53e58   1Gi        RWO            Delete           Bound    postgres-operator/hippo-instance1-x9vq-pgdata   standard                8m10s
```

To modify the reclaim policy set it to `Retain`, you can run a command similar to this:

```
kubectl patch pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58  -p '{"spec":{"persistentVolumeReclaimPolicy":"Retain"}}'
```

Verify that the change occurred:

```
kubectl get pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58
```

should show that `Retain` is set in the `RECLAIM POLICY` column:

```
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS   CLAIM                                           STORAGECLASS   REASON   AGE
pvc-aef7ee64-4495-4813-b896-8a67edc53e58   1Gi        RWO            Retain           Bound    postgres-operator/hippo-instance1-x9vq-pgdata   standard                9m53s
```

## Delete Postgres Cluster, Retain Volume

{{% notice warning %}}
**This is a potentially destructive action**. Please be sure that your volume retention is set correctly and/or you have backups in place to restore your data.
{{% / notice %}}

[Delete your Postgres cluster]({{< relref "tutorial/delete-cluster.md" >}}). You can delete it using the manifest or with a command similar to:

```
kubectl -n postgres-operator delete postgrescluster hippo
```

Wait for the Postgres cluster to finish deleting. You should then verify that the persistent volume is still there:

```
kubectl get pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58
```

should yield:

```
NAME                                       CAPACITY   ACCESS MODES   RECLAIM POLICY   STATUS     CLAIM                                           STORAGECLASS   REASON   AGE
pvc-aef7ee64-4495-4813-b896-8a67edc53e58   1Gi        RWO            Retain           Released   postgres-operator/hippo-instance1-x9vq-pgdata   standard                21m
```

## Create Postgres Cluster With Retained Volume

You can now create a new Postgres cluster with the retained volume. First, to aid the process, you will want to provide a label that is unique for your persistent volumes so we can identify it in the manifest. For example:

```
kubectl label pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58 pgo-postgres-cluster=postgres-operator-hippo
```

(This label uses the format `<namespace>-<clusterName>`).

Next, you will need to reference this persistent volume in your Postgres cluster manifest. For example:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
        selector:
          matchLabels:
            pgo-postgres-cluster: postgres-operator-hippo
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

Wait for the Pods to come up. You may see the Postgres Pod is in a `Pending` state. You will need to go in and clear the claim on the persistent volume that you want to use for this Postgres cluster, e.g.:

```
kubectl patch pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58  -p '{"spec":{"claimRef": null}}'
```

After that, your Postgres cluster will come up and will be using the previously used persistent volume!

If you ultimately want the volume to be deleted, you will need to revert the reclaim policy to `Delete`, e.g.:

```
kubectl patch pv pvc-aef7ee64-4495-4813-b896-8a67edc53e58  -p '{"spec":{"persistentVolumeReclaimPolicy":"Delete"}}'
```

After doing that, the next time you delete your Postgres cluster, the volume and your data will be deleted.

### Additional Notes on Storage Retention

Systems using "hostpath" storage or a storage class that does not support label selectors may not be able to use the label selector method for using a retained volume volume. You would have to specify the `volumeName` directly, e.g.:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
        volumeName: "pvc-aef7ee64-4495-4813-b896-8a67edc53e58"
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

Additionally, to add additional replicas to your Postgres cluster, you will have to make changes to your spec. You can do one of the following:

1. Remove the volume-specific configuration from the volume claim spec (e.g. delete `spec.instances.selector` or `spec.instances.volumeName`)

2. Add a new instance set specifically for your replicas, e.g.:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
      selector:
        matchLabels:
          pgo-postgres-cluster: postgres-operator-hippo
    - name: instance2
      dataVolumeClaimSpec:
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
