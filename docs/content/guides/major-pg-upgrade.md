---
title: "PostgreSQL Major Upgrade"
date:
draft: false
weight: 100
---

You can perform a major PostgreSQL upgrade from one version to another using PGO. Described below are the 
steps necessary to perform your upgrade, which uses the 
[pg_upgrade](https://www.postgresql.org/docs/current/pgupgrade.html) tool.

{{% notice warning %}}
Please note that your PostgresCluster will need to be in a healthy state in order for the upgrade to 
complete as expected. If there are any present issues, such as Pods that are not running correctly or 
other similar problems, they will need to be addressed before proceeding!
{{% /notice %}}

### Step 1: Take a pgBackRest Backup

Before starting your major upgrade, you should take a new database 
[backup]({{< relref "tutorial/backup-management.md" >}}). This will offer another layer of data 
protection in cases where the upgrade process does not complete as expected.

### Step 2: Scale Down Replicas

PGO needs to identify the primary database instance before running your upgrade because the primary
pgdata volume will need to be mounted to the upgrade Job. Any cluster replicas will not be
upgraded and will need to be recreated after the upgrade and post-upgrade tasks are completed. To ensure
errors are avoided, any replicas must be scaled down before initiating the upgrade process. 

Scaling down is simple: your manifest should be configured to have only one instance named under `spec.instances`
and the `replicas` value, if set, should be equal to `1`. For example, if your existing configuration is

```
spec:
  instances:
    - name: instancea
      replicas: 3
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteMany"
        resources:
          requests:
            storage: 1Gi
    - name: instanceb
      replicas: 2
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteMany"
        resources:
          requests:
            storage: 1Gi
```

You will need to set it to something similar to 

```
spec:
  instances:
    - name: instancea
      replicas: 1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteMany"
        resources:
          requests:
            storage: 1Gi
```

then apply the changes with

```
kubectl apply -k examples/postgrescluster
```

and wait for them to complete before continuing on to the next steps.

<!-- TODO(tjmoore4): This step should not be required after follow on work to run pgBackRest stanza 
upgrade and backup automatically during upgrade process. This will allow the replicas to be automatically
scaled down before the upgrade and back up once the post-upgrade steps are completed. -->

### Step 3: Configure the PostgresCluster

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

### Step 4: Run the PostgreSQL Upgrade

Once everything is configured as described above, the upgrade can be started by running

```
kubectl apply -k examples/postgrescluster
```

This will then terminate the database instance and start the pg_upgrade Job. This Job will perform the
necessary steps to upgrade the PostgresCluster to the desired version. Once this Job completes, the
primary instance will then be started and should return to the previous running state.

### Step 5: Remove the Upgrade Configuration

Once the upgrade Job has completed and the primary instance is running, the upgrade section configured above, i.e.

```
spec:
  upgrade:
    enabled: true
    fromPostgresVersion: {{< param fromPostgresVersion >}}
    image: {{< param imageCrunchyPGUpgrade >}}
```

should be removed.

### Step 6: Complete the Post-Upgrade Tasks

<!-- TODO(tjmoore4): These steps may be automated in follow-on work, where possible. -->

After the upgrade Job has completed, there will be some amount of post-upgrade processing that
needs to be done. During the upgrade process, `pg_upgrade` will issue warnings and create scripts
to perform the needed follow on work.

**Note that these scripts will need to be run as an administrator.**

This information can be viewed by examining the Job logs, with a command similar to

```
kubectl -n postgres-operator logs hippo-pgupgrade-abcd
```

For example, in the completed upgrade Job logs, you may see messages such as

```
Optimizer statistics are not transferred by pg_upgrade so,
once you start the new server, consider running:
    ./analyze_new_cluster.sh

Running this script will delete the old cluster's data files:
    ./delete_old_cluster.sh
```

The first script, `analyze_new_cluster.sh` will contain something similar to:

```
#!/bin/sh

echo 'This script will generate minimal optimizer statistics rapidly'
echo 'so your system is usable, and then gather statistics twice more'
echo 'with increasing accuracy.  When it is done, your system will'
echo 'have the default level of optimizer statistics.'
echo

echo 'If you have used ALTER TABLE to modify the statistics target for'
echo 'any tables, you might want to remove them and restore them after'
echo 'running this script because they will delay fast statistics generation.'
echo

echo 'If you would like default statistics as quickly as possible, cancel'
echo 'this script and run:'
echo '    "/usr/pgsql-{{< param postgresVersion >}}/bin/vacuumdb" --all --analyze-only'
echo

"/usr/pgsql-{{< param postgresVersion >}}/bin/vacuumdb" --all --analyze-in-stages
echo

echo 'Done'
```

The second script will contain something similar to 

```
#!/bin/sh

rm -rf '/pgdata/{{< param fromPostgresVersion >}}'
```

There also may be an `update_extensions.sql` file created, to facilitate extension updates
once the upgrade has been completed. If, for instance, the `pgaudit` extension was used in
 the hippo cluster before the upgrade, the resulting file would contain something like

```
\connect hippo
ALTER EXTENSION "pgaudit" UPDATE;
\connect postgres
ALTER EXTENSION "pgaudit" UPDATE;
\connect template1
ALTER EXTENSION "pgaudit" UPDATE;
```

These scripts can be run from inside the database container by using the `kubectl exec`
method of logging in:

```
$ kubectl exec -it -n postgres-operator -c database \
  $(kubectl get pods -n postgres-operator --selector='postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/role=master' -o name) -- bash
```

As these scripts will be located in `/pgdata`, they can be run as follows:

`$ /pgdata/analyze_new_cluster.sh`
`$ /pgdata/delete_old_cluster.sh`

If the `update_extensions.sql` file is created, it can be run with

`$ psql -f /pgdata/update_extensions.sql`

When executing these commands, it is important that they finish successfully. As noted in the
`Post-upgrade processing` step of the 
[pg_upgrade documentation](https://www.postgresql.org/docs/current/pgupgrade.html),

{{% notice warning %}}
"In general it is unsafe to access tables referenced in rebuild scripts until the rebuild scripts
have run to completion; doing so could yield incorrect results or poor performance. Tables not 
referenced in rebuild scripts can be accessed immediately."
{{% /notice %}}

<!-- TODO(tjmoore4): These steps assume the stanza upgrade and initial backup for the new Postgres version
have been accomplished automatically, as scheduled in a follow on task. -->

Once these scripts are successfully executed, the upgrade process is complete!
