---
title: "Kustomize"
date:
draft: false
weight: 10
---

# Installing PGO Using Kustomize

This section provides instructions for installing and configuring PGO using Kustomize.

If you are deploying using the installer from the [Crunchy Data Customer Portal](https://access.crunchydata.com/), please refer to the guide there for alternative setup information.

## Prerequisites

First, go to GitHub and [fork the Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork)
repository, which contains the PGO Kustomize installer.

[https://github.com/CrunchyData/postgres-operator-examples/fork](https://github.com/CrunchyData/postgres-operator-examples/fork)

Once you have forked this repo, you can download it to your working environment with a command
similar to this:

```
YOUR_GITHUB_UN="<your GitHub username>"
git clone --depth 1 "git@github.com:${YOUR_GITHUB_UN}/postgres-operator-examples.git"
cd postgres-operator-examples
```

The PGO installation project is located in the `kustomize/install` directory.

## Configuration

While the default Kustomize install should work in most Kubernetes environments, it may be
necessary to further customize the Kustomize project(s) according to your specific needs.

For instance, to customize the image tags utilized for the PGO Deployment, the `images` setting
in the `kustomize/install/default/kustomization.yaml` file can be modified:

```yaml
images:
- name: postgres-operator
  newName: {{< param operatorRepository >}}
  newTag: {{< param postgresOperatorTag >}}
```

If you are deploying using the images from the [Crunchy Data Customer Portal](https://access.crunchydata.com/), please refer to the [private registries]({{< relref "guides/private-registries.md" >}}) guide for additional setup information.

Please note that the Kustomize install project will also create a namespace for PGO
by default (though it is possible to install without creating the namespace, as shown below).  To
modify the name of namespace created by the installer, the `kustomize/install/namespace/namespace.yaml`
should be modified:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: custom-namespace
```

The `namespace` setting in  `kustomize/install/default/kustomization.yaml` should be
modified accordingly.

```yaml
namespace: custom-namespace
```

By default, PGO deploys with debug logging turned on. If you wish to disable this, you need to set the `CRUNCHY_DEBUG` environmental variable to `"false"` that is found in the `kustomize/install/manager/manager.yaml` file. Alternatively, you can add the following to your `kustomize/install/manager/kustomization.yaml` to disable debug logging:

```yaml
patchesStrategicMerge:
- |-
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: pgo
  spec:
    template:
      spec:
        containers:
        - name: operator
          env:
          - name: CRUNCHY_DEBUG
            value: "false"
```

You can also create additional Kustomize overlays to further patch and customize the installation according to your specific needs.

### Installation Mode

When PGO is installed, it can be configured to manage PostgreSQL clusters in all namespaces within
the Kubernetes cluster, or just those within a single namespace.  When managing PostgreSQL
clusters in all namespaces, a ClusterRole and ClusterRoleBinding is created to ensure PGO has
the permissions it requires to properly manage PostgreSQL clusters across all namespaces.  However,
when PGO is configured to manage PostgreSQL clusters within a single namespace only, a Role and
RoleBinding is created instead.

The installation of the necessary resources for a cluster-wide or a namespace-limited
operator is done automatically by Kustomize, as described below in the Install section.
The only potential change you may need to make is to the Namespace resource and the
`namespace` field if using a namespace other than the default `postgres-operator`.

## Install

Once the Kustomize project has been modified according to your specific needs, PGO can then
be installed using `kubectl` and Kustomize.  To create the target namespace, run the following:

```shell
kubectl apply -k kustomize/install/namespace
```

This will create the default `postgres-operator` namespace, unless you have edited the
`kustomize/install/namespace/namespace.yaml` resource. That `Namespace` resource should have the
same value as the `namespace` field in the `kustomization.yaml` file (located either at
`kustomize/install/default` or `kustomize/install/singlenamespace`, depending on whether you
are deploying the operator with cluster-wide or namespace-limited permissions).

To install PGO itself in cluster-wide mode, apply the kustomization file in the `default` folder:

```shell
kubectl apply -k kustomize/install/default
```

To install PGO itself in namespace-limited mode, apply the kustomization file in the
`singlenamespace` folder:

```shell
kubectl apply -k kustomize/install/singlenamespace
```

The `kustomization.yaml` files in those folders take care of applying the appropriate permissions.

### Automated Upgrade Checks

By default, PGO will automatically check for updates to itself and software components by making a request to a URL. If PGO detects there are updates available, it will print them in the logs. As part of the check, PGO will send aggregated, anonymized information about the current deployment to the endpoint. An upcoming release will allow for PGO to opt-in to receive and apply updates to software components automatically.

PGO will check for updates upon startup and once every 24 hours. Any errors in checking will have no impact on PGO's operation. To disable the upgrade check, you can set the `CHECK_FOR_UPGRADES` environmental variable on the `pgo` Deployment to `"false"`.

## Uninstall

Once PGO has been installed, it can also be uninstalled using `kubectl` and Kustomize.
To uninstall PGO (assuming it was installed in cluster-wide mode), the following command can be
utilized:

```shell
kubectl delete -k kustomize/install/default
```

To uninstall PGO installed with only namespace permissions, use:

```shell
kubectl delete -k kustomize/install/singlenamespace
```

The namespace created with this installation can likewise be cleaned up with:

```shell
kubectl delete -k kustomize/install/namespace
```
