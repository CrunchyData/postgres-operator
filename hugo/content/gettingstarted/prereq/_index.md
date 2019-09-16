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


## Service Ports

This is a list of service ports that are used in the PostgreSQL Operator. Verify that the ports are open and can be used.

| Service | Port |
| --- | --- |
| PostgreSQL | 5432 |
| pgpool | 5432 |
| pgbouncer | 5432 |
| pgbackrest | 2022 |
| node-exporter | 9100 |
| postgres-exporter | 9187 |

## Application Ports

This is a list of ports used by application containers that connect to the PostgreSQL Operator. If you are using these apps, verify that the port for for that app is open and can be used.

| App | Port |
| --- | --- |
| pgbadger | 10000 |
| Grafana | 3000 |
| Prometheus | 9090 |
