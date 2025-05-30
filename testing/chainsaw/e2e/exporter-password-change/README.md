# Exporter Password Change

## 00--create-cluster:
The TestStep will:

1) Apply the `files/inital-postgrescluster.yaml` file to create a cluster with monitoring enabled
2) Assert that conditions outlined in `files/initial-postgrescluster-checks.yaml` are met
    - PostgresCluster exists with a single ready replica
    - A pod with `cluster` and `crunchy-postgres-exporter` labels has the status `{phase: Running}`
    - A `<cluster>-monitoring` secret exists with correct labels and ownerReferences

## 00-assert:

This TestAssert will loop through a script until:
1) the instance pod has the `ContainersReady` condition with status `true`
2) the asserts from `00--create-cluster` are met.

## 01-assert:

This TestAssert will loop through a script until:
1) The metrics endpoint returns `pg_exporter_last_scrape_error 0` meaning the exporter was able to access postgres metrics
2) It is able to store the pid of the running postgres_exporter process

## 02-change-password:

This TestStep will:
1) Apply the `files/update-monitoring-password.yaml` file to set the monitoring password to `password`
2) Assert that conditions outlined in `files/update-monitoring-password-checks.yaml` are met
    - A `<cluster>-monitoring` secret exists with `data.password` set to the encoded value for `password`

## 02-assert:

This TestAssert will loop through a script until:
1) An exec command can confirm that `/opt/crunchy/password` file contains the updated password
2) It can confirm that the pid of the postgres_exporter process has changed
3) The metrics endpoint returns `pg_exporter_last_scrape_error 0` meaning the exporter was able to access postgres metrics using the updated password
