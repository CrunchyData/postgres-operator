---
title: "PGO Monitoring"
date:
draft: false
weight: 100
---

The PGO Monitoring stack is a fully integrated solution for monitoring and visualizing metrics
captured from PostgreSQL clusters created using PGO.  By leveraging [pgMonitor][] to configure
and integrate the various tools, components and metrics needed to effectively monitor PostgreSQL
clusters, PGO Monitoring provides an powerful and easy-to-use solution to effectively monitor
and visualize pertinent PostgreSQL database and container metrics. Included in the monitoring
infrastructure are the following components:

- [pgMonitor][] - Provides the configuration needed to enable the effective capture and
visualization of PostgreSQL database metrics using the various tools comprising the PostgreSQL
Operator Monitoring infrastructure
- [Grafana](https://grafana.com/) - Enables visual dashboard capabilities for monitoring
PostgreSQL clusters, specifically using Crunchy PostgreSQL Exporter data stored within Prometheus
- [Prometheus](https://prometheus.io/) - A multi-dimensional data model with time series data,
which is used in collaboration with the Crunchy PostgreSQL Exporter to provide and store
metrics
- [Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/) - Handles alerts
sent by Prometheus by deduplicating, grouping, and routing them to receiver integrations.

By leveraging the installation method described in this section, PGO Monitoring can be deployed
alongside PGO.



[pgMonitor]: https://github.com/CrunchyData/pgmonitor
