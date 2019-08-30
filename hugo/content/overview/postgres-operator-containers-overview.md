---
title: "PostgreSQL Operator Containers Overview"
date:
draft: false
weight: 6
---

## PostgreSQL Operator Containers Overview

The PostgreSQL Operator orchestrates a series of PostgreSQL and PostgreSQL related containers containers that enable rapid deployment of PostgreSQL, including administration and monitoring tools in a Kubernetes environment. The PostgreSQL Operator supports PostgreSQL 9.5+ with multiple PostgreSQL cluster deployment strategies and a variety of PostgreSQL related extensions and tools enabling enterprise grade PostgreSQL-as-a-Service.   A full list of the containers supported by the PostgreSQL Operator is provided below.   

### PostgreSQL Server and Extensions

* **PostgreSQL** (crunchy-postgres).  PostgreSQL database server.  The crunchy-postgres container image is unmodified, open source PostgreSQL packaged and maintained by Crunchy Data. 

* **PostGIS** (crunchy-postgres-gis).  PostgreSQL database server including the PostGIS extension. The crunchy-postgres-gis container image is unmodified, open source PostgreSQL packaged and maintained by Crunchy Data. This image is identical to the crunchy-postgres image except it includes the open source geospatial extension PostGIS for PostgreSQL in addition to the language extension PL/R which allows for writing functions in the R statistical computing language.

### Backup and Restore

* **pgBackRest** (crunchy-backrest-restore). pgBackRest is a high performance backup and restore utility for PostgreSQL.  The crunchy-backrest-restore container executes the pgBackRest utility, allowing FULL and DELTA restore capability.

* **pg_basebackup** (crunchy-backup). pg_basebackup is used to take base backups of a running PostgreSQL database cluster. The crunchy-backup container executes a full backup against another database container using the standard pg_basebackup utility that is included with PostgreSQL.

* **pgdump** (crunchy-pgdump). The crunchy-pgdump container executes either a pg_dump or pg_dumpall database backup against another PostgreSQL database.

* **crunchy-pgrestore** (restore). The restore image provides a means of performing a restore of a dump from pg_dump or pg_dumpall via psql or pg_restore to a PostgreSQL container database.


### Administration Tools

* **pgAdmin4** (crunchy-pgadmin4). PGAdmin4 is a graphical user interface administration tool for PostgreSQL.  The crunchy-pgadmin4 container executes the pgAdmin4 web application.

* **pgbadger** (crunchy-pgbadger).  pgbadger is a PostgreSQL log analyzer with fully detailed reports and graphs.  The crunchy-pgbadger container executes the pgBadger utility, which generates a PostgreSQL log analysis report using a small HTTP server running on the container.

* **pg_upgrade**  (crunchy-upgrade). The crunchy-upgrade container contains 9.5, 9.6, 10, and 11 PostgreSQL packages in order to perform a pg_upgrade from 9.5 to 9.6, 9.6 to 10, and 10 to 11 versions.

* **scheduler** (crunchy-scheduler).  The crunchy-scheduler container provides a cron like microservice for automating pgBaseBackup and pgBackRest backups within a single namespace. 

### Metrics and Monitoring

* **Metrics Collection** (crunchy-collect). The crunchy-collect container provides real time metrics about the PostgreSQL database via an API. These metrics are scraped and stored by a Prometheus time-series database and are then graphed and visualized through the open source data visualizer Grafana.  

* **Grafana** (crunchy-grafana).  Grafana is an open source Visual dashboards are created from the collected and stored data that crunchy-collect and crunchy-prometheus provide for the crunchy-grafana container, which hosts an open source web-based graphing dashboard called Grafana.

* **Prometheus** (crunchy-prometheus).  Prometheus is a multi-dimensional time series data model with an elastic query language. It is used in collaboration with Crunchy Collect and Grafana to provide metrics.

### Connection Pooling and Load Balancing

* **pgbouncer** (crunchy-pgbouncer).  pgbouncer is a lightweight connection pooler for PostgreSQL. The crunchy-pgbouncer container provides a pgbouncer image.

* **pgpool** (crunchy-pgpool).  pgPool II is a middleware that works between PostgreSQL servers and a PostgreSQL database client.  The crunchy-pgpool container executes the utility. pgPool can be used to provide a smart PostgreSQL-aware proxy to a PostgreSQL cluster, both primary and replica, so that applications only have to work with a single database connection.





