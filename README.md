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

The Operator deploys on Kubernetes and Openshift clusters.  Some form of storage is required, NFS, hostPath, and Storage Classes are currently supported.

The Operator includes various components that get deployed to your
Kubernetes cluster as shown in the following diagram and detailed
in the [Design](https://crunchydata.github.io/postgres-operator/stable/design/).

![Reference](https://crunchydata.github.io/postgres-operator/stable/Operator-Architecture.png)

The Operator is developed and tested on CentOS and RHEL linux platforms but is known to run on other Linux variants.

## Documentation

 - [Getting Started](https://crunchydata.github.io/postgres-operator/stable/gettingstarted/)
 - [pgo CLI Syntax and Examples](https://crunchydata.github.io/postgres-operator/stable/operator-cli/)
 - [Installation](https://crunchydata.github.io/postgres-operator/stable/installation/)
 - [Configuration](https://crunchydata.github.io/postgres-operator/stable/configuration/configuration/)
 - [pgo.yaml Description](https://crunchydata.github.io/postgres-operator/stable/configuration/pgo-yaml-configuration/)
 - [Security](https://crunchydata.github.io/postgres-operator/stable/security/)
 - [Design](https://crunchydata.github.io/postgres-operator/stable/design/)
 - [Developing](https://crunchydata.github.io/postgres-operator/stable/developer-setup/)
 - [Upgrading the Operator](https://crunchydata.github.io/postgres-operator/stable/upgrade/)


If you are looking for the latest documentation, please see the develop branch which is considered unstable. The development
documentation can be reviewed at https://crunchydata.github.io/postgres-operator/latest/.