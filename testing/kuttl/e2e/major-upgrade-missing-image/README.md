## Major upgrade missing image tests

This is a variation derived from our major upgrade KUTTL tests designed to
test scenarios where required container images are not defined in either the
PostgresCluster spec or via the RELATED_IMAGES environment variables.

### Basic PGUpgrade controller and CRD instance validation

* 01--valid-upgrade: create a valid PGUpgrade instance
* 01-assert: check that the PGUpgrade instance exists and has the expected status

### Verify new statuses for missing required container images

* 10--cluster: create the cluster with an unavailable image (i.e. Postgres 11)
* 10-assert: check that the PGUpgrade instance has the expected reason: "PGClusterNotShutdown"
* 11-shutdown-cluster: set the spec.shutdown value to 'true' as required for upgrade
* 11-assert: check that the new reason is set, "PGClusterPrimaryNotIdentified"

### Update to an available Postgres version, start and upgrade PostgresCluster

* 12--start-and-update-version: update the Postgres version on both CRD instances and set 'shutdown' to false
* 12-assert: verify that the cluster is running and the PGUpgrade instance now has the new status info with reason: "PGClusterNotShutdown"
* 13--shutdown-cluster: set spec.shutdown to 'true'
* 13-assert: check that the PGUpgrade instance has the expected reason: "PGClusterMissingRequiredAnnotation"
* 14--annotate-cluster: set the required annotation
* 14-assert: verify that the upgrade succeeded and the new Postgres version shows in the cluster's status
* 15--start-cluster: set the new Postgres version and spec.shutdown to 'false'

### Verify upgraded PostgresCluster

* 15-assert: verify that the cluster is running
* 16-check-pgbackrest: check that the pgbackrest setup has successfully completed
* 17--check-version: check the version reported by PostgreSQL
* 17-assert: assert the Job from the previous step succeeded


