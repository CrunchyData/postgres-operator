<h1 align="center">PGO: The Postgres Operator from Crunchy Data</h1>
<p align="center">
  <img width="150" src="./docs/static/logos/pgo.svg" alt="PGO: The Postgres Operator from Crunchy Data"/>
</p>

[![Go Report Card](https://goreportcard.com/badge/github.com/CrunchyData/postgres-operator)](https://goreportcard.com/report/github.com/CrunchyData/postgres-operator)

# Production Postgres Made Easy

[PGO](https://github.com/CrunchyData/postgres-operator), the [Postgres Operator](https://github.com/CrunchyData/postgres-operator) from [Crunchy Data](https://www.crunchydata.com), gives you a **declarative Postgres** solution that automatically manages your [PostgreSQL](https://www.postgresql.org) clusters.

Designed for your GitOps workflows, it is [easy to get started](https://access.crunchydata.com/documentation/postgres-operator/v5/quickstart/) with Postgres on Kubernetes with PGO. Within a few moments, you can have a production-grade Postgres cluster complete with high availability, disaster recovery, and monitoring, all over secure TLS communications. Even better, PGO lets you easily customize your Postgres cluster to tailor it to your workload!

With conveniences like cloning Postgres clusters to using rolling updates to roll out disruptive changes with minimal downtime, PGO is ready to support your Postgres data at every stage of your release pipeline. Built for resiliency and uptime, PGO will keep your Postgres cluster in its desired state, so you do not need to worry about it.

PGO is developed with many years of production experience in automating Postgres management on Kubernetes, providing a seamless cloud native Postgres solution to keep your data always available.

# Installation

We recommend following our [Quickstart](https://access.crunchydata.com/documentation/postgres-operator/v5/quickstart/) for how to install and get up and running with PGO, the Postgres Operator from Crunchy Data. However, if you can't wait to try it out, here are some instructions to get Postgres up and running on Kubernetes:

1. [Fork the Postgres Operator examples repository](https://github.com/CrunchyData/postgres-operator-examples/fork) and clone it to your host machine. For example:

```sh
YOUR_GITHUB_UN="<your GitHub username>"
git clone --depth 1 "git@github.com:${YOUR_GITHUB_UN}/postgres-operator-examples.git"
cd postgres-operator-examples
```

2. Run `kubectl apply -k kustomize/install`

For more information please read the [Quickstart](https://access.crunchydata.com/documentation/postgres-operator/v5/quickstart/) and [Tutorial](https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/).

# Cloud Native Postgres for Kubernetes

PGO, the Postgres Operator from Crunchy Data, comes with all of the features you need for a complete cloud native Postgres experience on Kubernetes!

#### PostgreSQL Cluster [Provisioning][provisioning]

[Create, Scale, & Delete PostgreSQL clusters with ease][provisioning], while fully customizing your
Pods and PostgreSQL configuration!

#### [High Availability][high-availability]

Safe, automated failover backed by a [distributed consensus high availability solution][high-availability].
Uses [Pod Anti-Affinity][k8s-anti-affinity] to help resiliency; you can configure how aggressive this can be!
Failed primaries automatically heal, allowing for faster recovery time.

Support for [standby PostgreSQL clusters][multiple-cluster] that work both within and across [multiple Kubernetes clusters][multiple-cluster].

#### [Disaster Recovery][disaster-recovery]

[Backups][backups] and [restores][disaster-recovery] leverage the open source [pgBackRest][] utility and
[includes support for full, incremental, and differential backups as well as efficient delta restores][backups].
Set how long you to retain your backups. Works great with very large databases!

#### Security and [TLS][tls]

PGO enforces that all connections are over [TLS][tls]. You can also [bring your own TLS infrastructure][tls] if you do not want to use the defaults provided by PGO.

PGO runs containers with locked-down settings and provides Postgres credentials in a secure, convenient way for connecting your applications to your data.

#### [Monitoring][monitoring]

[Track the health of your PostgreSQL clusters][monitoring] using the open source [pgMonitor][] library.

#### [Upgrade Management][update-postgres]

Safely [apply PostgreSQL updates][update-postgres] with minimal impact to the availability of your PostgreSQL clusters.

#### Advanced Replication Support

Choose between [asynchronous][high-availability] and synchronous replication
for workloads that are sensitive to losing transactions.

#### [Clone][clone]

[Create new clusters from your existing clusters or backups][clone] with efficient data cloning.

#### [Connection Pooling][pool]

Advanced [connection pooling][pool] support using [pgBouncer][].

#### Pod Anti-Affinity, Node Affinity, Pod Tolerations

Have your PostgreSQL clusters deployed to [Kubernetes Nodes][k8s-nodes] of your preference. Set your [pod anti-affinity][k8s-anti-affinity], node affinity, Pod tolerations, and more rules to customize your deployment topology!

#### [Scheduled Backups][backup-management]

Choose the type of backup (full, incremental, differential) and [how frequently you want it to occur][backup-management] on each PostgreSQL cluster.

#### Backup to Local Storage, [S3][backups-s3], [GCS][backups-gcs], [Azure][backups-azure], or a Combo!

[Store your backups in Amazon S3][backups-s3] or any object storage system that supports
the S3 protocol. You can also store backups in [Google Cloud Storage][backups-gcs] and [Azure Blob Storage][backups-azure].

You can also [mix-and-match][backups-multi]: PGO lets you [store backups in multiple locations][backups-multi].

#### [Full Customizability][customize-cluster]

PGO makes it easy to fully customize your Postgres cluster to tailor to your workload:

- Choose the resources for your Postgres cluster: [container resources and storage size][resize-cluster]. [Resize at any time][resize-cluster] with minimal disruption.
- - Use your own container image repository, including support `imagePullSecrets` and private repositories
- [Customize your PostgreSQL configuration][customize-cluster]

#### [Namespaces][k8s-namespaces]

Deploy PGO to watch Postgres clusters in all of your [namespaces][k8s-namespaces], or [restrict which namespaces][single-namespace] you want PGO to manage Postgres clusters in!

[backups]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backups/
[backups-s3]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backups/#using-s3
[backups-gcs]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backups/#using-google-cloud-storage-gcs
[backups-azure]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backups/#using-azure-blob-storage
[backups-multi]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backups/#set-up-multiple-backup-repositories
[backup-management]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backup-management/
[clone]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/disaster-recovery/#clone-a-postgres-cluster
[customize-cluster]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/customize-cluster/
[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/disaster-recovery/
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/high-availability/
[monitoring]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/monitoring/
[multiple-cluster]: https://access.crunchydata.com/documentation/postgres-operator/v5/architecture/high-availability/multi-cluster-kubernetes/
[pool]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/connection-pooling/
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/create-cluster/
[resize-cluster]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/resize-cluster/
[single-namespace]: https://access.crunchydata.com/documentation/postgres-operator/v5/installation/kustomize/#installation-mode
[tls]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/customize-cluster/#customize-tls
[update-postgres]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/update-cluster/
[k8s-anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity
[k8s-namespaces]: https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/
[k8s-nodes]: https://kubernetes.io/docs/concepts/architecture/nodes/

[pgBackRest]: https://www.pgbackrest.org
[pgBouncer]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/connection-pooling/
[pgMonitor]: https://github.com/CrunchyData/pgmonitor

## Included Components

[PostgreSQL containers](https://github.com/CrunchyData/crunchy-containers) deployed with the PostgreSQL Operator include the following components:

- [PostgreSQL](https://www.postgresql.org)
  - [PostgreSQL Contrib Modules](https://www.postgresql.org/docs/current/contrib.html)
  - [PL/Python + PL/Python 3](https://www.postgresql.org/docs/current/plpython.html)
  - [PL/Perl](https://www.postgresql.org/docs/current/plperl.html)
  - [PL/Tcl](https://www.postgresql.org/docs/current/pltcl.html)
  - [pgAudit](https://www.pgaudit.org/)
  - [pgAudit Analyze](https://github.com/pgaudit/pgaudit_analyze)
  - [pg_cron](https://github.com/citusdata/pg_cron)
  - [pg_partman](https://github.com/pgpartman/pg_partman)
  - [pgnodemx](https://github.com/CrunchyData/pgnodemx)
  - [set_user](https://github.com/pgaudit/set_user)
  - [TimescaleDB](https://github.com/timescale/timescaledb) (Apache-licensed community edition)
  - [wal2json](https://github.com/eulerto/wal2json)
- [pgBackRest](https://pgbackrest.org/)
- [pgBouncer](http://pgbouncer.github.io/)
- [pgAdmin 4](https://www.pgadmin.org/)
- [pgMonitor](https://github.com/CrunchyData/pgmonitor)
- [Patroni](https://patroni.readthedocs.io/)
- [LLVM](https://llvm.org/) (for [JIT compilation](https://www.postgresql.org/docs/current/jit.html))

In addition to the above, the geospatially enhanced PostgreSQL + PostGIS container adds the following components:

- [PostGIS](http://postgis.net/)
- [pgRouting](https://pgrouting.org/)

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

- Kubernetes 1.20+
- OpenShift 4.6+
- Rancher
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

This list only includes the platforms that the Postgres Operator is specifically
tested on as part of the release process: PGO works on other Kubernetes
distributions as well.

# Contributing to the Project

Want to contribute to the PostgreSQL Operator project? Great! We've put together
a set of contributing guidelines that you can review here:

- [Contributing Guidelines](CONTRIBUTING.md)

Once you are ready to submit a Pull Request, please ensure you do the following:

1. Reviewing the [contributing guidelines](CONTRIBUTING.md) and ensure your
that you have followed the commit message format, added testing where
appropriate, documented your changes, etc.
1. Open up a pull request based upon the guidelines. If you are adding a new
feature, please open up the pull request on the `master` branch.
1. Please be as descriptive in your pull request as possible. If you are
referencing an issue, please be sure to include the issue in your pull request

## Support

If you believe you have found a bug or have a detailed feature request, please open a GitHub issue and follow the guidelines for submitting a bug.

For general questions or community support, we welcome you to [join the PGO project community mailing list](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join) at [https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join) and ask your question there.

For other information, please visit the [Support](https://access.crunchydata.com/documentation/postgres-operator/latest/support/) section of the documentation.

# Documentation

For additional information regarding the design, configuration, and operation of the
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

To stay up-to-date on when releases are made available in the [Crunchy Data Developer Portal](https://www.crunchydata.com/developers), please sign up for the [Crunchy Data Developer Program Newsletter](https://www.crunchydata.com/developers/newsletter). You can also [join the PGO project community mailing list](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join)

The PGO Postgres Operator project source code is available subject to the [Apache 2.0 license](LICENSE.md) with the PGO logo and branding assets covered by [our trademark guidelines](docs/static/logos/TRADEMARKS.md).
