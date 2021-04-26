---
title: "Crunchy Postgres Exporter"
date:
draft: false
weight: 3
---

The crunchy-postgres-exporter container provides real time metrics about the PostgreSQL database
via an API. These metrics are scraped and stored by a [Prometheus](https://prometheus.io)
time-series database and are then graphed and visualized through the open source data
visualizer [Grafana](https://grafana.com/).

The crunchy-postgres-exporter container uses [pgMonitor](https://github.com/CrunchyData/pgmonitor) for advanced metric collection.
It is required that the `crunchy-postgres-ha` container has the `PGMONITOR_PASSWORD` environment
variable to create the appropriate user (`ccp_monitoring`) to collect metrics.

Custom queries to collect metrics can be specified by the user. By
mounting a **queries.yml** file to */conf* on the container, additional metrics
can be specified for the API to collect. For an example of a queries.yml file, see
[here](https://github.com/CrunchyData/pgmonitor/blob/master/exporter/postgres/queries_common.yml)

## Packages

The crunchy-postgres-exporter Docker image contains the following packages (versions vary depending on PostgreSQL version):

* PostgreSQL ({{< param postgresVersion13 >}}, {{< param postgresVersion12 >}}, {{< param postgresVersion11 >}}, {{< param postgresVersion10 >}}, {{< param postgresVersion96 >}} and {{< param postgresVersion95 >}})
* CentOS 8 - publicly available
* UBI 7, UBI 8  - customers only
* [PostgreSQL Exporter](https://github.com/wrouesnel/postgres_exporter)

## Environment Variables

### Required
**Name**|**Default**|**Description**
:-----|:-----|:-----
**EXPORTER_PG_PASSWORD**|none|Provides the password needed to generate the PostgreSQL URL required by the PostgreSQL Exporter to connect to a PG database.  Should typically match the `PGMONITOR_PASSWORD` value set in the `crunchy-postgres` container.|

### Optional
**Name**|**Default**|**Description**
:-----|:-----|:-----
**EXPORTER_PG_USER**|ccp_monitoring|Provides the username needed to generate the PostgreSQL URL required by the PostgreSQL Exporter to connect to a PG database.  Should typically be `ccp_monitoring` per the [crunchy-postgres](/container-specifications/crunchy-postgres) container specification (see environment varaible `PGMONITOR_PASSWORD`).
**EXPORTER_PG_HOST**|127.0.0.1|Provides the host needed to generate the PostgreSQL URL required by the PostgreSQL Exporter to connect to a PG database|
**EXPORTER_PG_PORT**|5432|Provides the port needed to generate the PostgreSQL URL required by the PostgreSQL Exporter to connect to a PG database|
**EXPORTER_PG_DATABASE**|postgres|Provides the name of the database used to generate the PostgreSQL URL required by the PostgreSQL Exporter to connect to a PG database|
**DATA_SOURCE_NAME**|None|Explicitly defines the URL for connecting to the PostgreSQL database (must be in the form of `postgresql://`).  If provided, overrides all other settings provided to generate the connection URL.
**CRUNCHY_DEBUG**|FALSE|Set this to true to enable debugging in logs. Note: this mode can reveal secrets in logs.
**POSTGRES_EXPORTER_PORT**|9187|Set the postgres-exporter port to listen on for web interface and telemetry.

### Viewing Cluster Metrics

To view a particular cluster's available metrics in a local browser window, port forwarding can be set up as follows.
For a pgcluster, `mycluster`, deployed in the `pgouser1` namespace, use

```
# If deployed to Kubernetes
kubectl port-forward -n pgouser1 svc/mycluster 9187:9187

# If deployed to OpenShift
oc port-forward -n pgouser1 svc/mycluster 9187:9187
```

Then, in your local browser, go to `http://127.0.0.1:9187/metrics` to view the available metrics for that cluster.

# Crunchy Postgres Exporter Metrics

You can find more information about the metrics available in the Crunchy Postgres Exporter by visiting the [pgMonitor](https://github.com/CrunchyData/pgmonitor) project and viewing details in the [exporter](https://github.com/CrunchyData/pgmonitor/tree/master/exporter/postgres) folder.

# [pgnodemx](https://github.com/CrunchyData/pgnodemx)

In addition to the metrics above, the [pgnodemx](https://github.com/CrunchyData/pgnodemx) PostgreSQL extension provides SQL functions to allow the capture of node OS metrics via SQL queries. For more information, please see the [pgnodemx](https://github.com/CrunchyData/pgnodemx) project page:

[https://github.com/CrunchyData/pgnodemx](https://github.com/CrunchyData/pgnodemx)
