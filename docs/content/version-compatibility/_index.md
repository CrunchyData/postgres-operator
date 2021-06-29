---
title: "Version Compatibility"
date:
draft: false
weight: 100
---

## Kubernetes Compatibility

PGO, the Postgres Operator from Crunchy Data, is tested on the following platforms:

- Kubernetes 1.18+
- OpenShift 4.5+
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

## Component Compatibility

The following table defines the comptability between PGO and the various component containers 
needed to deploy PostgreSQL clusters using PGO.

| Component | Version | PGO Version Min. | PGO Version Max. |
|-----------|---------|------------------|------------------|
| `crunchy-pgbackrest` | 2.33 | v5.0.0 | v5.0.0 |
| `crunchy-pgbouncer` | 1.15 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-ha` | 13.3 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-ha` | 12.7 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-ha` | 11.12 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-ha` | 10.17 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 13.3-3.1 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 13.3-3.0 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 12.7-3.0 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 12.7-2.5 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 11.12-2.5 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 11.12-2.4 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 10.17-2.4 | v5.0.0 | v5.0.0 |
| `crunchy-postgres-gis-ha` | 10.17-2.3 | v5.0.0 | v5.0.0 |
