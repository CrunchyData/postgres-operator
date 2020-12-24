---
title: "Crunchy PostgreSQL Operator"
date:
draft: false
---

# Crunchy PostgreSQL Operator

 <img width="25%" src="crunchy_logo.png"/>

## Run your own production-grade PostgreSQL-as-a-Service on Kubernetes!

Latest Release: {{< param operatorVersion >}}

The [Crunchy PostgreSQL Operator](https://www.crunchydata.com/developers/download-postgres/containers/postgres-operator) automates and simplifies deploying and managing open source PostgreSQL clusters on Kubernetes and other Kubernetes-enabled Platforms by providing the essential features you need to keep your PostgreSQL clusters up and running, including:

#### PostgreSQL Cluster [Provisioning]({{< relref "/architecture/provisioning.md" >}})

[Create, Scale, & Delete PostgreSQL clusters with ease](/architecture/provisioning/), while fully customizing your Pods and PostgreSQL configuration!

#### [High Availability]({{< relref "/architecture/high-availability/_index.md" >}})

Safe, automated failover backed by a [distributed consensus based high-availability solution](/architecture/high-availability/). Uses [Pod Anti-Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity) to help resiliency; you can configure how aggressive this can be! Failed primaries automatically heal, allowing for faster recovery time.

Support for [standby PostgreSQL clusters]({{< relref "/architecture/high-availability/multi-cluster-kubernetes.md" >}}) that work both within an across [multiple Kubernetes clusters]({{< relref "/architecture/high-availability/multi-cluster-kubernetes.md" >}}).

#### [Disaster Recovery]({{< relref "/architecture/disaster-recovery.md" >}})

Backups and restores leverage the open source [pgBackRest](https://www.pgbackrest.org) utility and [includes support for full, incremental, and differential backups as well as efficient delta restores](/architecture/disaster-recovery/). Set how long you want your backups retained for. Works great with very large databases!

#### TLS

Secure communication between your applications and data servers by [enabling TLS for your PostgreSQL servers](/pgo-client/common-tasks/#enable-tls), including the ability to enforce that all of your connections to use TLS.

#### [Monitoring]({{< relref "/architecture/monitoring.md" >}})

[Track the health of your PostgreSQL clusters]({{< relref "/architecture/monitoring.md" >}})
using the open source [pgMonitor](https://github.com/CrunchyData/pgmonitor)
library.

#### PostgreSQL User Management

Quickly add and remove users from your PostgreSQL clusters with powerful commands. Manage password expiration policies or use your preferred PostgreSQL authentication scheme.

#### Upgrade Management

Safely apply PostgreSQL updates with minimal availability impact to your PostgreSQL clusters.

#### Advanced Replication Support

Choose between [asynchronous replication](/architecture/high-availability/) and [synchronous replication](/architecture/high-availability/#synchronous-replication-guarding-against-transactions-loss) for workloads that are sensitive to losing transactions.

#### Clone

Create new clusters from your existing clusters or backups with [`pgo create cluster --restore-from`](/pgo-client/reference/pgo_create_cluster/).

#### Connection Pooling

 Use [pgBouncer]({{< relref "tutorial/pgbouncer.md" >}}) for connection pooling.

#### Affinity and Tolerations

Have your PostgreSQL clusters deployed to [Kubernetes Nodes](https://kubernetes.io/docs/concepts/architecture/nodes/) of your preference with [node affinity]({{< relref "architecture/high-availability/_index.md">}}#node-affinity), or designate which nodes Kubernetes can schedule PostgreSQL instances to with [tolerations]({{< relref "architecture/high-availability/_index.md">}}#tolerations).

#### Scheduled Backups

Choose the type of backup (full, incremental, differential) and [how frequently you want it to occur](/architecture/disaster-recovery/#scheduling-backups) on each PostgreSQL cluster.

#### Backup to S3

[Store your backups in Amazon S3](/architecture/disaster-recovery/#using-s3) or any object storage system that supports the S3 protocol. The PostgreSQL Operator can backup, restore, and create new clusters from these backups.

#### Multi-Namespace Support

You can control how the PostgreSQL Operator leverages [Kubernetes Namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/) with several different deployment models:

- Deploy the PostgreSQL Operator and all PostgreSQL clusters to the same namespace
- Deploy the PostgreSQL Operator to one namespaces, and all PostgreSQL clusters to a different namespace
- Deploy the PostgreSQL Operator to one namespace, and have your PostgreSQL clusters managed acrossed multiple namespaces
- Dynamically add and remove namespaces managed by the PostgreSQL Operator using the `pgo create namespace` and `pgo delete namespace` commands

#### Full Customizability

The Crunchy PostgreSQL Operator makes it easy to get your own PostgreSQL-as-a-Service up and running on Kubernetes-enabled platforms, but we know that there are further customizations that you can make. As such, the Crunchy PostgreSQL Operator allows you to further customize your deployments, including:

- Selecting different storage classes for your primary, replica, and backup storage
- Select your own container resources class for each PostgreSQL cluster deployment; differentiate between resources applied for primary and replica clusters!
- Use your own container image repository, including support `imagePullSecrets` and private repositories
- [Customize your PostgreSQL configuration]({{< relref "/advanced/custom-configuration.md" >}})
- Bring your own trusted certificate authority (CA) for use with the Operator API server
- Override your PostgreSQL configuration for each cluster

# How it Works

![Architecture](/Operator-Architecture.png)

The Crunchy PostgreSQL Operator extends Kubernetes to provide a higher-level abstraction for rapid creation and management of PostgreSQL clusters.  The Crunchy PostgreSQL Operator leverages a Kubernetes concept referred to as "[Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)” to create several [custom resource definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) that allow for the management of PostgreSQL clusters.


# Included Components

[PostgreSQL containers](https://github.com/CrunchyData/crunchy-containers) deployed with the PostgreSQL Operator include the following components:

- [PostgreSQL](https://www.postgresql.org)
  - [PostgreSQL Contrib Modules](https://www.postgresql.org/docs/current/contrib.html)
  - [PL/Python + PL/Python 3](https://www.postgresql.org/docs/current/plpython.html)
  - [PL/Perl](https://www.postgresql.org/docs/current/plperl.html)
  - [pgAudit](https://www.pgaudit.org/)
  - [pgAudit Analyze](https://github.com/pgaudit/pgaudit_analyze)
  - [pgnodemx](https://github.com/CrunchyData/pgnodemx)
  - [set_user](https://github.com/pgaudit/set_user)
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
- [PL/R](https://github.com/postgres-plr/plr)

[PostgreSQL Operator Monitoring]({{< relref "architecture/monitoring/_index.md" >}}) uses the following components:

- [pgMonitor](https://github.com/CrunchyData/pgmonitor)
- [Prometheus](https://github.com/prometheus/prometheus)
- [Grafana](https://github.com/grafana/grafana)
- [Alertmanager](https://github.com/prometheus/alertmanager)

Additional containers that are not directly integrated with the PostgreSQL Operator but can work alongside it include:

- [pgPool II](https://access.crunchydata.com/documentation/crunchy-postgres-containers/latest/container-specifications/crunchy-pgpool/)
- [pg_upgrade](https://access.crunchydata.com/documentation/crunchy-postgres-containers/latest/container-specifications/crunchy-upgrade/)
- [pgBench](https://access.crunchydata.com/documentation/crunchy-postgres-containers/latest/container-specifications/crunchy-pgbench/)

For more information about which versions of the PostgreSQL Operator include which components, please visit the [compatibility]({{< relref "configuration/compatibility.md" >}}) section of the documentation.

# Supported Platforms

The Crunchy PostgreSQL Operator is tested on the following Platforms:

- Kubernetes 1.13+
- OpenShift 3.11+
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- VMware Enterprise PKS 1.3+

## Storage

The Crunchy PostgreSQL Operator is tested with a variety of different types of Kubernetes storage and Storage Classes, including:

- Rook
- StorageOS
- Google Compute Engine persistent volumes
- NFS
- HostPath

and more. We have had reports of people using the PostgreSQL Operator with other [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/) as well.

We know there are a variety of different types of [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/) available for Kubernetes and we do our best to test each one, but due to the breadth of this area we are unable to verify PostgreSQL Operator functionality in each one. With that said, the PostgreSQL Operator is designed to be storage class agnostic and has been demonstrated to work with additional Storage Classes. Storage is a rapidly evolving field in Kubernetes and we will continue to adapt the PostgreSQL Operator to modern Kubernetes storage standards.
