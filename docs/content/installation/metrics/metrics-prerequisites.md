---
title: "Monitoring Prerequisites"
date:
draft: false
weight: 10
---

# Prerequisites

The following is required prior to installing PostgreSQL Operator Monitoring.

## Environment

PostgreSQL Operator Monitoring is tested in the following environments:

* Kubernetes v1.13+
* Red Hat OpenShift v3.11+
* Red Hat OpenShift v4.3+
* VMWare Enterprise PKS 1.3+

### Application Ports

The PostgreSQL Operator Monitoring installer deploys different services as needed to support 
PostgreSQL Operator Monitoring collection and monitoring. Below is a list of the applications
and their default Service ports.

| Service | Port |
| --- | --- |
| Grafana | 3000 |
| Prometheus | 9090 |
| Alertmanager | 9093 |
