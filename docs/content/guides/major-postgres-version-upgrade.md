---
title: "Postgres Major Version Upgrade"
date:
draft: false
weight: 100
---

You can perform a PostgreSQL major version upgrade declratively using PGO! The below guide will show you how you can upgrade Postgres to a newer major version. For minor updates, i.e. applying a bug fix release, you can follow the [applying software updates]({{< relref "tutorial/update-cluster.md" >}}) guide in the [tutoral]({{< relref "tutorial/_index.md" >}}).

Note that major version upgrades are **permanent**, you cannot roll back a major version upgrade through declarative management at this time. If this is an issue, we recommend keeping a copy of your Postgres cluster running your previous version of Postgres.

{{% notice warning %}}
**Please note the following prior to performing a PostgreSQL major version upgrade:**
- Any Postgres cluster being upgraded must be in a healthy state in order for the upgrade to
complete successfully.  If the cluster is experiencing issues such as Pods that are not running
properly, or any other similar problems, those issues must be addressed before proceeding.
- Major PostgreSQL version upgrades of PostGIS clusters are not currently supported.
{{% /notice %}}

## Step 1: Take a Full Backup

Before starting your major upgrade, you should take a new full [backup]({{< relref "tutorial/backup-management.md" >}}) of your data. This adds another layer of protection in cases where the upgrade process does not complete as expected.

## Step 2: Configure the Upgrade Parameters

At this point, your running cluster is ready for the major upgrade. As shown below, you will need to
configure the `spec.upgrade` section to enable the major upgrade. This section will also define the current
(i.e. pre-upgrade) Postgres version of the cluster and define the `crunchy-upgrade` image, assuming the
relevant environment variable, `RELATED_IMAGE_PGUPGRADE`, is not being utilized.

At the same time, you will need to set the `spec.postgresVersion` and `spec.image` values to reflect the
version the PostgresCluster will be after the upgrade is completed. Below, you'll see an example configuration
for a {{< param fromPostgresVersion >}} to {{< param postgresVersion >}} upgrade:

```
spec:
  upgrade:
    enabled: true
    fromPostgresVersion: {{< param fromPostgresVersion >}}
    image: {{< param imageCrunchyPGUpgrade >}}
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
```

Please note, the `spec.upgrade` section can be set in advance of running the upgrade, just be sure
to set `enabled` to `false`.

{{% notice warning %}}
Setting and applying the `postgresVersion` or `image` values before configuring and enabling the
`spec.upgrade` section will result in an error due to the PostgreSQL version mismatch!
{{% /notice %}}

## Step 3: Perform the Upgrade

Once everything is configured, you can apply the changes. For example, if you used the [tutorial]({{< relref "tutorial/_index.md" >}}) to [create your Postgres cluster]({{< relref "tutorial/create-cluster.md" >}}), you would run the following command:

```
kubectl apply -k kustomize/postgres
```

PGO will terminate the Postgres instances and start the upgrade Job. This Job will perform the necessary steps to upgrade the Postgres cluster to the desired version.

Once the upgrade Job completes, PGO will start the primary instance.

## Step 4: Remove the Upgrade Parameters

Once the upgrade Job has completed and the primary instance is running, remove the upgrade configuration you set, i.e.:

```
spec:
  upgrade:
    enabled: true
    fromPostgresVersion: {{< param fromPostgresVersion >}}
    image: {{< param imageCrunchyPGUpgrade >}}
```

## Step 5: Complete the Post-Upgrade Tasks

After the upgrade Job has completed, there will be some amount of post-upgrade processing that
needs to be done. During the upgrade process, the upgrade Job, via [`pg_upgrade`](https://www.postgresql.org/docs/current/pgupgrade.html), will issue warnings and possibly create scripts to perform post-upgrade tasks. You can see the full output of the upgrade Job by running a command similar to this:

```
kubectl -n postgres-operator logs hippo-pgupgrade-abcd
```

While the scripts are provided placed on the Postgres data PVC, you may not have access to them. The below information describes what each script does and how you can execute them.

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
rm -rf '/pgdata/{{< param fromPostgresVersion >}}'
```

When you are satisfied with the upgrade, you can execute this command to remove the old data directory. Do so at your discretion.

`pg_upgrade` may also create a file called `update_extensions.sql` file created to facilitate any extension upgrades.

For example, if you are using the `pgaudit` extension, you may see this in the file:

```sql
\connect hippo
ALTER EXTENSION "pgaudit" UPDATE;
\connect postgres
ALTER EXTENSION "pgaudit" UPDATE;
\connect template1
ALTER EXTENSION "pgaudit" UPDATE;
```

You can execute this script using `kubectl exec`, e.g.

```
$ kubectl exec -it -n postgres-operator -c database \
  $(kubectl get pods -n postgres-operator --selector='postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master' -o name) -- psql -f /pgdata/update_extensions.sql
```

If you cannot exec into your Pod, you can also manually run these commands as a Postgres superuser.

Ensure the execution of this and any other SQL scripts completes successfully, otherwise your data may be unavailable.

Once this is done, your major upgrade is complete! Enjoy using your newer version of Postgres!
