---
title: "Installing Metrics Infrastructure"
date:
draft: false
weight: 22
---

# Installing

PostgreSQL clusters created by the Crunchy PostgreSQL Operator can optionally be 
configured to serve performance metrics via Prometheus Exporters.  The metric exporters 
included in the database pod serve realtime metrics for the database container.  In 
order to store and view this data, Grafana and Prometheus are required.  The Crunchy 
PostgreSQL Operator does not create this infrastructure, however, they can be installed 
using the provided Ansible roles.

## Prerequisites

The following assumes the proper [prerequisites are satisfied](/installation/install-with-ansible/prereq/prerequisites/)
we can now install the PostgreSQL Operator.

At a minimum, the following inventory variables should be configured to install the 
metrics infrastructure:

| Name                              | Default     | Description                                                                                                                                                                      |
|-----------------------------------|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `ccp_image_prefix`                | crunchydata | Configures the image prefix used when creating containers from Crunchy Container Suite.                                                                                          |
| `ccp_image_tag`                   |             | Configures the image tag (version) used when creating containers from Crunchy Container Suite.                                                                                   |
| `grafana_admin_username`          | admin       | Set to configure the login username for the Grafana administrator.                                                                                                               |
| `grafana_admin_password`          |             | Set to configure the login password for the Grafana administrator.                                                                                                               |
| `grafana_install`                 | true        | Set to true to install Crunchy Grafana to visualize metrics.                                                                                                                     |
| `grafana_storage_access_mode`     |             | Set to the access mode used by the configured storage class for Grafana persistent volumes.                                                                                      |
| `grafana_storage_class_name`      |             | Set to the name of the storage class used when creating Grafana persistent volumes.                                                                                              |
| `grafana_volume_size`             |             | Set to the size of persistent volume to create for Grafana.                                                                                                                      |
| `kubernetes_context`              |             | When deploying to Kubernetes, set to configure the context name of the kubeconfig to be used for authentication.                                                                 |
| `metrics`                         | false       | Set to true enable performance metrics on all newly created clusters.  This can be disabled by the client.                                                                       |
| `metrics_namespace`               | metrics     | Configures the target namespace when deploying Grafana and/or Prometheus                                                                                                         |
| `openshift_host`                  |             | When deploying to OpenShift, set to configure the hostname of the OpenShift cluster to connect to.                                                                               |
| `openshift_password`              |             | When deploying to OpenShift, set to configure the password used for login.                                                                                                       |
| `openshift_skip_tls_verify`       |             | When deploying to Openshift, set to ignore the integrity of TLS certificates for the OpenShift cluster.                                                                          |
| `openshift_token`                 |             | When deploying to OpenShift, set to configure the token used for login (when not using username/password authentication).                                                        |
| `openshift_user`                  |             | When deploying to OpenShift, set to configure the username used for login.                                                                                                       |
| `prometheus_install`              | true        | Set to true to install Crunchy Prometheus timeseries database.                                                                                                                   |
| `prometheus_storage_access_mode`  |             | Set to the access mode used by the configured storage class for Prometheus persistent volumes.                                                                                   |
| `prometheus_storage_class_name`   |             | Set to the name of the storage class used when creating Prometheus persistent volumes.                                                                                           |

{{% notice tip %}}
Administrators can choose to install Grafana, Prometheus or both by configuring the 
`grafana_install` and `prometheus_install` variables in the inventory files.
{{% /notice %}}

The following commands should be run in the directory where the Crunchy PostgreSQL Operator
playbooks are located.  See the `ansible` directory in the Crunchy PostgreSQL Operator
project for the inventory file, main playbook and ansible roles.

{{% notice tip %}}
At this time the Crunchy PostgreSQL Operator Playbooks only support storage classes.
For more information on storage classes see the [official Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/).
{{% /notice %}}

## Installing on Linux

On a Linux host with Ansible installed we can run the following command to install 
the Metrics stack:

```bash
ansible-playbook -i /path/to/inventory --tags=install-metrics main.yml
```

If the Crunchy PostgreSQL Operator playbooks were installed using `yum`, use the
following commands:

```bash
export ANSIBLE_ROLES_PATH=/usr/share/ansible/roles/crunchydata

ansible-playbook -i /path/to/inventory --tags=install-metrics --ask-become-pass \
    /usr/share/ansible/postgres-operator/playbooks/main.yml
```

## Installing on MacOS

On a MacOS host with Ansible installed we can run the following command to install 
the Metrics stack:

```bash
ansible-playbook -i /path/to/inventory --tags=install-metrics main.yml
```

## Installing on Windows

On a Windows host with the Ubuntu subsystem we can run the following commands to install 
the Metrics stack:

```bash
ansible-playbook -i /path/to/inventory --tags=install-metrics main.yml
```

## Verifying the Installation

This may take a few minutes to deploy.  To check the status of the deployment run 
the following:

```bash
# Kubernetes
kubectl get deployments -n <NAMESPACE_NAME>
kubectl get pods -n <NAMESPACE_NAME>

# OpenShift
oc get deployments -n <NAMESPACE_NAME>
oc get pods -n <NAMESPACE_NAME>
```

## Verify Grafana

In a separate terminal we need to setup a port forward to the Crunchy Grafana deployment 
to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward <GRAFANA_POD_NAME> -n <METRICS_NAMESPACE> 3000:3000

# If deployed to OpenShift
oc port-forward <GRAFANA_POD_NAME> -n <METRICS_NAMESPACE> 3000:3000
```

In a browser navigate to `https://127.0.0.1:3000` to access the Grafana dashboard.

{{% notice tip %}}
No metrics will be scraped if no exporters are available.  To create a PostgreSQL
cluster with metric exporters run the following command:

```bash
pgo create cluster <NAME OF CLUSTER> --metrics --namespace=<NAMESPACE>
```
{{% /notice %}}

## Verify Prometheus

In a separate terminal we need to setup a port forward to the Crunchy Prometheus deployment 
to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward <PROMETHEUS_POD_NAME> -n <METRICS_NAMESPACE> 9090:9090

# If deployed to OpenShift
oc port-forward <PROMETHEUS_POD_NAME> -n <METRICS_NAMESPACE> 9090:9090
```

In a browser navigate to `https://127.0.0.1:9090` to access the Prometheus dashboard.

{{% notice tip %}}
No metrics will be scraped if no exporters are available.  To create a PostgreSQL 
cluster with metric exporters run the following command:

```bash
pgo create cluster <NAME OF CLUSTER> --metrics --namespace=<NAMESPACE>
```
{{% /notice %}}
