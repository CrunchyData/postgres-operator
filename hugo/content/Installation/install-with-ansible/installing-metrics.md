---
title: "Installing Metrics Stack"
date:
draft: false
weight: 22
---

# Installing

The following assumes the proper [prerequisites are satisfied](/installation/install-with-ansible/prereq/prerequisites/)
we can now install the PostgreSQL Operator.

The commands should be run in the directory where the Crunchy PostgreSQL Operator
playbooks is stored.  See the `ansible` directory in the Crunchy PostgreSQL Operator
project for the inventory file, main playbook and ansible roles.

## Installing on Linux

On a Linux host with Ansible installed we can run the following command to install 
the Metrics stack:

```bash
ansible-playbook -i /path/to/inventory --tags=install-metrics main.yml
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
