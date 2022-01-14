### Postgres major upgrade test
* 00: Create a standard Postgres 13 cluster; verify that the cluster instance is created as expected and that the initial backup job completed successfully.
* 01: Run postgres-version-check.sh, which verifies the expected Postgres version is returned.
* 02: Update the spec from step 00 to upgrade from 13 to 14; verify that the instance is ready, the version on the spec is correct, and both the backup and upgrade Jobs have completed successfully.
* 03: Rerun postgres-version-check.sh to verify the expected Postgres version is returned.
