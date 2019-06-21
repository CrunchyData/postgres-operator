---
title: "Crunchy Data Postgres Operator"
date:
draft: false
---

 <img width="25%" src="crunchy_logo.png"/>

Latest Release: 4.0.1

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

The Operator is validated for deployment on Kubernetes, OpenShift, and VMware Enterprise PKS clusters.  Some form of storage is required, NFS, HostPath, and Storage Classes are currently supported.

The Operator includes various components that get deployed to your
Kubernetes cluster as shown in the following diagram and detailed
in the [Design](/design). 

![Architecture](/Operator-Architecture.png)

The Operator is developed and tested on CentOS and RHEL Linux platforms but is known to run on other Linux variants.

## Documentation
The following documentation is provided:

 - [pgo CLI Syntax and Examples](/operator-cli)
 - [Installation](/installation)
 - [Configuration](/configuration) 
 - [pgo.yaml Configuration](/configuration/pgo-yaml-configuration) 
 - [Security](/security) 
 - [Design Overview](/design) 
 - [Developing](/developer-setup) 
 - [Upgrading the Operator](/upgrade)
 - [Contributing](/contributing/documentation-updates)

