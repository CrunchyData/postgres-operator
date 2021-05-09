<h1 align="center">PGO: The Postgres Operator from Crunchy Data</h1>
<p align="center">
  <img width="150" src="./docs/static/logos/pgo.svg" alt="PGO: The Postgres Operator from Crunchy Data"/>
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/CrunchyData/postgres-operator)](https://goreportcard.com/report/github.com/CrunchyData/postgres-operator)

# Production Postgres Made Easy

[PGO](https://github.com/CrunchyData/postgres-operator), the [Postgres Operator](https://github.com/CrunchyData/postgres-operator) from [Crunchy Data](https://www.crunchydata.com), gives you a **declarative Postgres** solution that automatically manages your [PostgreSQL](https://www.postgresql.org) clusters.

Designed for your GitOps workflows, it is [easy to get started](https://access.crunchydata.com/documentation/postgres-operator/latest/quickstart/) with Postgres on Kubernetes with PGO. Within a few moments, you can have a production grade Postgres cluster complete with high availability, disaster recovery, and monitoring, all over secure TLS communications. Even better, PGO lets you easily customize your Postgres cluster to tailor it to your workload!

With conveniences like cloning Postgres clusters to using rolling updates to roll out disruptive changes with minimal downtime, PGO is ready to support your Postgres data at every stage of your release pipeline. Built for resiliency and uptime, PGO will keep your desired Postgres in a desired state so you do not need to worry about it.

PGO is developed with over six years of production experience in automating Postgres management on Kubernetes, providing a seamless cloud native Postgres solution to keep your data always available.

# Installation

We recommend following our [Quickstart](https://access.crunchydata.com/documentation/postgres-operator/latest/quickstart/) for how to install and get up and running with PGO, the Postgres Operator from Crunchy Data. However, if you just can't wait to try it out, here are some instructions to get Postgres up and running on Kubernetes:

1. [Fork the Postgres Operator examples repository](https://github.com/CrunchyData/postgres-operator-examples/fork) and clone it to your host machine.
1. Go into your directory and run `kubectl apply -k kustomize/install`

For more information please read the [Quickstart](https://access.crunchydata.com/documentation/postgres-operator/latest/quickstart/) and [Tutorial](https://access.crunchydata.com/documentation/postgres-operator/latest/tutorial/).

# Cloud Native Postgres for Kubernetes

PGO, the Postgres Operator from Crunchy Data, comes with all of the features you need for a complete cloud native Postgres experience!

#### PostgreSQL Cluster [Provisioning][provisioning]

[Create, Scale, & Delete PostgreSQL clusters with ease][provisioning], while fully customizing your
Pods and PostgreSQL configuration!

#### [High Availability][high-availability]

Safe, automated failover backed by a [distributed consensus based high-availability solution][high-availability].
Uses [Pod Anti-Affinity][k8s-anti-affinity] to help resiliency; you can configure how aggressive this can be!
Failed primaries automatically heal, allowing for faster recovery time.

Support for [standby PostgreSQL clusters][multiple-cluster] that work both within and across [multiple Kubernetes clusters][multiple-cluster].

#### [Disaster Recovery][disaster-recovery]

Backups and restores leverage the open source [pgBackRest][] utility and
[includes support for full, incremental, and differential backups as well as efficient delta restores][disaster-recovery].
Set how long you want your backups retained for. Works great with very large databases!

#### Security and TLS

PGO enforces that all connections are over TLS. You can also bring your own TLS infrastructure if you do not want to use the defaults provided by PGO.

PGO runs containers with locked-down settings and provides Postgres credentials in a secure, convenient way for connecting your applications to your data.

#### [Monitoring][monitoring]

[Track the health of your PostgreSQL clusters][monitoring] using the open source [pgMonitor][] library.

#### PostgreSQL User Management

Quickly add and remove users from your PostgreSQL clusters with powerful commands. Manage password
expiration policies or use your preferred PostgreSQL authentication scheme.

#### Upgrade Management

Safely apply PostgreSQL updates with minimal availability impact to your PostgreSQL clusters.

#### Advanced Replication Support

Choose between [asynchronous replication][high-availability] and [synchronous replication][high-availability-sync]
for workloads that are sensitive to losing transactions.

#### Clone

Create new clusters from your existing clusters or backups with [`pgo create cluster --restore-from`][pgo-create-cluster].

#### Connection Pooling

Use [pgBouncer][] for connection pooling

#### Node Affinity

Have your PostgreSQL clusters deployed to [Kubernetes Nodes][k8s-nodes] of your preference

#### Scheduled Backups

Choose the type of backup (full, incremental, differential) and [how frequently you want it to occur][disaster-recovery-scheduling] on each PostgreSQL cluster.

#### Backup to S3

[Store your backups in Amazon S3][disaster-recovery-s3] or any object storage system that supports
the S3 protocol. The PostgreSQL Operator can backup, restore, and create new clusters from these backups.


#### Full Customizability

The Crunchy PostgreSQL Operator makes it easy to get your own PostgreSQL-as-a-Service up and running on Kubernetes-enabled platforms, but we know that there are further customizations that you can make. As such, the Crunchy PostgreSQL Operator allows you to further customize your deployments, including:

- Selecting different storage classes for your primary, replica, and backup storage
- Select your own container resources class for each PostgreSQL cluster deployment; differentiate between resources applied for primary and replica clusters!
- Use your own container image repository, including support `imagePullSecrets` and private repositories
- [Customize your PostgreSQL configuration](https://access.crunchydata.com/documentation/postgres-operator/latest/advanced/custom-configuration/)
- Bring your own trusted certificate authority (CA) for use with the Operator API server
- Override your PostgreSQL configuration for each cluster


[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/
[disaster-recovery-s3]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/#using-s3
[disaster-recovery-scheduling]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/#scheduling-backups
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/
[high-availability-sync]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/#synchronous-replication-guarding-against-transactions-loss
[monitoring]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/monitoring/
[multiple-cluster]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/multi-cluster-kubernetes/
[pgo-create-cluster]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/reference/pgo_create_cluster/
[pgo-task-tls]: https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/common-tasks/#enable-tls
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/provisioning/

[k8s-anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity
[k8s-namespaces]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
[k8s-nodes]: https://kubernetes.io/docs/concepts/architecture/nodes/

[pgBackRest]: https://www.pgbackrest.org
[pgBouncer]: https://access.crunchydata.com/documentation/pgbouncer/
[pgMonitor]: https://github.com/CrunchyData/pgmonitor

## Included Components

[PostgreSQL containers](https://github.com/CrunchyData/crunchy-containers) deployed with the PostgreSQL Operator include the following components:

- [PostgreSQL](https://www.postgresql.org)
  - [PostgreSQL Contrib Modules](https://www.postgresql.org/docs/current/contrib.html)
  - [PL/Python + PL/Python 3](https://www.postgresql.org/docs/current/plpython.html)
  - [pgAudit](https://www.pgaudit.org/)
  - [pgAudit Analyze](https://github.com/pgaudit/pgaudit_analyze)
  - [pg_partman](https://github.com/pgpartman/pg_partman)
  - [pgnodemx](https://github.com/CrunchyData/pgnodemx)
  - [set_user](https://github.com/pgaudit/set_user)
  - [wal2json](https://github.com/eulerto/wal2json)
- [pgBackRest](https://pgbackrest.org/)
- [pgBouncer](https://www.pgbouncer.org/)
- [pgAdmin 4](https://www.pgadmin.org/)
- [pgMonitor](https://github.com/CrunchyData/pgmonitor)
- [Patroni](https://patroni.readthedocs.io/)
- [LLVM](https://llvm.org/) (for [JIT compilation](https://www.postgresql.org/docs/current/jit.html))

In addition to the above, the geospatially enhanced PostgreSQL + PostGIS container adds the following components:

- [PostGIS](http://postgis.net/)
- [pgRouting](https://pgrouting.org/)
- [PL/R](https://github.com/postgres-plr/plr)

[PostgreSQL Operator Monitoring](https://crunchydata.github.io/postgres-operator/latest/architecture/monitoring/) uses the following components:

- [pgMonitor](https://github.com/CrunchyData/pgmonitor)
- [Prometheus](https://github.com/prometheus/prometheus)
- [Grafana](https://github.com/grafana/grafana)
- [Alertmanager](https://github.com/prometheus/alertmanager)

Additional containers that are not directly integrated with the PostgreSQL Operator but can work alongside it include:

- [pgPool II](https://access.crunchydata.com/documentation/crunchy-postgres-containers/latest/container-specifications/crunchy-pgpool/)
- [pg_upgrade](https://access.crunchydata.com/documentation/crunchy-postgres-containers/latest/container-specifications/crunchy-upgrade/)
- [pgBench](https://access.crunchydata.com/documentation/crunchy-postgres-containers/latest/container-specifications/crunchy-pgbench/)

For more information about which versions of the PostgreSQL Operator include which components, please visit the [compatibility](https://access.crunchydata.com/documentation/postgres-operator/latest/configuration/compatibility/) section of the documentation.

## Supported Platforms

PGO, the Postgres Operator from Crunchy Data, is tested on the following platforms:

- Kubernetes 1.18+
- OpenShift 4.4+
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

This list only includes the platforms that the Postgres Operator is specifically
tested on as part of the release process: PGO works on other Kubernetes
distributions as well, such as Rancher.

# Contributing to the Project

Want to contribute to the PostgreSQL Operator project? Great! We've put together
as set of contributing guidelines that you can review here:

- [Contributing Guidelines](CONTRIBUTING.md)

If you want to learn how to get up your development environment, please read our
documentation here:

 - [Developer Setup](https://access.crunchydata.com/documentation/postgres-operator/latest/contributing/developer-setup/)

Once you are ready to submit a Pull Request, please ensure you do the following:

1. Reviewing the [contributing guidelines](CONTRIBUTING.md) and ensure your
that you have followed the commit message format, added testing where
appropriate, documented your changes, etc.
1. Open up a pull request based upon the guidelines. If you are adding a new
feature, please open up the pull request on the `master` branch. If you have
a bug fix for a supported version, open up a pull request against the supported
version branch (e.g. `REL_4_2` for 4.2)
1. Please be as descriptive in your pull request as possible. If you are
referencing an issue, please be sure to include the issue in your pull request

# Support

If you believe you have found a bug or have detailed feature request, please open a GitHub issue and follow the guidelines for submitting a bug.

For general questions or community support, we welcome you to [join the PostgreSQL Operator community mailing list](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join) at [https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join) and ask your question there.

For other information, please visit the [Support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/) section of the documentation.

# Documentation

For additional information regarding design, configuration and operation of the
PostgreSQL Operator, pleases see the [Official Project Documentation][documentation].

If you are looking for the [nightly builds of the documentation](https://crunchydata.github.io/postgres-operator/latest/), you can view them at:

https://crunchydata.github.io/postgres-operator/latest/

[documentation]: https://access.crunchydata.com/documentation/postgres-operator/

## Past Versions

Documentation for previous releases can be found at the [Crunchy Data Access Portal](https://access.crunchydata.com/documentation/)

# Releases

When a PostgreSQL Operator general availability (GA) release occurs, the container images are distributed on the following platforms in order:

- [Crunchy Data Customer Portal](https://access.crunchydata.com/)
- [Crunchy Data Developer Portal](https://www.crunchydata.com/developers)
- [DockerHub](https://hub.docker.com/u/crunchydata)

The image rollout can occur over the course of several days.

To stay up-to-date on when releases are made available in the [Crunchy Data Developer Portal](https://www.crunchydata.com/developers), please sign up for the [Crunchy Data Developer Program Newsletter](https://www.crunchydata.com/developers/newsletter)

The PGO Postgres Operator project source code is available subject to the [Apache 2.0 license](LICENSE.md) with the PGO logo and branding assets covered by [our trademark guidelines](docs/static/logos/TRADEMARKS.md).
