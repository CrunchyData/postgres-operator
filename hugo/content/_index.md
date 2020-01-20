---
title: "Crunchy PostgreSQL Operator"
date:
draft: false
---

# Crunchy PostgreSQL Operator

 <img width="25%" src="crunchy_logo.png"/>

## Run your own production-grade PostgreSQL-as-a-Service on Kubernetes!

Latest Release: 4.3.0

The Crunchy PostgreSQL Operator automates and simplifies deploying and managing open source PostgreSQL clusters on Kubernetes and other Kubernetes-enabled Platforms by providing the essential features you need to keep your PostgreSQL clusters up and running, including:

#### PostgreSQL Cluster Provisioning

[Create, Scale, & Delete PostgreSQL clusters with ease](/architecture/provisioning/), while fully customizing your Pods and PostgreSQL configuration!

#### High-Availability

Safe, automated failover backed by a [distributed consensus based high-availability solution](/architecture/high-availability/). Uses [Pod Anti-Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity) to help resiliency; you can configure how aggressive this can be! Failed primaries automatically heal, allowing for faster recovery time.

#### Disaster Recovery

Backups and restores leverage the open source [pgBackRest](https://www.pgbackrest.org) utility and [includes support for full, incremental, and differential backups as well as efficient delta restores](/architecture/disaster-recovery/). Set how long you want your backups retained for. Works great with very large databases!

#### Monitoring

Track the health of your PostgreSQL clusters using the open source [pgMonitor](https://github.com/CrunchyData/pgmonitor) library.

#### PostgreSQL User Management

Quickly add and remove users from your PostgreSQL clusters with powerful commands. Manage password expiration policies or use your preferred PostgreSQL authentication scheme.

#### Upgrade Management

Safely apply PostgreSQL updates with minimal availability impact to your PostgreSQL clusters.

#### Advanced Replication Support

Choose between [asynchronous replication](/architecture/high-availability/) and [synchronous replication](/architecture/high-availability/#synchronous-replication-guarding-against-transactions-loss) for workloads that are sensitive to losing transactions.

#### Clone

Create new clusters from your existing clusters with a simple [`pgo clone`](/pgo-client/reference/pgo_clone/) command.

#### Connection Pooling

 Use [pgBouncer](https://access.crunchydata.com/documentation/pgbouncer/) for connection pooling

#### Node Affinity

Have your PostgreSQL clusters deployed to [Kubernetes Nodes](https://kubernetes.io/docs/concepts/architecture/nodes/) of your preference

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
- Bring your own trusted certificate authority (CA) for use with the Operator API server
- Override your PostgreSQL configuration for each cluster

# How it Works

![Architecture](/Operator-Architecture.png)

The Crunchy PostgreSQL Operator extends Kubernetes to provide a higher-level abstraction for rapid creation and management of PostgreSQL clusters.  The Crunchy PostgreSQL Operator leverages a Kubernetes concept referred to as "[Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)” to create several [custom resource definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions) that allow for the management of PostgreSQL clusters.


# Supported Platforms

The Crunchy PostgreSQL Operator is tested on the following Platforms:

- Kubernetes 1.13 - 1.15 (See note about 1.16 and beyond)
- OpenShift 3.11+
- Google Kubernetes Engine (GKE), including Anthos
- VMware Enterprise PKS 1.3+

**NOTE**: At present, while the Crunchy PostgreSQL Operator has compatibility for Kubernetes 1.16 and beyond, it has not been verified for the v4.3.0 release.

## Storage

The Crunchy PostgreSQL Operator is tested with a variety of different types of Kubernetes storage and Storage Classes, including:

- Google Compute Engine persistent volumes
- HostPath
- NFS
- Rook
- StorageOS

and more.

We know there are a variety of different types of [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/) available for Kubernetes and we do our best to test each one, but due to the breadth of this area we are unable to verify PostgreSQL Operator functionality in each one. With that said, the PostgreSQL Operator is designed to be storage class agnostic and has been demonstrated to work with additional Storage Classes.
