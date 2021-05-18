---
title: "Monitoring Configuration Reference"
date:
draft: false
weight: 30
---

# PostgreSQL Operator Monitoring Installer Configuration

When installing the PostgreSQL Operator Monitoring infrastructure you have various configuration options available, which
are defined below.

## General Configuration

These variables affect the general configuration of PostgreSQL Operator Monitoring.

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `alertmanager_log_level` | info |  | Set the log level for Alertmanager logging. |
| `alertmanager_service_type` | ClusterIP | **Required** | How to [expose][k8s-service-type] the Alertmanager service. |
| `alertmanager_storage_access_mode` | ReadWriteOnce | **Required** | Set to the access mode used by the configured storage class for Alertmanager persistent volumes. |
| `alertmanager_storage_class_name` | fast |  | Set to the name of the storage class used when creating Alertmanager persistent volumes.  Omit if not using storage classes. |
| `alertmanager_supplemental_groups` | 65534 |  | Set to configure any supplemental groups that should be added to security contexts for Alertrmanager. |
| `alertmanager_volume_size` | 1Gi | **Required** | Set to the size of persistent volume to create for Alertmanager. |
| `create_rbac` | true | **Required** | Set to true if the installer should create the RBAC resources required to run the PostgreSQL Operator Monitoring infrastructure. |
| `db_port` | 5432 | **Required** | Set to configure the PostgreSQL port used by all PostgreSQL clusters. |
| `delete_metrics_namespace` | false |  | Set to configure whether or not the metrics namespace (defined using variable `metrics_namespace`) is deleted when uninstalling the monitoring infrastructure. |
| `disable_fsgroup` | false |  | Set to `true` for deployments where you do not want to have the default PostgreSQL fsGroup (26) set. The typical usage is in OpenShift environments that have a `restricted` Security Context Constraints. If you use the `anyuid` SCC, you would want to set this to `false`. The Postgres Operator will set this value appropriately by default, except for when using the `anyuid` SCC.  |
| `grafana_admin_password` | admin | **Required** | Set to configure the login password for the Grafana administrator. |
| `grafana_admin_username` | admin | **Required** | Set to configure the login username for the Grafana administrator. |
| `grafana_install` | true | **Required** | Set to true to install Grafana to visualize metrics. |
| `grafana_service_type` | ClusterIP | **Required** | How to [expose][k8s-service-type] the Grafana service. |
| `grafana_storage_access_mode` | ReadWriteOnce | **Required** | Set to the access mode used by the configured storage class for Grafana persistent volumes. |
| `grafana_storage_class_name` | fast |  | Set to the name of the storage class used when creating Grafana persistent volumes.  Omit if not using storage classes. |
| `grafana_supplemental_groups` | 65534 |  | Set to configure any supplemental groups that should be added to security contexts for Grafana. |
| `grafana_volume_size` | 1Gi | **Required** | Set to the size of persistent volume to create for Grafana. |
| `metrics_image_pull_secret` |  |  | Name of a Secret containing credentials for container image registries. |
| `metrics_image_pull_secret_manifest` |  |  | Provide a path to the image Secret manifest to be installed in the metrics namespace. |
| `metrics_namespace` | 1G | **Required** | The namespace that should be created (if it doesn't already exist) and utilized for installation of the Matrics infrastructure. |
| `pgbadgerport` | 10000 | **Required** | Set to configure the port used by pgbadger in any PostgreSQL clusters. |
| `prometheus_install` | false | **Required** | Set to true to install Promotheus in order to capture metrics exported from PostgreSQL clusters. |
| `prometheus_service_type` | true | **Required** | How to [expose][k8s-service-type] the Prometheus service. |
| `prometheus_storage_access_mode` | ReadWriteOnce | **Required** | Set to the access mode used by the configured storage class for Prometheus persistent volumes. |
| `prometheus_storage_class_name` | fast |  | Set to the name of the storage class used when creating Prometheus persistent volumes.  Omit if not using storage classes. |
| `prometheus_supplemental_groups` | 65534 |  | Set to configure any supplemental groups that should be added to security contexts for Prometheus. |
| `prometheus_volume_size` | 1Gi | **Required** | Set to the size of persistent volume to create for Prometheus. |

## Custom Configuration

When installing the PostgreSQL Operator Monitoring infrastructure, it is possible to further customize
the various Deployments included (e.g. Alertmanager, Grafana, and/or Prometheus) using custom configuration files.
Specifically, by pointing the PostgreSQL Operator Monitoring installer to one or more ConfigMaps
containing any desired custom configuration settings, those settings will then be applied during
configuration and installation of the PostgreSQL Operator Monitoring infrastructure.  

The specific custom configuration settings available are as follows:

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `alertmanager_custom_config` | alertmanager-config |  | The name of a ConfigMap containing a custom `alertmanager.yml` configuration file. |
| `alertmanager_custom_rules_config` | alertmanager-rules-config |  | The name of a ConfigMap container custom alerting rules for Prometheus. |
| `grafana_datasources_custom_config` | grafana-datasources |  | The name of a ConfigMap containing custom Grafana datasources. |
| `grafana_dashboards_custom_config` | grafana-dashboards |   | The name of a ConfigMap containing custom Grafana dashboards. |
| `prometheus_custom_configmap` | crunchy-prometheus |  | The name of a ConfigMap containing a custom `prometheus.yml` configuration file. |

_Please note that when using custom ConfigMaps per the above configuration settings, any defaults
for the specific configuration being customized are no longer applied._

## Using Alertmanager

The Alertmanager deployment requires a custom configuration file to configure reciever
integrations that are supported by Prometheus Alertmanager. The installer will create
a configmap containing an example Alertmanager configuration file created by
the pgMonitor project, this file can be found in the [pgMonitor](https://github.com/CrunchyData/pgmonitor/blob/master/prometheus/crunchy-alertmanager.yml)
repository. This example file, along with the [Alertmanager configuration docs](https://prometheus.io/docs/alerting/latest/configuration/),
will help you to configure alerting for you specific use cases.

{{% notice tip %}}
Alertmanager cannot be installed without also deploying the Crunchy Prometheus deployment.
Once both are deployed, Prometheus is automatically configured to send alerts to
the Alertmanager.
{{% /notice %}}

## Using RedHat Certified Containers & Custom Images

By default, the PostgreSQL Operator Monitoring installer will deploy the official Grafana and
Prometheus containers that are publically available on [dockerhub](https://hub.docker.com/):

- https://hub.docker.com/r/grafana/grafana
- https://hub.docker.com/r/prom/prometheus
- https://hub.docker.com/r/prom/alertmanager

However, if RedHat certified containers are needed, the following certified images have also
been verified with the PostgreSQL Operator Metric infrastructure, and can therefore be
utilized instead:

- https://catalog.redhat.com/software/containers/openshift4/ose-grafana/5cdc17d55a13467289f58321
- https://catalog.redhat.com/software/containers/openshift4/ose-prometheus/5cdc1e585a13467289f5841a
- https://catalog.redhat.com/software/containers/openshift4/ose-prometheus-alertmanager/5cdc1cfbbed8bd5717d60b17

The following configuration settings can be applied to properly configure the image prefix, name
and tag as needed to use the RedHat certified containers:

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `alertmanager_image_prefix` | prom | **Required** | Configure the image prefix to use for the Alertmanager container. |
| `alertmanager_image_name` | alertmanager | **Required** | Configure the image name to use for the Alertmanager container. |
| `alertmanager_image_tag` | v0.21.0 | **Required** | Configures the image tag to use for the Alertmanager container. |
| `grafana_image_prefix` | grafana | **Required** | Configures the image prefix to use for the Grafana container.|
| `grafana_image_name` | grafana | **Required** | Configures the image name to use for the Grafana container. |
| `grafana_image_tag` | 7.4.5 | **Required** | Configures the image tag to use for the Grafana container. |
| `prometheus_image_prefix` | prom | **Required** | Configures the image prefix to use for the Prometheus container. |
| `prometheus_image_name` | promtheus | **Required** | Configures the image name to use for the Prometheus container. |
| `prometheus_image_tag` | v2.26.1 | **Required** | Configures the image tag to use for the Prometheus container. |

Additionally, these same settings can be utilized as needed to support custom image names,
tags, and additional container registries.

## Helm Only Configuration Settings

When using Helm, the following settings can be defined to control the image prefix and image tag
utilized for the `pgo-deployer` container that is run to install, update or uninstall the
PostgreSQL Operator Monitoring infrastructure:

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `pgo_image_prefix` | registry.developers.crunchydata.com/crunchydata | **Required** | Configures the image prefix used by the `pgo-deployer` container |
| `pgo_image_tag` | {{< param centosBase >}}-{{< param operatorVersion >}} | **Required** | Configures the image tag used by the `pgo-deployer` container |

[k8s-service-type]: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types
