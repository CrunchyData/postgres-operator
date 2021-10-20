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
- Rancher
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

## Components Compatibility

The following table defines the compatibility between PGO and the various component containers
needed to deploy PostgreSQL clusters using PGO.

The listed versions of Postgres show the latest minor release (e.g. 13.4) of each major version (e.g. 13). Older minor releases may still be compatible with PGO. We generally recommend to run the latest minor release for the [same reasons that the PostgreSQL community provides](https://www.postgresql.org/support/versioning/).

Note that for the 5.0.3 release and beyond, the Postgres containers were renamed to `crunchy-postgres` and `crunchy-postgres-gis`.

| Component | Version | PGO Version Min. | PGO Version Max. |
|-----------|---------|------------------|------------------|
| `crunchy-pgbackrest` | 2.35 | 5.0.3 | 5.0.3 |
| `crunchy-pgbackrest` | 2.33 | 5.0.0 | 5.0.2 |
| `crunchy-pgbouncer` | 1.15 | 5.0.0 | 5.0.3 |
| `crunchy-postgres` | 14.0 | 5.0.3 | 5.0.3 |
| `crunchy-postgres` | 13.4 | 5.0.3 | 5.0.3 |
| `crunchy-postgres` | 12.8 | 5.0.3 | 5.0.3 |
| `crunchy-postgres` | 11.13 | 5.0.3 | 5.0.3 |
| `crunchy-postgres` | 10.18 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 14.0-3.1 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 13.4-3.1 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 13.4-3.0 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 12.8-3.0 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 12.8-2.5 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 11.13-2.5 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 11.13-2.4 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 10.18-2.4 | 5.0.3 | 5.0.3 |
| `crunchy-postgres-gis` | 10.18-2.3 | 5.0.3 | 5.0.3 |

The latest Postgres containers include Patroni 2.1.1.

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

For example, if pulling from the [customer portal](https://access.crunchydata.com/), the following would both be valid tags to reference the pgBouncer container:

- `ubi8-1.15-3`
- `ubi8-1.15-5.0.3-0`
- `centos8-1.15-3`
- `centos8-1.15-5.0.3-0`

The [developer portal](https://www.crunchydata.com/developers/download-postgres/containers) provides CentOS based images. For example, pgBouncer would use this tag:

- `centos8-1.15-3`

PostGIS enabled containers have both the Postgres and PostGIS software versions included. For example, Postgres 14 with Postgis 3.1 would use the following tags:

- `ubi8-14.0-3.1-0`
- `ubi8-14.0-3.1-5.0.3-0`
- `centos8-14.0-3.1-0`
- `centos8-14.0-3.1-5.0.3-0`

## Extensions Compatibility

The following table defines the compatibility between Postgres extensions and versions of Postgres they are available in. The "Postgres version" corresponds with the major version of a Postgres container.

The table also lists the initial PGO version that the version of the extension is available in.

| Extension | Version | Postgres Versions | Initial PGO Version |
|-----------|---------|-------------------|---------------------|
| `pgAudit` | 1.6.0 | 14  | 5.0.3 |
| `pgAudit` | 1.5.0 | 13  | 5.0.0 |
| `pgAudit` | 1.4.1 | 12  | 5.0.0 |
| `pgAudit` | 1.3.2 | 11  | 5.0.0 |
| `pgAudit` | 1.2.2 | 10  | 5.0.0 |
| `pgAudit Analyze` | 1.0.8 | 14, 13, 12, 11, 10  | 5.0.3 |
| `pgAudit Analyze` | 1.0.7 | 13, 12, 11, 10  | 5.0.0 |
| `pg_cron` | 1.3.1 | 14, 13, 12, 11, 10  | 5.0.0 |
| `pg_partman` | 4.5.1 | 13, 12, 11, 10  | 5.0.0 |
| `pgnodemx` | 1.0.5 | 14, 13, 12, 11, 10  | 5.0.3 |
| `pgnodemx` | 1.0.4 | 13, 12, 11, 10  | 5.0.0 |
| `set_user` | 3.0.0 | 14, 13, 12, 11, 10  | 5.0.3 |
| `set_user` | 2.0.1 | 13, 12, 11, 10  | 5.0.2 |
| `set_user` | 2.0.0 | 13, 12, 11, 10  | 5.0.0 |
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
