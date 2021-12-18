---
title: "Kustomize"
date:
draft: false
weight: 10
---

# Installing PGO Monitoring Using Kustomize

This section provides instructions for installing and configuring PGO Monitoring using Kustomize.

## Prerequisites

First, go to GitHub and [fork the Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork)
repository, which contains the PGO Monitoring Kustomize installer.

[https://github.com/CrunchyData/postgres-operator-examples/fork](https://github.com/CrunchyData/postgres-operator-examples/fork)

Once you have forked this repo, you can download it to your working environment with a command
similar to this:

```
YOUR_GITHUB_UN="<your GitHub username>"
git clone --depth 1 "git@github.com:${YOUR_GITHUB_UN}/postgres-operator-examples.git"
cd postgres-operator-examples
```

The PGO Monitoring project is located in the `kustomize/monitoring` directory.

## Configuration

While the default Kustomize install should work in most Kubernetes environments, it may be
necessary to further customize the project according to your specific needs.

For instance, by default `fsGroup` is set to `26` for the `securityContext` defined for the
various Deployments comprising the PGO Monitoring stack:

```yaml
securityContext:
  fsGroup: 26
```

In most Kubernetes environments this setting is needed to ensure processes within the container
have the permissions needed to write to any volumes mounted to each of the Pods comprising the PGO
Monitoring stack.  However, when installing in an OpenShift environment (and more specifically when
using the `restricted` Security Context Constraint), the `fsGroup` setting should be removed
since OpenShift will automatically handle setting the proper `fsGroup` within the Pod's
`securityContext`.

Additionally, within this same section it may also be necessary to modify the `supplmentalGroups`
setting according to your specific storage configuration:

```yaml
securityContext:
  supplementalGroups : 65534
```

Therefore, the following files (located under `kustomize/monitoring`) should be modified and/or
patched (e.g. using additional overlays) as needed to ensure the `securityContext` is properly
defined for your Kubernetes environment:

- `deploy-alertmanager.yaml`
- `deploy-grafana.yaml`
- `deploy-prometheus.yaml`

And to modify the configuration for the various storage resources (i.e. PersistentVolumeClaims)
created by the PGO Monitoring installer, the `kustomize/monitoring/pvcs.yaml` file can also
be modified.

Additionally, it is also possible to further customize the configuration for the various components
comprising the PGO Monitoring stack (Grafana, Prometheus and/or AlertManager) by modifying the
following configuration resources:

- `alertmanager-config.yaml`
- `alertmanager-rules-config.yaml`
- `grafana-datasources.yaml`
- `prometheus-config.yaml`

Finally, please note that the default username and password for Grafana can be updated by
modifying the Grafana Secret in file `kustomize/monitoring/grafana-secret.yaml`.

## Install

Once the Kustomize project has been modified according to your specific needs, PGO Monitoring can
then be installed using `kubectl` and Kustomize:

```shell
kubectl apply -k kustomize/monitoring
```

## Uninstall

And similarly, once PGO Monitoring has been installed, it can uninstalled using `kubectl` and
Kustomize:

```shell
kubectl delete -k kustomize/monitoring
```
