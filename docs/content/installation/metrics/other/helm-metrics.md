---
title: "Monitoring Helm Chart"
date:
draft: false
weight: 20
---

# The PostgreSQL Operator Monitoring Helm Chart

## Overview

The PostgreSQL Operator comes with a container called `pgo-deployer` which
handles a variety of lifecycle actions for the PostgreSQL Operator Monitoring infrastructure,
including:

- Installation
- Upgrading
- Uninstallation

After configuring the `values.yaml` file with you configuration options, the
installer will be run using the `helm` command line tool and takes care of
setting up all of the objects required to run the PostgreSQL Operator.

The PostgreSQL Operator Monitoring Helm chart is available in the
[Helm](https://github.com/CrunchyData/postgres-operator/tree/master/installers/metrics/helm)
directory in the PostgreSQL Operator repository.

## Requirements

### RBAC

The Helm chart will create the ServiceAccount, ClusterRole, and ClusterRoleBinding
that are required to run the `pgo-deployer`. If you have already configured the
ServiceAccount and ClusterRoleBinding for the installation process (e.g. from a
previous installation), you can disable their creation using the `rbac.create`
and `serviceAccount.create` variables in the `values.yaml` file. If these options
are disabled, you must provide the name of your preconfigured ServiceAccount using
`serviceAccount.name`.

### Namespace

In order to install the PostgreSQL Operator using the Helm chart you will need
to first create the namespace in which the `pgo-deployer` will be run. By default,
it will run in the namespace that is provided to `helm` at the command line.

```
kubectl create namespace <namespace>
helm install postgres-operator-metrics -n <namespace> /path/to/chart_dir
```

### Config Map

The `pgo-deployer` uses a [Kubernetes ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)
to pass configuration options into the installer. The values in your `values.yaml`
file will be used to populate the configuation options in the ConfigMap.

### Configuration - `values.yaml`

The `values.yaml` file contains all of the configuration parameters for deploying
the PostgreSQL Operator Monitoring infrastructure.
The [values.yaml file](https://github.com/CrunchyData/postgres-operator/blob/master/installers/metrics/helm/values.yaml)
contains the defaults that should work in most Kubernetes environments, but it may require some customization.

Note that in OpenShift and CodeReady Containers you will need to set the
`disable_fsgroup` to `true` attribute to `true` if you are using the
`restricted` Security Context Constraint (SCC). If you are using the `anyuid`
SCC, you will need to set `disable_fsgroup` to `false`.

For a detailed description of each configuration parameter, please read the
[PostgreSQL Operator Monitoring Installer Configuration Reference](<{{< relref "/installation/metrics/metrics-configuration.md">}}>)

## Installation

Once you have configured the PostgreSQL Operator Monitoring installer to your
specification, you can install the PostgreSQL Operator Monitoring infrastructure
with the following command:

```shell
helm install <name> -n <namespace> /path/to/chart_dir
```

{{% notice tip %}}
Take note of the `name` used when installing, this `name` will be used to
upgrade and uninstall the PostgreSQL Operator.
{{% /notice %}}

## Upgrade and Uninstall

Once install has be completed using Helm, it will also be used to upgrade and
uninstall your PostgreSQL Operator.

{{% notice tip %}}
The `name` and `namespace` in the following sections should match the options
provided at install.
{{% /notice %}}

### Upgrade

To make changes to your deployment of the PostgreSQL Operator you will use the
`helm upgrade` command. Once the configuration changes have been made to you
`values.yaml` file, you can run the following command to implement them in the
deployment:

```shell
helm upgrade <name> -n <namespace> /path/to/updated_chart
```

### Uninstall

To uninstall the PostgreSQL Operator you will use the `helm uninstall` command.
This will uninstall the operator and clean up resources used by the `pgo-deployer`.

```shell
helm uninstall <name> -n <namespace>
```

## Debugging

When the `pgo-deployer` job does not complete successfully, the resources that
are created and normally cleaned up by Helm will be left in your
Kubernetes cluster. This will allow you to use the failed job and its logs to
debug the issue. The following command will show the logs for the `pgo-deployer`
job:

```shell
kubectl logs -n <namespace> job.batch/pgo-metrics-deploy
```

{{% notice tip %}}
You can also view the logs as the job is running by using the `kubectl -f`
follow flag:
```shell
kubectl logs -n <namespace> job.batch/pgo-metrics-deploy -f
```
{{% /notice %}}


These logs will provide feedback if there are any misconfigurations in your
install. Once you have finished debugging the failed job and fixed any configuration
issues, you can take steps to re-run your install, upgrade, or uninstall. By
running another command the resources from the failed install will be cleaned up
so that a successfull install can run.
