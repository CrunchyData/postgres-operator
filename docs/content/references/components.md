---
title: "Components and Compatibility"
date:
draft: false
weight: 110
---

## Kubernetes Compatibility

PGO, the Postgres Operator from Crunchy Data, is tested on the following platforms:

- Kubernetes 1.20+
- OpenShift 4.6+
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

| Component | Version | PGO Version Min. | PGO Version Max. |
|-----------|---------|------------------|------------------|
| `crunchy-pgadmin4` | 4.30 | 5.1.0 | {{< param operatorVersion >}} |
| `crunchy-pgbackrest` | 2.38 | 5.1.0 | {{< param operatorVersion >}} |
| `crunchy-pgbackrest` | 2.36 | 5.0.4 | {{< param operatorVersionLatestRel5_0 >}} |
| `crunchy-pgbackrest` | 2.35 | 5.0.3 | 5.0.3 |
| `crunchy-pgbackrest` | 2.33 | 5.0.0 | 5.0.2 |
| `crunchy-pgbouncer` | 1.16.2 | 5.1.0 | {{< param operatorVersion >}} |
| `crunchy-pgbouncer` | 1.16.1 | 5.0.4 | {{< param operatorVersion >}} |
| `crunchy-pgbouncer` | 1.15 | 5.0.0 | {{< param operatorVersion >}} |
| `crunchy-postgres` | {{< param postgresVersion14 >}} | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres` | {{< param postgresVersion13 >}} | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres` | {{< param postgresVersion12 >}} | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres` | {{< param postgresVersion11 >}} | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres` | {{< param postgresVersion10 >}} | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion14 >}}-3.1 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion13 >}}-3.1 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion13 >}}-3.0 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion12 >}}-3.0 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion12 >}}-2.5 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion11 >}}-2.5 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion11 >}}-2.4 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion10 >}}-2.4 | 5.0.3 | {{< param operatorVersion >}} |
| `crunchy-postgres-gis` | {{< param postgresVersion10 >}}-2.3 | 5.0.3 | {{< param operatorVersion >}} |

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

PostGIS enabled containers have both the Postgres and PostGIS software versions included. For example, Postgres 14 with PostGIS 3.1 would use the following tags:

- `{{< param postgres14GIS31ComponentTagUbi8 >}}`
- `{{< param postgres14GIS31TagUbi8 >}}`

## Extensions Compatibility

The following table defines the compatibility between Postgres extensions and versions of Postgres they are available in. The "Postgres version" corresponds with the major version of a Postgres container.

The table also lists the initial PGO version that the version of the extension is available in.

| Extension | Version | Postgres Versions | Initial PGO Version |
|-----------|---------|-------------------|---------------------|
| `pgAudit` | 1.6.2 | 14  | 5.1.0 |
| `pgAudit` | 1.6.1 | 14  | 5.0.4 |
| `pgAudit` | 1.6.0 | 14  | 5.0.3 |
| `pgAudit` | 1.5.2 | 13  | 5.1.0 |
| `pgAudit` | 1.5.0 | 13  | 5.0.0 |
| `pgAudit` | 1.4.3 | 12  | 5.1.0 |
| `pgAudit` | 1.4.1 | 12  | 5.0.0 |
| `pgAudit` | 1.3.4 | 11  | 5.1.0 |
| `pgAudit` | 1.3.2 | 11  | 5.0.0 |
| `pgAudit` | 1.2.4 | 10  | 5.1.0 |
| `pgAudit` | 1.2.2 | 10  | 5.0.0 |
| `pgAudit Analyze` | 1.0.8 | 14, 13, 12, 11, 10  | 5.0.3 |
| `pgAudit Analyze` | 1.0.7 | 13, 12, 11, 10  | 5.0.0 |
| `pg_cron` | 1.3.1 | 14, 13, 12, 11, 10  | 5.0.0 |
| `pg_partman` | 4.6.0 | 14, 13, 12, 11, 10  | 5.0.4 |
| `pg_partman` | 4.5.1 | 13, 12, 11, 10  | 5.0.0 |
| `pgnodemx` | 1.3.0 | 14, 13, 12, 11, 10  | 5.1.0 |
| `pgnodemx` | 1.2.0 | 14, 13, 12, 11, 10  | 5.0.4 |
| `pgnodemx` | 1.0.5 | 14, 13, 12, 11, 10  | 5.0.3 |
| `pgnodemx` | 1.0.4 | 13, 12, 11, 10  | 5.0.0 |
| `set_user` | 3.0.0 | 14, 13, 12, 11, 10  | 5.0.3 |
| `set_user` | 2.0.1 | 13, 12, 11, 10  | 5.0.2 |
| `set_user` | 2.0.0 | 13, 12, 11, 10  | 5.0.0 |
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
| `PostGIS` | 3.1 | 14, 13  | 5.0.0 |
| `PostGIS` | 3.0 | 13, 12  | 5.0.0 |
| `PostGIS` | 2.5 | 12, 11  | 5.0.0 |
| `PostGIS` | 2.4 | 11, 10  | 5.0.0 |
| `PostGIS` | 2.3 | 10  | 5.0.0 |
| `pgrouting` | 3.1.3 | 13 | 5.0.0 |
| `pgrouting` | 3.0.5 | 13, 12 | 5.0.0 |
| `pgrouting` | 2.6.3 | 12, 11, 10 | 5.0.0 |
