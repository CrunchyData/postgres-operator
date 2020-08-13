---
title: "PostgreSQL Operator Monitoring"
date:
draft: false
weight: 60
---

The PostgreSQL Operator Monitoring infrastructure is a fully integrated solution for monitoring
and visualizing metrics captured from PostgreSQL clusters created using the PostgreSQL Operator.
By leveraging [pgMonitor][] to configure and integrate
the various tools, components and metrics needed to effectively monitor PostgreSQL clusters,
the PostgreSQL Operator Monitoring infrastructure provides an powerful and easy-to-use solution
to effectively monitor and visualize pertinent PostgreSQL database and container metrics.
Included in the monitoring infrastructure are the following components:

- [pgMonitor][] - Provides the configuration
needed to enable the effective capture and visualization of PostgreSQL database metrics using
the various tools comprising the PostgreSQL Operator Monitoring infrastructure
- [Grafana](https://grafana.com/) - Enables visual dashboard capabilities for monitoring
PostgreSQL clusters, specifically using Crunchy PostgreSQL Exporter data stored within Prometheus
- [Prometheus](https://prometheus.io/) - A multi-dimensional data model with time series data,
which is used in collaboration with the Crunchy PostgreSQL Exporter to provide and store
metrics
- [Alertmanager](https://prometheus.io/docs/alerting/latest/alertmanager/) - Handles alerts 
sent by Prometheus by deduplicating, grouping, and routing them to reciever integrations.

When installing the monitoring infrastructure, various configuration options and settings
are available to tailor the installation according to your needs.  For instance, custom dashboards
and datasources can be utilized with Grafana, while custom scrape configurations can be utilized
with Promtheus.  Please see the
[monitoring configuration reference](<{{< relref "/installation/metrics/metrics-configuration.md">}}>)
for additional details.

By leveraging the various installation methods described in this section, the PostgreSQL Operator
Metrics infrastructure can be deployed alongside the PostgreSQL Operator.  There are several
different ways to install and deploy the
[PostgreSQL Operator Monitoring infrastructure](https://www.crunchydata.com/developers/download-postgres/containers/postgres-operator)
based upon your use case.

For the vast majority of use cases, we recommend using the
[PostgreSQL Operator Monitoring Installer]({{< relref "/installation/metrics/postgres-operator-metrics.md" >}}),
which uses the `pgo-deployer` container to set up all of the objects required to
run the PostgreSQL Operator Monitoring infrastructure.  
Additionally, [Ansible](<{{< relref "/installation/metrics/metrics-configuration.md">}}>) and
[Helm](<{{< relref "/installation/metrics/other/ansible">}}>) installers are available.

Before selecting your installation method, it's important that you first read
the [prerequisites]({{< relref "/installation/metrics/metrics-prerequisites.md" >}}) for your
deployment environment to ensure that your setup meets the needs for installing
the PostgreSQL Operator Monitoring infrastructure.

[pgMonitor]: https://github.com/CrunchyData/pgmonitor
