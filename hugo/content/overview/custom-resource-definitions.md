---
title: "Custom Resource Definitions Overview"
date:
draft: false
weight: 5
---

## PostgreSQL Operator Custom Resource Definitions  

The PostgreSQL Operator defines the following series of Kubernetes [Custom Resource Definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#custom-resources):

![Reference](/operator-crd-architecture.png)

Each of these CRDs are used in the design of the PostgreSQL Operator to perform PostgreSQL related operations in order to enable with on-demand, PostgreSQL-as-a-Service workflows.  

### Cluster (pgclusters)

The Cluster or pgcluster CRD is used by the PostgreSQL Operator to define the PostgreSQL cluster definition and make new PostgreSQL cluster requests. 

### Backup (pgbackups)

The Backup or pgbackup CRD is used by the PostgreSQL Operator to perform a pgbasebackup and to hold the workflow and status of the last backup job.  Crunchy Data plans to deprecate this CRD in a future release in favor of a more general pgtask resource

### Tasks (pgtask)

The Tasks or pgtask CRD is used by the PostgreSQL Operator to perform workflow and other related administration tasks.  The pgtasks CRD captures workflows and administrative tasks for a given pgcluster. 

### Replica (pgreplica)

The Replica or pgreplica CRD is used by teh PostgreSQL Operator to create a PostgreSQL replica.  When a user creates a PostgreSQL replica, a pgreplica CRD is created to define that replica.

Metadata about each PostgreSQL cluster deployed by the PostgreSQL Operator are stored within these CRD resources which act as the source of truth for the
Operator.  The PostgreSQL Operator makes use of CRDs to maintain state and resource definitions as offered by the PostgreSQL Operator. 






