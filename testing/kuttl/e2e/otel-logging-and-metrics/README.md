# Test OTel Logging and Metrics

## Assumptions

This test assumes that the operator has both OpenTelemetryLogs and OpenTelemetryMetrics feature gates turned on and that you are using an operator versioned 5.8 or greater.

## Process

1. Create a basic cluster with pgbouncer and pgadmin in place.
    1. Ensure cluster comes up, that all containers are running and ready, and that the initial backup is complete.
2. Add the `instrumentation` spec to both PostgresCluster and PGAdmin manifests.
    1. Ensure that OTel collector containers and `crunchy-otel-collector` labels are added to the four pods (postgres instance, repo-host, pgbouncer, & pgadmin) and that the collector containers are running and ready.
    2. Assert that the instance pod collector is getting postgres and patroni metrics and postgres, patroni, and pgbackrest logs.
    3. Assert that the pgbouncer pod collector is getting pgbouncer metrics and logs.
    4. Assert that the pgAdmin pod collector is getting pgAdmin and gunicorn logs.
    5. Assert that the repo-host pod collector is NOT getting pgbackrest logs. We do not expect logs yet as the initial backup completed and created a log file; however, we configure the collector to only ingest new logs after it has started up.
    6. Create a manual backup and ensure that it completes successfully.
    7. Ensure that the repo-host pod collector is now getting pgbackrest logs.
3. Add both "add" and "remove" custom queries to the PostgresCluster `instrumentation` spec and create a ConfigMap that holds the custom queries to add.
    1. Ensure that the ConfigMap is created.
    2. Assert that the metrics that were removed (which we checked for earlier) are in fact no longer present in the collector metrics.
    3. Assert that the custom metrics that were added are present in the collector metrics.
4. Add an `otlp` exporter to both PostgresCluster and PGAdmin `instrumentation` specs and create a standalone OTel collector to receive data from our sidecar collectors.
    1. Ensure that the ConfigMap, Service, and Deployment for the standalone OTel collector come up and that the collector container is running and ready.
    2. Assert that the standalone collector is receiving logs from all of our components (i.e. the standalone collector is getting logs for postgres, patroni, pgbackrest, pgbouncer, pgadmin, and gunicorn).

### NOTES

It is possible this test could flake if for some reason a component is not producing any logs. If we start to see this happen, we could either create some test steps that execute some actions that should trigger logs or turn up the log levels (although the latter option could create more problems as we have seen issues with the collector when the stream of logs is too voluminous).
