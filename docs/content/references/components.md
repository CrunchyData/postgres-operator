---
title: "Components and Compatibility"
date:
draft: false
weight: 110
---

## Kubernetes Compatibility

PGO, the Postgres Operator from Crunchy Data, is tested on the following platforms:

- Kubernetes 1.18+
- OpenShift 4.5+
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

## Components Compatibility

The following table defines the compatibility between PGO and the various component containers
needed to deploy PostgreSQL clusters using PGO.

The listed versions of Postgres show the latest minor release (e.g. 13.4) of each major version (e.g. 13). Older minor releases may still be compatible with PGO. We generally recommend to run the latest minor release for the [same reasons that the PostgreSQL community provides](https://www.postgresql.org/support/versioning/).

| Component | Version | PGO Version Min. | PGO Version Max. |
|-----------|---------|------------------|------------------|
| `crunchy-pgbackrest` | 2.33 | 5.0.0 | 5.0.2 |
| `crunchy-pgbouncer` | 1.15 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-ha` | 13.4 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-ha` | 12.8 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-ha` | 11.13 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-ha` | 10.18 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 13.4-3.1 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 13.4-3.0 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 12.8-3.0 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 12.8-2.5 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 11.13-2.5 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 11.13-2.4 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 10.18-2.4 | 5.0.0 | 5.0.2 |
| `crunchy-postgres-gis-ha` | 10.18-2.3 | 5.0.0 | 5.0.2 |

The Crunchy Postgres components include Patroni 2.1.0.

## Extensions Compatibility

The following table defines the compatibility between Postgres extensions and versions of Postgres they are available in. The "Postgres version" corresponds with the major version of a Postgres container.

The table also lists the initial PGO version that the version of the extension is available in.

| Extension | Version | Postgres Versions | Initial PGO Version |
|-----------|---------|-------------------|---------------------|
| `pgAudit` | 1.5.0 | 13  | 5.0.0 |
| `pgAudit` | 1.4.1 | 12  | 5.0.0 |
| `pgAudit` | 1.3.2 | 11  | 5.0.0 |
| `pgAudit` | 1.2.2 | 10  | 5.0.0 |
| `pgAudit Analyze` | 1.0.7 | 13, 12, 11, 10  | 5.0.0 |
| `pg_cron` | 1.3.1 | 13, 12, 11, 10  | 5.0.0 |
| `pg_partman` | 4.5.1 | 13, 12, 11, 10  | 5.0.0 |
| `pgnodemx` | 1.0.4 | 13, 12, 11, 10  | 5.0.0 |
| `set_user` | 2.0.1 | 13, 12, 11, 10  | 5.0.2 |
| `set_user` | 2.0.0 | 13, 12, 11, 10  | 5.0.0 |
| `TimescaleDB` | 2.4.0 | 13, 12  | 5.0.2 |
| `TimescaleDB` | 2.3.1 | 11  | 5.0.1 |
| `TimescaleDB` | 2.2.0 | 13, 12, 11  | 5.0.0 |
| `wal2json` | 2.3 | 13, 12, 11, 10 | 5.0.0 |

### Geospatial Extensions

The following extensions are available in the geospatially aware containers (`crunchy-postgres-gis-ha`):

| Extension | Version | Postgres Versions | Initial PGO Version |
|-----------|---------|-------------------|---------------------|
| `PostGIS` | 3.1 | 13  | 5.0.0 |
| `PostGIS` | 3.0 | 13, 12  | 5.0.0 |
| `PostGIS` | 2.5 | 12, 11  | 5.0.0 |
| `PostGIS` | 2.4 | 11, 10  | 5.0.0 |
| `PostGIS` | 2.3 | 10  | 5.0.0 |
| `pgrouting` | 3.1.3 | 13 | 5.0.0 |
| `pgrouting` | 3.0.5 | 13, 12 | 5.0.0 |
| `pgrouting` | 2.6.3 | 12, 11, 10 | 5.0.0 |
