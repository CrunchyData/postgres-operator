<h1 align="center">Crunchy Data PostgreSQL Operator</h1>
<p align="center">
  <img width="150" src="./crunchy_logo.png?raw=true"/>
</p>


The *postgres-operator* is a controller that runs within a Kubernetes cluster that provides a means to deploy and manage PostgreSQL clusters.

Use the postgres-operator to:

 * deploy PostgreSQL containers including streaming replication clusters
 * scale up PostgreSQL clusters with extra replicas
 * add pgpool, pgbouncer, and metrics sidecars to PostgreSQL clusters
 * apply SQL policies to PostgreSQL clusters
 * assign metadata tags to PostgreSQL clusters
 * maintain PostgreSQL users and passwords
 * perform minor upgrades to PostgreSQL clusters
 * load simple CSV and JSON files into PostgreSQL clusters
 * perform database backups


## Deployment Requirements

The PostgreSQL Operator is validated for deployment on Kubernetes, OpenShift, and VMware Enterprise PKS clusters.  Some form of storage is required, NFS, hostPath, and Storage Classes are currently supported.

The PostgreSQL Operator includes various components that get deployed to your
Kubernetes cluster as shown in the following diagram and detailed
in the Design section of the documentation for the version you are running.

![Reference](https://access.crunchydata.com/documentation/postgres-operator/latest/Operator-Architecture.png)

The PostgreSQL Operator is developed and tested on CentOS and RHEL linux platforms but is known to run on other Linux variants.

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


### PGO CLI Installation

If you have the PostgreSQL Operator installed in your environment, and are interested in installation of the client interface, please start here:

- [PGO Client Install](https://access.crunchydata.com/documentation/postgres-operator/latest/installation/install-pgo-client/)


### Using the PostgreSQL Operator

If you have the PostgreSQL and Client Interface installed in your environment and are interested in guidance on the use of the Crunchy PostgreSQL Operator, please start here:

- [PGO CLI Overview](https://access.crunchydata.com/documentation/postgres-operator/latest/operatorcli/pgo-overview/)


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
version branch (e.g. `REL_4_1` for 4.1)
1. Please be as descriptive in your pull request as possible. If you are
referencing an issue, please be sure to include the issue in your pull request

## Submitting an Issue

Please use GitHub to submit an issue for the PostgreSQL Operator project.

If you would like to work the issue, please add that information in the issue so that we can confirm we are not already working no need to duplicate efforts.

If you have any question you can submit a Support - Question and Answer issue and we will work with you on how you can get more involved.


## Complete Documentation

For additional information regarding design, configuration and operation of the PostgreSQL Operator, pleases see the [Official Project Documentation](https://access.crunchydata.com/documentation/postgres-operator/latest/)

If you are looking for the latest documentation, please see the develop branch which is considered unstable. The development
documentation can be reviewed at https://crunchydata.github.io/postgres-operator/latest/.


## Past Versions

Documentation for previous releases can be found at the [Crunchy Data Access Portal](https://access.crunchydata.com/documentation)
