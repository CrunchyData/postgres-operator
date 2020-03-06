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
