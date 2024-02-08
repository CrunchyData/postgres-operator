# Tablespace Enabled

**Note**: This series of tests depends on PGO being deployed with the `TablespaceVolume` feature gate enabled.

00: Start a cluster with tablespace volumes and a configmap `databaseInitSQL` to create tablespaces with the non-superuser as owner
01: Connect to the db; check that the tablespaces exist; create tables in the tablespaces; and create a table outside the tablespaces and move it into a tablespace
