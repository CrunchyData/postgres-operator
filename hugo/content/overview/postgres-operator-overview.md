---
title: "PostgreSQL Operator Overview"
date:
draft: false
weight: 5
---

## PostgreSQL Operator Overview

The PostgreSQL Operator extends Kubernetes to provide a higher- level abstraction enabling the rapid creation and management of PostgreSQL databases and clusters.  

The PostgreSQL Operator include the following components:

* PostgreSQL Operator
* PostgreSQL Operator Containers
* PostgreSQL Operator PGO Client
* PostgreSQL Operator REST API Server
* PostgreSQL PGO Schedule

#### PostgreSQL Operator

The PostgreSQL Operator makes use of Kubernetes “Custom Resource Definitions” or “CRDs” to extend Kubernetes with custom, PostgreSQL specific, Kubernetes objects such as “Database” and “Cluster”.  The PostgreSQL Operator users these CRDs to enable users to deploy, configure and administer PostgreSQL databases and clusters as Kubernetes-natvie, open source PostgreSQL-as-a-Service infrastructure. 

#### PostgreSQL Operator Containers

The PostgreSQL Operator orchestrates a series of PostgreSQL and PostgreSQL related containers containers that enable rapid deployment of PostgreSQL, including administration and monitoring tools in a Kubernetes environment. 

#### PostgreSQL Operator PGO Client 

The PostgreSQL Operator provides a command line interface (CLI), pgo. This CLI tool may be used an end-user to create databases or clusters, or make changes to existing databases.  The CLI interacts with the REST API deployed within the postgres-operator deployment.

#### PostgreSQL Operator REST API Server

A feature of the PostgreSQL Operator is to provide a REST API upon which users or custom applications can inspect and cause actions within the Operator such as provisioning resources or viewing status of resources.  This API is secured by a RBAC (role based access control) security model whereby each API call has a permission assigned to it. API roles are defined to provide granular authorization to Operator services.

#### PostgreSQL Operator PGO Scheduler

The PostgreSQL Operator includes a cron like scheduler application called pgo-scheduler. The purpose of pgo-scheduler is to run automated tasks such as PostgreSQL backups or SQL policies against PostgreSQL clusters created by the PostgreSQL Operator.  PGO Scheduler watches Kubernetes for configmaps with the label crunchy-scheduler=true in the same namespace where the PostgreSQL Operator is deployed. 

