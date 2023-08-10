## Empty image upgrade status tests

This is a variation derived from our major upgrade KUTTL tests designed to
test a scenario where a required container images is not defined in either the
PostgresCluster spec or via the RELATED_IMAGES environment variables.

### Basic PGUpgrade controller and CRD instance validation

* 01--valid-upgrade: create a valid PGUpgrade instance
* 01-assert: check that the PGUpgrade instance exists and has the expected status

### Verify new statuses for missing required container images

* 10--cluster: create the cluster with an unavailable image (i.e. Postgres 10)
* 10-assert: check that the PGUpgrade instance has the expected reason: "PGClusterNotShutdown"
* 11-shutdown-cluster: set the spec.shutdown value to 'true' as required for upgrade
* 11-assert: check that the new reason is set, "PGClusterPrimaryNotIdentified"
