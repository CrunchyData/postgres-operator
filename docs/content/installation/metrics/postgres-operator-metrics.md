---
title: Install PostgreSQL Operator Monitoring
date:
draft: false
weight: 20
---

# PostgreSQL Operator Monitoring Installer

## Quickstart

If you believe that all the default settings in the installation manifest work
for you, you can take a chance by running the metrics manifest directly from the
repository:

```
kubectl create namespace pgo
kubectl apply -f https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/metrics/kubectl/postgres-operator-metrics.yml
```

Note that in OpenShift and CodeReady Containers you will need to set the
`disable_fsgroup` to `true` attribute to `true` if you are using the
`restricted` Security Context Constraint (SCC). If you are using the `anyuid`
SCC, you will need to set `disable_fsgroup` to `false`.

However, we still advise that you read onward to see how to properly configure
the PostgreSQL Operator Monitoring infrastructure.

## Overview

The PostgreSQL Operator comes with a container called `pgo-deployer` which
handles a variety of lifecycle actions for the PostgreSQL Operator Monitoring infrastructure,
including:

- Installation
- Upgrading
- Uninstallation

After configuring the Job template, the installer can be run using
[`kubectl apply`](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#apply)
and takes care of setting up all of the objects required to run the PostgreSQL
Operator.

The installation manifest, called [`postgres-operator-metrics.yml`](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/metrics/kubectl/postgres-operator-metrics.yml), is available in the [`installers/metrics/kubectl/postgres-operator-metrics.yml`](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/metrics/kubectl/postgres-operator-metrics.yml)
path in the PostgreSQL Operator repository.


## Requirements

### RBAC

The `pgo-deployer` requires a [ServiceAccount](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
and [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)
to run the installation job. Both of these resources are already defined
in the `postgres-operator-metrics.yml`, but can be updated based on your specific
environmental requirements.

By default, the `pgo-deployer` uses a ServiceAccount called `pgo-metrics-deployer-sa`
that has a ClusterRoleBinding (`pgo-metrics-deployer-crb`) with several ClusterRole
permissions.  This ClusterRole is needed for the initial configuration and deployment
of the various applications comprising the monitoring infrastructure.  This includes permissions
to create:

* RBAC for use by Prometheus and/or Grafana
* The metrics namespace

The required list of privileges are available in the
[postgres-operator-metrics.yml](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/metrics/kubectl/postgres-operator-metrics.yml)
file:

[https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/metrics/kubectl/postgres-operator-metrics.yml](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml)

If you have already configured the ServiceAccount and ClusterRoleBinding for the
installation process (e.g. from a previous installation), then you can remove
these objects from the `postgres-operator-metrics.yml` manifest.

### Config Map

The `pgo-deployer` uses a [Kubernetes ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)
to pass configuration options into the installer. The ConfigMap is defined in
the `postgres-operator-metrics.yaml` file and can be updated based on your configuration
preferences.

### Namespaces

By default, the PostgreSQL Operator Monitoring installer will run in the `pgo` Namespace. This can be
updated in the `postgres-operator-metrics.yml` file. **Please ensure that this namespace
exists before the job is run**.

For example, to create the `pgo` namespace:

```
kubectl create namespace pgo
```

## Configuration - `postgres-operator-metrics.yml`

The `postgres-operator-metrics.yml` file contains all of the configuration parameters
for deploying PostgreSQL Operator Monitoring. The [example file](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/metrics/kubectl/postgres-operator-metrics.yml)
contains defaults that should work in most Kubernetes environments, but it may
require some customization.

Note that in OpenShift and CodeReady Containers you will need to set the
`disable_fsgroup` to `true` attribute to `true` if you are using the
`restricted` Security Context Constraint (SCC). If you are using the `anyuid`
SCC, you will need to set `disable_fsgroup` to `false`.

For a detailed description of each configuration parameter, please read the
[PostgreSQL Operator Monitoring Installer Configuration Reference](<{{< relref "/installation/metrics/metrics-configuration.md">}}>)

#### Configuring to Update and Uninstall

The deploy job can be used to perform different deployment actions for the
PostgreSQL Operator Monitoring infrastructure. When you run the job it will install
the monitoring infrastructure by default but you can change the deployment action to
uninstall or update. The `DEPLOY_ACTION` environment variable in the `postgres-operator-metrics.yml`
file can be set to `install-metrics`, `update-metrics`, and `uninstall-metrics`.

### Image Pull Secrets

If you are pulling PostgreSQL Operator Monitoring images from a private registry, you
will need to setup an
[imagePullSecret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
with access to the registry. The image pull secret will need to be added to the
installer service account to have access. The secret will need to be created in
each namespace that the PostgreSQL Operator will be using.

After you have configured your image pull secret in the Namespace the installer
runs in (by default, this is `pgo`),
add the name of the secret to the job yaml that you are using. You can update
the existing section like this:

```
apiVersion: v1
kind: ServiceAccount
metadata:
    name: pgo-metrics-deployer-sa
    namespace: pgo
imagePullSecrets:
  - name: <image_pull_secret_name>
```

If the service account is configured without using the job yaml file, you
can link the secret to an existing service account with the `kubectl` or `oc`
clients.

```
# kubectl
kubectl patch serviceaccount <deployer-sa> -p '{"imagePullSecrets": [{"name": "myregistrykey"}]}' -n <install-namespace>

# oc
oc secrets link <registry-secret> <deployer-sa> --for=pull --namespace=<install-namespace>
```

## Installation

Once you have configured the PostgreSQL Operator Monitoring installer to your
specification, you can install the PostgreSQL Operator Monitoring infrastructure
with the following command:

```shell
kubectl apply -f /path/to/postgres-operator-metrics.yml
```

## Post-Installation

To clean up the installer artifacts, you can simply run:

```shell
kubectl delete -f /path/to/postgres-operator-metrics.yml
```

Note that if you still have the ServiceAccount and ClusterRoleBinding in there,
you will need to have elevated privileges.
