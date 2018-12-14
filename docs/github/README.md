<h1 align="center">Crunchy Data PostgreSQL Operator</h1>
<p align="center">
  <img width="150" src="crunchy_logo.png?raw=true"/>
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

The Operator is developed and tested on CentOS and RHEL linux platforms but is known to run on other Linux variants.

## Documentation
The following documentation is provided:

 - [pgo CLI Syntax and Examples](pgo-cli) 
 - [Installation](installation)
 - [Configuration](configuration) 
 - [pgo.yaml Description](pgo-yaml-configuration) 
 - [Security](security) 
 - [Design Overview](design) 
 - [Developing](developing) 
 - [Upgrading the Operator](upgrading)

<!--stackedit_data:
eyJoaXN0b3J5IjpbLTEyNTIzNzQ4NjksMjAwMTM0ODg5MSwyOD
g2NTg1NjUsLTIxMTAwMjE5NzgsLTIxMzg3NzMwNDgsOTY5Nzky
OTgwLDc3NDMwMzk4OCwxNTI5NDA0MzcxLDgxMTA4NTg0MV19
-->
