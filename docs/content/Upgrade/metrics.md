---
title: Monitoring Upgrade Guidance
date:
draft: false
weight: 100
---

# Upgrade Guidance for PostgreSQL Operator Monitoring

## Migration to Upstream Containers

The Crunchy PostgreSQL Monitoring infrastructure now uses upstream Prometheus and Grafana
containers. By default the installers will deploy the monitoring infrastructure using
images from Docker Hub but can easily be updated to point to a Red Hat certified
container repository. The Red Hat certified image catalog can be found
[here](https://catalog.redhat.com/software/containers/explore) and the Docker Hub
images can be found at the following links:

- https://hub.docker.com/r/prom/prometheus
- https://hub.docker.com/r/grafana/grafana
- https://hub.docker.com/r/prom/alertmanager

These containers are configurable through Kubernetes ConfigMaps and the updated
PostgreSQL Operator Monitoring installers. Once deployed Prometheus and Grafana
will be populated with resource data from metrics-enabled PostgreSQL clusters.

## New Monitoring Features

### Alerting
The updated PostgreSQL Operator Monitoring Infrastructure supports deployment of
Prometheus Alertmanager. This deployment uses upstream Prometheus
Alertmanager images that can be installed and configured with the metrics
installers and Kubernetes ConfigMaps.

### Updated pgMonitor
Prometheus and Grafana have been updated to include a default configuration from
[pgMonitor](https://github.com/CrunchyData/pgmonitor) that is tailored for
container-based PostgreSQL deployments. This updated configuration will show
container specific resource information from your metrics-enabled PostgreSQL
clusters. By default the metrics infrastructure will include:

- New Grafana dashboards tailored for container-based PostgreSQL deployments
- Container specific operating system metrics
- General PostgreSQL alerting rules.

### Updated Monitoring Installer
The installer for the PostgreSQL Operating Monitoring infrastructure has been 
split out into a separate set of installers. With each installer
([Ansible]({{< relref "/installation/metrics/other/ansible" >}}),
a [Kubectl job]({{< relref "installation/metrics/postgres-operator-metrics" >}}),
or [Helm]({{< relref "/installation/metrics/other/helm-metrics" >}}))
you will be able to apply custom configurations through Kubernetes
ConfigMaps. This includes:

- Custom Grafana dashboards and datasources
- Custom Prometheus scrape configuration
- Custom Prometheus alerting rules
- Custom Alertmanager notification configuration

## Updating from Pre-4.5.0 Monitoring

Ensure that you have a copy of any install or custom configurations you have
applied to your previous metrics install.

You can upgrade the Grafana and Prometheus deployments in place by using the new
installers. After you have updated the PostgreSQL Operator and configured the
`values.yaml`, run the
[metrics update]({{< relref "/installation/metrics/other/ansible/updating-metrics" >}}).
This will replace the old deployments while keeping your pvcs in place.

{{% notice tip %}}
To make use of the updated exporter queries you must update
the PostgreSQL Operator and
[upgrade]({{< relref "/upgrade/automatedupgrade" >}})
your cluster.
{{% /notice %}}


