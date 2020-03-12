---
title: "Prerequisites"
date:
draft: false
weight: 20
---

# Prerequisites

The following is required prior to installing PostgreSQL Operator:

* Kubernetes v1.13+
* Red Hat OpenShift v3.11+
* VMWare Enterprise PKS 1.3+
* `kubectl` or `oc` configured to communicate with Kubernetes

### IBM Cloud Pak Data

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

We recommend using the [Ansible](/installation/install-with-ansible/prerequisites/)
installation method to install Crunchy PostgreSQL for Kubernetes in a Cloud Pak
Data environment.   

## Container Ports

The API server port is required to connect to the API server with the `pgo` cli. The `nsqd` and `nsqadmin` ports are required to connect to the event stream and listen for real-time events.

| Container | Port |
| --- | --- |
| API Server | 8443 |
| nsqadmin | 4151 |
| nsqd | 4150 |

## Service Ports

This is a list of service ports that are used in the PostgreSQL Operator. Verify that the ports are open and can be used.

| Service | Port |
| --- | --- |
| PostgreSQL | 5432 |
| pgbouncer | 5432 |
| pgbackrest | 2022 |
| postgres-exporter | 9187 |

## Application Ports

This is a list of ports used by application containers that connect to the PostgreSQL Operator. If you are using one of these apps, verify that the service port for that app is open and can be used.

| App | Port |
| --- | --- |
| pgbadger | 10000 |
| Grafana | 3000 |
| Prometheus | 9090 |
