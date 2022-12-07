---
title: "Components and Compatibility"
date:
draft: false
weight: 110
---

## Kubernetes Compatibility

PGO, the Postgres Operator from Crunchy Data, is tested on the following platforms:

- Kubernetes 1.22-1.25
- OpenShift 4.8-4.11
- Rancher
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

## Components Compatibility

The following table defines the compatibility between PGO and the various component containers
needed to deploy PostgreSQL clusters using PGO.

The listed versions of Postgres show the latest minor release (e.g. {{< param postgresVersion13 >}}) of each major version (e.g. {{< param postgresVersion >}}). Older minor releases may still be compatible with PGO. We generally recommend to run the latest minor release for the [same reasons that the PostgreSQL community provides](https://www.postgresql.org/support/versioning/).

Note that for the 5.0.3 release and beyond, the Postgres containers were renamed to `crunchy-postgres` and `crunchy-postgres-gis`.

| PGO | pgAdmin* | pgBackRest | PgBouncer | Postgres | PostGIS |
|-----|---------|------------|-----------|----------|---------|
| `5.3.0` | `4.30` | `2.41` | `1.17` | `15,14,13,12,11` | `3.3,3.2,3.1,3.0,2.5,2.4` |
| `5.2.1` | `4.30` | `2.41` | `1.17` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.2.0` | `4.30` | `2.40` | `1.17` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.1.4` | `4.30` | `2.41` | `1.17` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.1.3` | `4.30` | `2.40` | `1.17` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.1.2` | `4.30` | `2.38` | `1.16` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.1.1` | `4.30` | `2.38` | `1.16` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.1.0` | `4.30` | `2.38` | `1.16` | `14,13,12,11,10` | `3.1,3.0,2.5,2.4,2.3` |
| `5.0.9` | `n/a` | `2.41` | `1.17` | `14,13,12,11,10` | `3.1,3.0,2.5,2.4,2.3` |
| `5.0.8` | `n/a` | `2.40` | `1.17` | `14,13,12,11,10` | `3.1,3.0,2.5,2.4,2.3` |
| `5.0.7` | `n/a` | `2.38` | `1.16` | `14,13,12,11,10` | `3,2,3.1,3.0,2.5,2.4,2.3` |
| `5.0.6` | `n/a` | `2.38` | `1.16` | `14,13,12,11,10` | `3.2,3.1,3.0,2.5,2.4,2.3` |
| `5.0.5` | `n/a` | `2.36` | `1.16` | `14,13,12,11,10` | `3.1,3.0,2.5,2.4,2.3` |
| `5.0.4` | `n/a` | `2.36` | `1.16` | `14,13,12,11,10` | `3.1,3.0,2.5,2.4,2.3` |
| `5.0.3` | `n/a` | `2.35` | `1.15` | `14,13,12,11,10` | `3.1,3.0,2.5,2.4,2.3` |

_*pgAdmin 4.30 does not currently support Postgres 15._

The latest Postgres containers include Patroni 2.1.3.

The following are the Postgres containers available for version 5.0.2 of PGO and older:

| Component | Version | PGO Version Min. | PGO Version Max. |
|-----------|---------|------------------|------------------|
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

### Container Tags

The container tags follow one of two patterns:

- `<baseImage>-<softwareVersion>-<buildVersion>`
- `<baseImage>-<softwareVersion>-<pgoVersion>-<buildVersion>` (Customer Portal only)

For example, when pulling from the [customer portal](https://access.crunchydata.com/), the following would both be valid tags to reference the PgBouncer container:

- `{{< param PGBouncerComponentTagUbi8 >}}`
- `{{< param PGBouncerTagUbi8 >}}`

On the [developer portal](https://www.crunchydata.com/developers/download-postgres/containers), PgBouncer would use this tag:

- `{{< param PGBouncerComponentTagUbi8 >}}`

PostGIS enabled containers have both the Postgres and PostGIS software versions included. For example, Postgres 14 with PostGIS 3.2 would use the following tags:

- `{{< param postgres14GIS32ComponentTagUbi8 >}}`
- `{{< param postgres14GIS32TagUbi8 >}}`

## Extensions Compatibility

The following table defines the compatibility between Postgres extensions and versions of Postgres they are available in. The "Postgres version" corresponds with the major version of a Postgres container.

The table also lists the initial PGO version that the version of the extension is available in.

| Extension | Version | Postgres Versions | Initial PGO Version |
|-----------|---------|-------------------|---------------------|
| `orafce` | 3.25.1 | 15, 14, 13, 12, 11  | 5.3.0 |
| `orafce` | 3.22.0 | 14, 13, 12, 11, 10  | 5.2.0 |
| `orafce` | 3.22.0 | 14, 13, 12, 11, 10  | 5.1.3 |
| `orafce` | 3.22.0 | 14, 13, 12, 11, 10  | 5.0.8 |
| `pgAudit` | 1.6.2 | 14  | 5.1.0 |
| `pgAudit` | 1.6.2 | 14  | 5.0.6 |
| `pgAudit` | 1.6.1 | 14  | 5.0.4 |
| `pgAudit` | 1.6.0 | 14  | 5.0.3 |
| `pgAudit` | 1.5.2 | 13  | 5.1.0 |
| `pgAudit` | 1.5.2 | 13  | 5.0.6 |
| `pgAudit` | 1.5.0 | 13  | 5.0.0 |
| `pgAudit` | 1.4.3 | 12  | 5.1.0 |
| `pgAudit` | 1.4.1 | 12  | 5.0.0 |
| `pgAudit` | 1.3.4 | 11  | 5.1.0 |
| `pgAudit` | 1.3.4 | 11  | 5.0.6 |
| `pgAudit` | 1.3.2 | 11  | 5.0.0 |
| `pgAudit` | 1.2.4 | 10  | 5.1.0 |
| `pgAudit` | 1.2.4 | 10  | 5.0.6 |
| `pgAudit` | 1.2.2 | 10  | 5.0.0 |
| `pgAudit Analyze` | 1.0.8 | 14, 13, 12, 11, 10  | 5.0.3 |
| `pgAudit Analyze` | 1.0.7 | 13, 12, 11, 10  | 5.0.0 |
| `pg_cron` | 1.3.1 | 14, 13, 12, 11, 10  | 5.0.0 |
| `pg_partman` | 4.7.1 | 15, 14, 13, 12, 11  | 5.3.0 |
| `pg_partman` | 4.6.2 | 14, 13, 12, 11, 10  | 5.2.0 |
| `pg_partman` | 4.6.2 | 14, 13, 12, 11, 10  | 5.1.3 |
| `pg_partman` | 4.6.2 | 14, 13, 12, 11, 10  | 5.0.8 |
| `pg_partman` | 4.6.1 | 14, 13, 12, 11, 10  | 5.1.1 |
| `pg_partman` | 4.6.1 | 14, 13, 12, 11, 10  | 5.0.6 |
| `pg_partman` | 4.6.0 | 14, 13, 12, 11, 10  | 5.0.4 |
| `pg_partman` | 4.5.1 | 13, 12, 11, 10  | 5.0.0 |
| `pgnodemx` | 1.3.0 | 14, 13, 12, 11, 10  | 5.1.0 |
| `pgnodemx` | 1.3.0 | 14, 13, 12, 11, 10  | 5.0.6 |
| `pgnodemx` | 1.2.0 | 14, 13, 12, 11, 10  | 5.0.4 |
| `pgnodemx` | 1.0.5 | 14, 13, 12, 11, 10  | 5.0.3 |
| `pgnodemx` | 1.0.4 | 13, 12, 11, 10  | 5.0.0 |
| `set_user` | 3.0.0 | 14, 13, 12, 11, 10  | 5.0.3 |
| `set_user` | 2.0.1 | 13, 12, 11, 10  | 5.0.2 |
| `set_user` | 2.0.0 | 13, 12, 11, 10  | 5.0.0 |
| `TimescaleDB` | 2.8.1 | 14, 13, 12  | 5.3.0 |
| `TimescaleDB` | 2.6.1 | 14, 13, 12  | 5.1.1 |
| `TimescaleDB` | 2.6.1 | 14, 13, 12  | 5.0.6 |
| `TimescaleDB` | 2.6.0 | 14, 13, 12  | 5.1.0 |
| `TimescaleDB` | 2.5.0 | 14, 13, 12  | 5.0.3 |
| `TimescaleDB` | 2.4.2 | 13, 12  | 5.0.3 |
| `TimescaleDB` | 2.4.0 | 13, 12  | 5.0.2 |
| `TimescaleDB` | 2.3.1 | 11  | 5.0.1 |
| `TimescaleDB` | 2.2.0 | 13, 12, 11  | 5.0.0 |
| `wal2json` | 2.4 | 14, 13, 12, 11, 10 | 5.0.3 |
| `wal2json` | 2.3 | 13, 12, 11, 10 | 5.0.0 |

### Geospatial Extensions

The following extensions are available in the geospatially aware containers (`crunchy-postgres-gis`):

| Extension | Version | Postgres Versions | Initial PGO Version |
|-----------|---------|-------------------|---------------------|
| `PostGIS` | 3.2 | 14  | 5.1.1 |
| `PostGIS` | 3.2 | 14  | 5.0.6 |
| `PostGIS` | 3.1 | 14, 13  | 5.0.0 |
| `PostGIS` | 3.0 | 13, 12  | 5.0.0 |
| `PostGIS` | 2.5 | 12, 11  | 5.0.0 |
| `PostGIS` | 2.4 | 11, 10  | 5.0.0 |
| `PostGIS` | 2.3 | 10  | 5.0.0 |
| `pgrouting` | 3.1.3 | 13 | 5.0.0 |
| `pgrouting` | 3.0.5 | 13, 12 | 5.0.0 |
| `pgrouting` | 2.6.3 | 12, 11, 10 | 5.0.0 |
