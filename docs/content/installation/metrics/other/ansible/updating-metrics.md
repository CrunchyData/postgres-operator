---
title: "Updating the Monitoring Infrastructure"
date:
draft: false
weight: 30
---

# Updating the Monitoring Infrastructure

Updating the PostgreSQL Operator Monitoring infrastrcutre is essential to the lifecycle management
of the service.  Using the `update-metrics` flag will:

* Update and redeploy the monitoring infrastructure deployments
* Recreate configuration maps and/or secrets used by the monitoring infrastructure
* Remove any deprecated objects
* Allow administrators to change settings configured in the `values.yaml`

The following assumes the proper [prerequisites are satisfied][ansible-prerequisites]
we can now update the PostgreSQL Operator.

The commands should be run in the directory where the Crunchy PostgreSQL Operator
playbooks is stored.  See the `ansible` directory in the Crunchy PostgreSQL Operator
project for the inventory file, values file, main playbook and ansible roles.

## Updating on Linux

On a Linux host with Ansible installed we can run the following command to update  
the PostgreSQL Operator:

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=update --ask-become-pass main.yml
```

## Updating on macOS

On a macOS host with Ansible installed we can run the following command to update  
the PostgreSQL Operator.

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=update --ask-become-pass main.yml
```

## Updating on Windows Ubuntu Subsystem

On a Windows host with an Ubuntu subsystem we can run the following commands to update  
the PostgreSQL Operator.

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=update --ask-become-pass main.yml
```

## Verifying the Update

This may take a few minutes to deploy.  To check the status of the deployment run
the following:

```bash
# Kubernetes
kubectl get deployments -n <metrics_namespace>
kubectl get pods -n <metrics_namespace>

# OpenShift
oc get deployments -n <metrics_namespace>
oc get pods -n <metrics_namespace>
```

## Verify Alertmanager

In a separate terminal we need to setup a port forward to the Crunchy Alertmanager deployment
to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n <METRICS_NAMESPACE> svc/crunchy-alertmanager  9093:9093

# If deployed to OpenShift
oc port-forward -n <METRICS_NAMESPACE> svc/crunchy-alertmanager 9093:9093
```

In a browser navigate to `http://127.0.0.1:9093` to access the Alertmanager dashboard.

## Verify Grafana

In a separate terminal we need to setup a port forward to the Crunchy Grafana deployment
to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n <METRICS_NAMESPACE> svc/crunchy-grafana 3000:3000

# If deployed to OpenShift
oc port-forward -n <METRICS_NAMESPACE> svc/crunchy-grafana 3000:3000
```

In a browser navigate to `http://127.0.0.1:3000` to access the Grafana dashboard.

{{% notice tip %}}
No metrics will be scraped if no exporters are available.  To create a PostgreSQL
cluster with metric exporters, run the following command following installation
of the PostgreSQL Operator:

```bash
pgo create cluster <NAME OF CLUSTER> --metrics --namespace=<NAMESPACE>
```
{{% /notice %}}

## Verify Prometheus

In a separate terminal we need to setup a port forward to the Crunchy Prometheus deployment
to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n <METRICS_NAMESPACE> svc/crunchy-prometheus  9090:9090

# If deployed to OpenShift
oc port-forward -n <METRICS_NAMESPACE> svc/crunchy-prometheus 9090:9090
```

In a browser navigate to `http://127.0.0.1:9090` to access the Prometheus dashboard.

{{% notice tip %}}
No metrics will be scraped if no exporters are available.  To create a PostgreSQL
cluster with metric exporters run the following command:

```bash
pgo create cluster <NAME OF CLUSTER> --metrics --namespace=<NAMESPACE>
```
{{% /notice %}}

[ansible-prerequisites]: {{< relref "/installation/metrics/other/ansible/metrics-prerequisites.md" >}}
