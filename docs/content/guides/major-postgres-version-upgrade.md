---
title: "Postgres Major Version Upgrade"
date:
draft: false
weight: 100
---

You can perform a PostgreSQL major version upgrade declaratively using PGO! The below guide will show you how you can upgrade Postgres to a newer major version. For minor updates, i.e. applying a bug fix release, you can follow the [applying software updates]({{< relref "/tutorial/update-cluster.md" >}}) guide in the [tutorial]({{< relref "/tutorial/_index.md" >}}).

Note that major version upgrades are **permanent**: you cannot roll back a major version upgrade through declarative management at this time. If this is an issue, we recommend keeping a copy of your Postgres cluster running your previous version of Postgres.

{{% notice warning %}}
**Please note the following prior to performing a PostgreSQL major version upgrade:**
- Any Postgres cluster being upgraded must be in a healthy state in order for the upgrade to
complete successfully.  If the cluster is experiencing issues such as Pods that are not running
properly, or any other similar problems, those issues must be addressed before proceeding.
- Major PostgreSQL version upgrades of PostGIS clusters are not currently supported.
{{% /notice %}}

## Step 1: Take a Full Backup

Before starting your major upgrade, you should take a new full [backup]({{< relref "tutorial/backup-management.md" >}}) of your data. This adds another layer of protection in cases where the upgrade process does not complete as expected.

At this point, your running cluster is ready for the major upgrade.

## Step 2: Configure the Upgrade Parameters through a PGUpgrade object

The next step is to create a `PGUpgrade` resource. This is the resource that tells the PGO-Upgrade controller which cluster to upgrade, what version to upgrade from, and what version to upgrade to. There are other optional fields to fill in as well, such as `Resources` and `Tolerations`; to learn more about these optional fields, check out the [Upgrade CRD API]({{< relref "references/crd.md" >}}).

For instance, if you have a Postgres cluster named `hippo` running PG {{< param fromPostgresVersion >}} but want to upgrade it to PG {{< param postgresVersion >}}, the corresponding `PGUpgrade` manifest would look like this:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PGUpgrade
metadata:
  name: hippo-upgrade
spec:
  image: {{< param imageCrunchyPGUpgrade >}}
  postgresClusterName: hippo
  fromPostgresVersion: {{< param fromPostgresVersion >}}
  toPostgresVersion: {{< param postgresVersion >}}
```

The `postgresClusterName` gives the name of the target Postgres cluster to upgrade and `toPostgresVersion` gives the version to update to. It may seem unnecessary to include the `fromPostgresVersion`, but that is one of the safety checks we have built into the upgrade process: in order to successfully upgrade a Postgres cluster, you have to know what version you mean to be upgrading from.

One very important thing to note: upgrade objects should be made in the same namespace as the Postgres cluster that you mean to upgrade. For security, the PGO-Upgrade controller does not allow for cross-namespace processes.

If you look at the status of the `PGUpgrade` object at this point, you should see a condition saying this:

```
type: "progressing",
status: "false",
reason: "PGClusterNotShutdown",
message: "PostgresCluster instances still running",
```

What that means is that the upgrade process is blocked because the cluster is not yet shutdown. We are stuck ("progressing" is false) until we shutdown the cluster. So let's go ahead and do that now.

## Step 3: Shutdown and Annotate the Cluster

In order to kick off the upgrade process, you need to shutdown the cluster and add an annotation to the cluster signalling which PGUpgrade to run.

Why do we need to add an annotation to the cluster if the PGUpgrade already has the cluster's name? This is another security mechanism--think of it as a two-key nuclear system: the `PGUpgrade` has to know which Postgres cluster to upgrade; and the Postgres cluster has to allow this upgrade to work on it.

The annotation to add is `postgres-operator.crunchydata.com/allow-upgrade`, with the name of the `PGUpgrade` object as the value. So for our example above with a Postgres cluster named `hippo` and a `PGUpgrade` object named `hippo-upgrade`, we could annotate the cluster with the command

```bash
kubectl -n postgres-operator annotate postgrescluster hippo postgres-operator.crunchydata.com/allow-upgrade="hippo-upgrade"
```

To shutdown the cluster, edit the `spec.shutdown` field to true and reapply the spec with `kubectl`. For example, if you used the [tutorial]({{< relref "tutorial/_index.md" >}}) to [create your Postgres cluster]({{< relref "tutorial/create-cluster.md" >}}), you would run the following command:

```
kubectl -n postgres-operator apply -k kustomize/postgres
```

(Note: you could also change the annotation at the same time as you shutdown the cluster; the purpose of demonstrating how to annotate was primarily to show what the label would look like.)

## Step 4: Watch and wait

When the last Postgres Pod is terminated, the PGO-Upgrade process will kick into action, upgrading the primary database and preparing the replicas. If you are watching the namespace, you will see the PGUpgrade controller start Pods for each of those actions. But you don't have to watch the namespace to keep track of the upgrade process.

To keep track of the process and see when it finishes, you can look at the `status.conditions` field of the `PGUpgrade` object. If the upgrade process encounters any blockers preventing it from finishing, the `status.conditions` field will report on those blockers. When it finishes upgrading the cluster, it will show the status conditions:

```
type:   "Progressing"
status: "false"
reason: "PGUpgradeCompleted"

type:   "Succeeded"
status: "true"
reason: "PGUpgradeSucceeded"
```

You can also check the Postgres cluster itself to see when the upgrade has completed. When the upgrade is complete, the cluster will show the new version in its `status.postgresVersion` field.

If the process encounters any errors, the upgrade process will stop to prevent further data loss; and the `PGUpgrade` object will report the failure in its status. For more specifics about the failure, you can check the logs of the individual Pods that were doing the upgrade jobs.

## Step 5: Restart your Postgres cluster with the new version

Once the upgrade process is complete, you can erase the `PGUpgrade` object, which will clean up any Jobs and Pods that were created during the upgrade. But as long as the process completed successfully, that `PGUpgrade` object will remain inert. If you find yourself needing to upgrade the cluster again, you will not be able to edit the existing `PGUpgrade` object with the new versions, but will have to create a new `PGUpgrade` object. Again, this is a safety mechanism to make sure that any PGUpgrade can only be run once.

Likewise, you may remove the annotation on the Postgres cluster as part of the cleanup. While not necessary, it is recommended to leave your cluster without unnecessary annotations.

To restart your newly upgraded Postgres cluster, you will have to update the `spec.postgresVersion` to the new version. You may also have to update the `spec.image` value to reflect the image you plan to use if that field is already filled in. Turn `spec.shutdown` to false, and PGO will restart your cluster:

```
spec:
  shutdown: false
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
```

{{% notice warning %}}
Setting and applying the `postgresVersion` or `image` values before the upgrade will result in the upgrade process being rejected.
{{% /notice %}}

## Step 6: Complete the Post-Upgrade Tasks

After the upgrade Job has completed, there will be some amount of post-upgrade processing that
needs to be done. During the upgrade process, the upgrade Job, via [`pg_upgrade`](https://www.postgresql.org/docs/current/pgupgrade.html), will issue warnings and possibly create scripts to perform post-upgrade tasks. You can see the full output of the upgrade Job by running a command similar to this:

```
kubectl -n postgres-operator logs hippo-pgupgrade-abcd
```

While the scripts are placed on the Postgres data PVC, you may not have access to them. The below information describes what each script does and how you can execute them.

In Postgres 13 and older, `pg_upgrade` creates a script called `analyze_new_cluster.sh` to perform a post-upgrade analyze using [`vacuumdb`](https://www.postgresql.org/docs/current/app-vacuumdb.html) on the database.

The script provides two ways of doing so:

```
vacuumdb --all --analyze-in-stages
```

or

```
vacuumdb --all --analyze-only
```

Note that these commands need to be run as a Postgres superuser (e.g. `postgres`). For more information on the difference between the options, please see the documentation for [`vacuumdb`](https://www.postgresql.org/docs/current/app-vacuumdb.html).

If you are unable to exec into the Pod, you can run `ANALYZE` directly on each of your databases.

`pg_upgrade` may also create a script called `delete_old_cluster.sh`, which contains the equivalent of

```
rm -rf '/pgdata/pg{{< param fromPostgresVersion >}}'
```

When you are satisfied with the upgrade, you can execute this command to remove the old data directory. Do so at your discretion.

Note that the `delete_old_cluster.sh` script does not delete the old WAL files. These are typically found in `/pgdata/pg{{< param fromPostgresVersion >}}_wal`, although they can be stored elsewhere. If you would like to delete these files, this must be done manually.

If you have extensions installed you may need to upgrade those as well. For example, for the `pgaudit` extension we recommend running the following to upgrade:

```sql
DROP EXTENSION pgaudit;
CREATE EXTENSION pgaudit;
```

`pg_upgrade` may also create a file called `update_extensions.sql` to facilitate extension upgrades. Be aware some of the recommended ways to upgrade may be outdated.

Please carefully review the `update_extensions.sql` file before you run it, and if you want to upgrade `pgaudit` via this file, update the file with the above commands for `pgaudit` prior to execution. We recommend verifying all extension updates from this file with the appropriate extension documentation and their recommendation for upgrading the extension prior to execution. After you update the file, you can execute this script using `kubectl exec`, e.g.

```
$ kubectl -n postgres-operator exec -it -c database \
  $(kubectl -n postgres-operator get pods --selector='postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master' -o name) -- psql -f /pgdata/update_extensions.sql
```

If you cannot exec into your Pod, you can also manually run these commands as a Postgres superuser.

Ensure the execution of this and any other SQL scripts completes successfully, otherwise your data may be unavailable.

Once this is done, your major upgrade is complete! Enjoy using your newer version of Postgres!
