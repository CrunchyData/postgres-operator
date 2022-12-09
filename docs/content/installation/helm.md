---
title: "Helm"
date:
draft: false
weight: 20
---

# Installing PGO Using Helm

This section provides instructions for installing and configuring PGO using Helm.

There are two sources for the PGO Helm chart:
* the Postgres Operator examples repo;
* the Helm chart hosted on the Crunchy container registry, which supports direct Helm installs.

# The Postgres Operator Examples repo

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
`controllerImages` setting:

```yaml
controllerImages:
  cluster: {{< param operatorRepository >}}:{{< param postgresOperatorTag >}}
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

### Automated Upgrade Checks

By default, PGO will automatically check for updates to itself and software components by making a request to a URL. If PGO detects there are updates available, it will print them in the logs. As part of the check, PGO will send aggregated, anonymized information about the current deployment to the endpoint. An upcoming release will allow for PGO to opt-in to receive and apply updates to software components automatically.

PGO will check for updates upon startup and once every 24 hours. Any errors in checking will have no impact on PGO's operation. To disable the upgrade check, you can set the `disable_check_for_upgrades` value in the Helm chart to `true`.

For more information about collected data, see the Crunchy Data [collection notice](https://www.crunchydata.com/developers/data-collection-notice).

## Uninstall

To uninstall PGO, remove all your PostgresCluster objects, then use the `helm uninstall` command:

```shell
helm uninstall <name> -n <namespace>
```

Helm [leaves the CRDs][helm-crd-limits] in place. You can remove them with `kubectl delete`:

```shell
kubectl delete -f helm/install/crds
```

# The Crunchy Container Registry

## Installing directly from the registry

Crunchy Data hosts an OCI registry that `helm` can use directly.
(Not all `helm` commands support OCI registries. For more information on
which commands can be used, see [the Helm documentation](https://helm.sh/docs/topics/registries/).)

You can install PGO directly from the registry using the `helm install` command:

```
helm install pgo {{< param operatorHelmRepository >}}
```

Or to see what values are set in the default `values.yaml` before installing, you could run a
`helm show` command just as you would with any other registry:

```
helm show values {{< param operatorHelmRepository >}}
```

## Downloading from the registry

Rather than deploying directly from the Crunchy registry, you can instead use the registry as the
source for the Helm chart.

To do so, download the Helm chart from the Crunchy Container Registry:

```
# To pull down the most recent Helm chart
helm pull {{< param operatorHelmRepository >}}

# To pull down a specific Helm chart
helm pull {{< param operatorHelmRepository >}} --version {{< param operatorVersion >}}
```

Once the Helm chart has been downloaded, uncompress the bundle

```
tar -xvf pgo-{{< param operatorVersion >}}.tgz
```

And from there, you can follow the instructions above on setting the [Configuration](#configuration)
and installing a local Helm chart.
