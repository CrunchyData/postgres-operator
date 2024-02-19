## Cluster Migrate

This test was developed to check that users could bypass some known problems when
migrating from a non-Crunchy PostgreSQL image to a Crunchy PostgreSQL image:

1) it changes the ownership of the data directory (which depends on fsGroup
behavior to change group ownership which is not available in all providers);
2) it makes sure a postgresql.conf file is available, as required by Patroni.

Important note on *environment*:
As noted above, this work relies on fsGroup, so this test will not work in the current
form in all environments. For instance, this creates a PG cluster with fsGroup set,
which will result in an error in OpenShift.

Important note on *PV permissions*:
This test involves changing permissions on PersistentVolumes, which may not be available
in all environments to all users (since this is a cluster-wide permission).

Important note on migrating between different builds of *Postgres 15*:
PG 15 introduced new behavior around database collation versions, which result in errors like:

```
WARNING:  database \"postgres\" has a collation version mismatch
DETAIL:  The database was created using collation version 2.31, but the operating system provides version 2.28
```

This error occurred in `reconcilePostgresDatabases` and prevented PGO from finishing the reconcile
loop. For _testing purposes_, this problem is worked around in steps 06 and 07, which wait for
the PG pod to be ready and then send a command to `REFRESH COLLATION VERSION` on the `postgres`
and `template1` databases (which were the only databases where this error was observed during
testing).

This solution is fine for testing purposes, but is not a solution that should be done in production
as an automatic step. User intervention and supervision is recommended in that case.

### Steps

* 01: Create a non-Crunchy PostgreSQL cluster and wait for it to be ready
* 02: Create data on that cluster
* 03: Alter the Reclaim policy of the PV so that it will survive deletion of the cluster
* 04: Delete the original cluster, leaving the PV
* 05: Create a PGO-managed `postgrescluster` with the remaing PV as the datasource
* 06-07: Wait for the PG pod to be ready and alter the collation (PG 15 only, see above)
* 08: Alter the PV to the original Reclaim policy
* 09: Check that the data successfully migrated
