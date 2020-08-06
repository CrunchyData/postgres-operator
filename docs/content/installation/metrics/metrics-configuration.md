---
title: "Metrics Configuration Reference"
date:
draft: false
weight: 30
---

# PostgreSQL Operator Metrics Installer Configuration

When installing the PostgreSQL Operator Metrics infrastructure you have various configuration options available, which
are defined below.

## General Configuration

These variables affect the general configuration of PostgreSQL Operator Metrics.

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `create_rbac` | true | **Required** | Set to true if the installer should create the RBAC resources required to run the PostgreSQL Operator Metrics infrastructure. |
| `db_port` | 5432 | **Required** | Set to configure the PostgreSQL port used by all PostgreSQL clusters. |
| `delete_metrics_namespace` | false |  | Set to configure whether or not the metrics namespace (defined using variable `metrics_namespace`) is deleted when uninstalling the metrics infrastructure. |
| `disable_fsgroup` | false |  | Set to `true` for deployments where you do not want to have the default PostgreSQL fsGroup (26) set. The typical usage is in OpenShift environments that have a `restricted` Security Context Constraints. |
| `grafana_admin_password` | admin | **Required** | Set to configure the login password for the Grafana administrator. |
| `grafana_admin_username` | admin | **Required** | Set to configure the login username for the Grafana administrator. |
| `grafana_install` | true | **Required** | Set to true to install Grafana to visualize metrics. |
| `grafana_service_type` | ClusterIP | **Required** | The service type to use for the Grafana service. |
| `grafana_storage_access_mode` | ReadWriteOnce | **Required** | Set to the access mode used by the configured storage class for Grafana persistent volumes. |
| `grafana_storage_class_name` | fast |  | Set to the name of the storage class used when creating Grafana persistent volumes.  Omit if not using storage classes. |
| `grafana_supplemental_groups` | 65534 |  | Set to configure any supplemental groups that should be added to security contexts for Grafana. |
| `grafana_volume_size` | 1G | **Required** | Set to the size of persistent volume to create for Grafana. |
| `metrics_image_pull_secret` |  |  | Name of a Secret containing credentials for container image registries. |
| `metrics_image_pull_secret_manifest` |  |  | Provide a path to the image Secret manifest to be installed in the metrics namespace. |
| `metrics_namespace` | 1G | **Required** | The namespace that should be created (if it doesn't already exist) and utilized for installation of the Matrics infrastructure. |
| `pgbadgerport` | 10000 | **Required** | Set to configure the port used by pgbadger in any PostgreSQL clusters. |
| `prometheus_install` | false | **Required** | Set to true to install Promotheus in order to capture metrics exported from PostgreSQL clusters. |
| `prometheus_service_type` | true | **Required** | The service type to use for the Prometheus service. |
| `prometheus_storage_access_mode` | ReadWriteOnce | **Required** | Set to the access mode used by the configured storage class for Prometheus persistent volumes. |
| `prometheus_storage_class_name` | fast |  | Set to the name of the storage class used when creating Prometheus persistent volumes.  Omit if not using storage classes. |
| `prometheus_supplemental_groups` | 65534 |  | Set to configure any supplemental groups that should be added to security contexts for Prometheus. |
| `prometheus_volume_size` | 1G | **Required** | Set to the size of persistent volume to create for Prometheus. |

## Custom Configuration

When installing the PostgreSQL Operator Metrics infrastructure, it is possible to further customize
the various Deployments included (e.g. Grafana and/or Prometheus) using custom configuration files.
Specifically, by pointing the PostgreSQL Operator Metrics installer to one or more ConfigMaps
containing any desired custom configuration settings, those settings will then be applied during
configuration and installation of the PostgreSQL Operator Metrics infrastructure.  

The specific custom configuration settings available are as follows:

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `prometheus_custom_configmap` | crunchy-prometheus |  | The name of a ConfigMap containing a custom `prometheus.yml` configuration file. |
| `grafana_datasources_custom_config` | grafana-datasources |  | The name of a ConfigMap containing custom Grafana datasources. |
| `grafana_dashboards_custom_config` | grafana-dashboards |   | The name of a ConfigMap containing custom Grafana dashboards. |

_Please note that when using custom ConfigMaps per the above configuration settings, any defaults
for the specific configuration being customized are no longer applied._

## Using RedHat Certified Containers & Custom Images

By default, the PostgreSQL Operator Metrics installer will deploy the official Grafana and
Prometheus containers that are publically available on [dockerhub](https://hub.docker.com/):

- https://hub.docker.com/r/grafana/grafana
- https://hub.docker.com/r/prom/prometheus

However, if RedHat certified containers are needed, the following certified images have also
been verified with the PostgreSQL Operator Metric infrastructure, and can therefore be
utilized instead:

- https://catalog.redhat.com/software/containers/openshift4/ose-grafana/5cdc17d55a13467289f58321
- https://catalog.redhat.com/software/containers/openshift4/ose-prometheus/5cdc1e585a13467289f5841a

The following configuration settings can be applied to properly configure the image prefix, name
and tag as needed to use the RedHat certified containers:

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `grafana_image_prefix` | grafana | **Required** | Configures the image prefix to use for the Grafana container.|
| `grafana_image_name` | grafana | **Required** | Configures the image name to use for the Grafana container. |
| `grafana_image_tag` | 6.7.4 | **Required** | Configures the image tag to use for the Grafana container. |
| `prometheus_image_prefix` | prom | **Required** | Configures the image prefix to use for the Prometheus container. |
| `prometheus_image_name` | promtheus | **Required** | Configures the image name to use for the Prometheus container. |
| `prometheus_image_tag` | v2.20.0 | **Required** | Configures the image tag to use for the Prometheus container. |

Additionally, these same settings can be utilized as needed to support custom image names,
tags, and additional container registries.

## Disabling Default Dashboards

By default, the following Grafana Dashboards are enabled as provided by the [pgmonitor project](https://github.com/CrunchyData/pgmonitor):

- Bloat_Details.json
- CRUD_Details.json
- PGBackrest.json
- PG_Details.json
- PG_Overview.json
- TableSize_Details.json

Using the `grafana-dashboard` configuration setting, it is possible to customize this list
of dashboards, and therefore further control which dashboards are loaded into Grafana
(specifically via the `grafana-dashboards` ConfigMap) by default.  For instance, the following
setting can be utilized to enable only a subset of the default dashboards defined above:

```yaml
grafana_dashboards:
  - CRUD_Details.json
  - PG_Details.json
  - PG_Overview.json
```

## Helm Only Configuration Settings

When using Helm, the following settings can be defined to control the image prefix and image tag
utilized for the `pgo-deployer` container that is run to install, update or uninstall the
PostgreSQL Operator Metrics infrastructure:

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `pgo_image_prefix` | registry.developers.crunchydata.com/crunchydata | **Required** | Configures the image prefix used by the `pgo-deployer` container | 
| `pgo_image_tag` | {{< param centosBase >}}-{{< param operatorVersion >}} | **Required** | Configures the image tag used by the `pgo-deployer` container |
