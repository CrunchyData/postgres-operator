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

The Operator is validated for deployment on Kubernetes, OpenShift, and VMware Enterprise PKS clusters.  Some form of storage is required, NFS, hostPath, and Storage Classes are currently supported.

The Operator includes various components that get deployed to your
Kubernetes cluster as shown in the following diagram and detailed
in the Design section of the documentation for the version you are running.

![Reference](https://access.crunchydata.com/documentation/postgres-operator/4.0.0/Operator-Architecture.png)

The Operator is developed and tested on CentOS and RHEL linux platforms but is known to run on other Linux variants.

## Documentation 4.0.0

 - [Getting Started](https://access.crunchydata.com/documentation/postgres-operator/4.0.0/gettingstarted/)
 - [Installation via Bash](https://access.crunchydata.com/documentation/postgres-operator/4.0.0/installation/operator-install/)
 - [Installation via Ansible](https://access.crunchydata.com/documentation/postgres-operator/4.0.0/installation/install-with-ansible/)
 - [PGO CLI Overview](https://access.crunchydata.com/documentation/postgres-operator/4.0.0/operatorcli/pgo-overview/)
 - [Contributing](https://access.crunchydata.com/documentation/postgres-operator/4.0.0/installation/developer-setup/)

## Documentation 3.5.3

 - [Installation](https://access.crunchydata.com/documentation/postgres-operator/3.5.3/installation/)
 - [PGO CLI Overview](https://access.crunchydata.com/documentation/postgres-operator/3.5.3/operator-cli/)
 - [Contributing](https://access.crunchydata.com/documentation/postgres-operator/3.5.3/contributing/)

Documentation for previous releases can be found at the [Crunchy Data Access Portal](https://access.crunchydata.com/documentation)


If you are looking for the latest documentation, please see the develop branch which is considered unstable. The development
documentation can be reviewed at https://crunchydata.github.io/postgres-operator/latest/.
