<h1 align="center">Crunchy Data PostgreSQL Operator</h1>
<p align="center">
  <img width="150" src="./crunchy_logo.png?raw=true"/>
</p>


# Run your own production-grade PostgreSQL-as-a-Service on Kubernetes!

The [Crunchy PostgreSQL Operator](https://access.crunchydata.com/documentation/postgres-operator/) automates and simplifies deploying and managing open source PostgreSQL clusters on Kubernetes and other Kubernetes-enabled Platforms by providing the essential features you need to keep your PostgreSQL clusters up and running, including:

#### PostgreSQL Cluster Provisioning

[Create, Scale, & Delete PostgreSQL clusters with ease](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/provisioning/), while fully customizing your Pods and PostgreSQL configuration!

#### High-Availability

Safe, automated failover backed by a [distributed consensus based high-availability solution](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/). Uses [Pod Anti-Affinity](https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity) to help resiliency; you can configure how aggressive this can be! Failed primaries automatically heal, allowing for faster recovery time.

#### Disaster Recovery

Backups and restores leverage the open source [pgBackRest](https://www.pgbackrest.org) utility and [includes support for full, incremental, and differential backups as well as efficient delta restores](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/). Set how long you want your backups retained for. Works great with very large databases!

#### Monitoring

Track the health of your PostgreSQL clusters using the open source [pgMonitor](https://github.com/CrunchyData/pgmonitor) library.

#### PostgreSQL User Management

Quickly add and remove users from your PostgreSQL clusters with powerful commands. Manage password expiration policies or use your preferred PostgreSQL authentication scheme.

#### Upgrade Management

Safely apply PostgreSQL updates with minimal availability impact to your PostgreSQL clusters.

#### Advanced Replication Support

Choose between [asynchronous replication](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/) and [synchronous replication](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/high-availability/#synchronous-replication-guarding-against-transactions-loss) for workloads that are sensitive to losing transactions.

#### Clone

Create new clusters from your existing clusters with a simple [`pgo clone`](https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/reference/pgo_clone/) command.

#### Connection Pooling

Use [pgBouncer](https://access.crunchydata.com/documentation/pgbouncer/) for connection pooling

#### Node Affinity

Have your PostgreSQL clusters deployed to [Kubernetes Nodes](https://kubernetes.io/docs/concepts/architecture/nodes/) of your preference

#### Scheduled Backups

Choose the type of backup (full, incremental, differential) and [how frequently you want it to occur](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/#scheduling-backups) on each PostgreSQL cluster.

#### Backup to S3

[Store your backups in Amazon S3](https://access.crunchydata.com/documentation/postgres-operator/latest/architecture/disaster-recovery/#using-s3) or any object storage system that supports the S3 protocol. The PostgreSQL Operator can backup, restore, and create new clusters from these backups.

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

## Deployment Requirements

The PostgreSQL Operator is validated for deployment on Kubernetes, OpenShift, and VMware Enterprise PKS clusters.  Some form of storage is required, NFS, hostPath, and Storage Classes are currently supported.

The PostgreSQL Operator includes various components that get deployed to your
Kubernetes cluster as shown in the following diagram and detailed
in the Design section of the documentation for the version you are running.

![Reference](https://access.crunchydata.com/documentation/postgres-operator/latest/Operator-Architecture.png)

The PostgreSQL Operator is developed and tested on CentOS and RHEL linux platforms but is known to run on other Linux variants.

### Supported Platforms

The Crunchy PostgreSQL Operator is tested on the following Platforms:

- Kubernetes 1.13+
- OpenShift 3.11+
- Google Kubernetes Engine (GKE), including Anthos
- VMware Enterprise PKS 1.3+

### Storage

The Crunchy PostgreSQL Operator is tested with a variety of different types of Kubernetes storage and Storage Classes, including:

- Google Compute Engine persistent volumes
- HostPath
- NFS
- Rook
- StorageOS

and more.

We know there are a variety of different types of [Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/) available for Kubernetes and we do our best to test each one, but due to the breadth of this area we are unable to verify PostgreSQL Operator functionality in each one. With that said, the PostgreSQL Operator is designed to be storage class agnostic and has been demonstrated to work with additional Storage Classes.

## Installation

### PostgreSQL Operator Installation

The PostgreSQL Operator provides a few different methods for installation.  

For an automated deployment using Ansible playbooks, please start here:

 - [Installation via Ansible](https://access.crunchydata.com/documentation/postgres-operator/latest/installation/install-with-ansible/)

 For a step by step customer installation using Bash, please start here:

 - [Installation via Bash](https://access.crunchydata.com/documentation/postgres-operator/latest/installation/operator-install/)

For a quick start deployment using OperatorHub.io, please see instructions here:

-  [OperatorHub.io Guidance](https://operatorhub.io/operator/postgresql)

For a quick start deployment to Google Kubernetes Engine (GKE), please see instructions here:

-  [GKE Quickstart Guidance](https://info.crunchydata.com/blog/install-postgres-operator-kubernetes-on-gke-ansible)


### `pgo` Client Installation

If you have the PostgreSQL Operator installed in your environment, and are interested in installation of the client interface, please start here:

- [pgo Client Install](https://access.crunchydata.com/documentation/postgres-operator/latest/installation/install-pgo-client/)

There is also a `pgo-client` container if you wish to deploy the client directly to your Kubernetes environment.

### Using the PostgreSQL Operator

If you have the PostgreSQL and Client Interface installed in your environment and are interested in guidance on the use of the Crunchy PostgreSQL Operator, please start here:

- [PostgreSQL Operator Documentation](https://access.crunchydata.com/documentation/postgres-operator/)
- [pgo Client User Guide](https://access.crunchydata.com/documentation/postgres-operator/latest/pgo-client/)

## Contributing to the Project

Want to contribute to the PostgreSQL Operator project? Great! We've put together
as set of contributing guidelines that you can review here:

- [Contributing Guidelines](CONTRIBUTING.md)

If you want to learn how to get up your development environment, please read our
documentation here:

 - [Developer Setup](https://access.crunchydata.com/documentation/postgres-operator/latest/installation/developer-setup/)

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

## Submitting an Issue

Please use GitHub to submit an issue for the PostgreSQL Operator project.

If you would like to work the issue, please add that information in the issue so that we can confirm we are not already working no need to duplicate efforts.

If you have any question you can submit a Support - Question and Answer issue and we will work with you on how you can get more involved.


## Complete Documentation

For additional information regarding design, configuration and operation of the
PostgreSQL Operator, pleases see the
[Official Project Documentation](https://access.crunchydata.com/documentation/postgres-operator/)

If you are looking for the [nightly builds of the documentation](https://crunchydata.github.io/postgres-operator/latest), you can view it at:

https://crunchydata.github.io/postgres-operator/latest

## Past Versions

Documentation for previous releases can be found at the [Crunchy Data Access Portal](https://access.crunchydata.com/documentation/)
