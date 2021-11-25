---
title: "Helm"
date:
draft: false
weight: 20
---

# Installing PGO Using Helm

This section provides instructions for installing and configuring PGO using Helm.

## Prerequisites

First, go to GitHub and [fork the Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork)
repository, which contains the PGO Helm installer.

[https://github.com/CrunchyData/postgres-operator-examples/fork](https://github.com/CrunchyData/postgres-operator-examples/fork)

Once you have forked this repo, you can download it to your working environment with a command
similar to this:

```
YOUR_GITHUB_UN="<your GitHub username>"
git clone --depth 1 "git@github.com:${YOUR_GITHUB_UN}/postgres-operator-examples.git"
cd postgres-operator-examples
```

The PGO Helm chart is located in the `helm/install` directory of this repository.

## Configuration

The `values.yaml` file for the Helm chart contains all of the available configuration settings for
PGO. The default `values.yaml` settings should work in most Kubernetes environments, but it may
require some customization depending on your specific environment and needs.

For instance, it might be necessary to customize the image tags that are utilized using the
`image` setting:

```yaml
image:
  image: {{< param repository >}}/postgres-operartor:{{< param postgresOperatorTag >}}
```

Please note that the `values.yaml` file is located in `helm/install`.

### Logging

By default, PGO deploys with debug logging turned on. If you wish to disable this, you need to set the `debug` attribute in the `values.yaml` to false, e.g.:

```yaml
debug: false
```

### Installation Mode

When PGO is installed, it can be configured to manage PostgreSQL clusters in all namespaces within
the Kubernetes cluster, or just those within a single namespace.  When managing PostgreSQL
clusters in all namespaces, a ClusterRole and ClusterRoleBinding is created to ensure PGO has
the permissions it requires to properly manage PostgreSQL clusters across all namespaces.  However,
when PGO is configured to manage PostgreSQL clusters within a single namespace only, a Role and
RoleBinding is created instead.

In order to select between these two modes when installing PGO using Helm, the `singleNamespace`
setting in the `values.yaml` file can be utilized:

```yaml
singleNamespace: false
```

Specifically, if this setting is set to `false` (which is the default), then a ClusterRole and
ClusterRoleBinding will be created, and PGO will manage PostgreSQL clusters in all namespaces.
However, if this setting is set to `true`, then a Role and RoleBinding will be created instead,
allowing PGO to only manage PostgreSQL clusters in the same namespace utilized when installing
the PGO Helm chart.

## Install

Once you have configured the Helm chart according to your specific needs, it can then be installed
using `helm`:

```shell
helm install <name> -n <namespace> helm/install
```

## Upgrade and Uninstall

And once PGO has been installed, it can then be upgraded and uninstalled using applicable `helm`
commands:

```shell
helm upgrade <name> -n <namespace> helm/install
```

```shell
helm uninstall <name> -n <namespace>
```

## Automated check for upgrades

To help keep track of developments to PGO, you have the option of turning on a process that
will check for available versions. If you set the environment variable `CHECK_FOR_UPGRADES`
to `true` in your PGO deployment, that will start a process that will check available
PGO versions every 24 hours.

Currently this process is set to only log information, and so should not interfere
with PGO's regular functions: if it retrieves information or runs into an error, it will
log that event without interrupting PGO's performance.

This is currently a work-in-progress.
