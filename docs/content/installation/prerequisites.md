---
title: "Prerequisites"
date:
draft: false
weight: 10
---

# Prerequisites

The following is required prior to installing PostgreSQL Operator.

## Environment

The PostgreSQL Operator is tested in the following environments:

* Kubernetes v1.13+
* Red Hat OpenShift v3.11+
* Red Hat OpenShift v4.4+
* Amazon EKS
* VMWare Enterprise PKS 1.3+
* IBM Cloud Pak Data

#### IBM Cloud Pak Data

If you install the PostgreSQL Operator, which comes with Crunchy
PostgreSQL for Kubernetes, on IBM Cloud Pak Data, please note the following
additional requirements:

* Cloud Pak Data Version 2.5
* Minimum Node Requirements (Cloud Paks Cluster): 3
* Crunchy PostgreSQL for Kuberentes (Service):
  * Minimum CPU Requirements: 0.2 CPU
  * Minimum Memory Requirements: 120MB
  * Minimum Storage Requirements: 5MB

**Note**: PostgreSQL clusters deployed by the PostgreSQL Operator with
Crunchy PostgreSQL for Kubernetes are workload dependent. As such, users should
allocate enough resources for their PostgreSQL clusters.

## Client Interfaces

The PostgreSQL Operator installer will install the [`pgo` client]({{< relref "/pgo-client/_index.md" >}}) interface
to help with using the PostgreSQL Operator. However, it is also recommend that
you have access to [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
or [`oc`](https://www.okd.io/download.html) and are able to communicate with the
Kubernetes or OpenShift cluster that you are working with.

## Ports

There are several application ports to note when using the PostgreSQL Operator.
These ports allow for the [`pgo` client]({{< relref "/pgo-client/_index.md" >}})
to interface with the PostgreSQL Operator API as well as for users of the event
stream to connect to `nsqd` and `nsqdadmin`:

| Container | Port |
| --- | --- |
| API Server | 8443 |
| nsqadmin | 4151 |
| nsqd | 4150 |

If you are using these services, ensure your cluster administrator has given you
access to these ports.

### Application Ports

The PostgreSQL Operator deploys different services to support a production
PostgreSQL environment. Below is a list of the applications and their default
Service ports.

| Service | Port |
| --- | --- |
| PostgreSQL | 5432 |
| pgbouncer | 5432 |
| pgBackRest | 2022 |
| postgres-exporter | 9187 |
| pgbadger | 10000 |
