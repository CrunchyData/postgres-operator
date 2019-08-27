---
title: "Failover in the PostgreSQL Operator Overview"
date:
draft: false
weight: 6
---

## Failover in the PostgreSQL Operator

There are a number of potential events that could cause a primary database instance or cluster to become unavailable during the course of normal operations, including: 

* A database storage (disk) failure or any other hardware failure 
* The network on which the database resides becomes unreachable
* The host operating system becomes unstable and crashes
* A key database file becomes corrupted
* Total loss of data center

There may also be downtime events that are due to the normal case of operations, such as performing a minor upgrade, security patching of operating system, hardware upgrade, or other maintenance.

To enable rapid recovery from the unavailability of the primary PostgreSQL instance within a PostgreSQL cluster, the PostgreSQL Operator supports both Manual and Automated failover within a single Kubernetes cluster. 

### PostgreSQL Cluster Architecture 

The failover from a primary PostgreSQL instances to a replica PostgreSQL instance within a PostgreSQL cluster. 

### Manual Failover

Manual failover is performed by PostgreSQL Operator API actions involving a *query* and then a *target* being specified to pick the fail-over replica target.

### Automatic Failover 

Automatic failover is performed by the PostgreSQL Operator by evaluating the readiness of a primary.  Automated failover can be globally specified for all clusters or specific clusters. If desired, users can configure the PostgreSQL Operator to replace a failed primary PostgreSQL instance with a new PostgreSQL replica.

The PostgreSQL Operator automatic failover logic includes:

 * deletion of the failed primary Deployment
 * pick the best replica to become the new primary
 * label change of the targeted Replica to match the primary Service
 * execute the PostgreSQL promote command on the targeted replica
