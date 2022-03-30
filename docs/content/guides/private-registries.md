---
title: "Private Registries"
date:
draft: false
weight: 200
---

PGO, the open source Postgres Operator, can use containers that are stored in private registries.
There are a variety of techniques that are used to load containers from private registries,
including [image pull secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).
This guide will demonstrate how to install PGO and deploy a Postgres cluster using the
[Crunchy Data Customer Portal](https://access.crunchydata.com/) registry as an example.

## Create an Image Pull Secret

The Kubernetes documentation provides several methods for creating
[image pull secrets](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/).
You can choose the method that is most appropriate for your installation. You will need to create
image pull secrets in the namespace that PGO is deployed and in each namespace where you plan to
deploy Postgres clusters.

For example, to create an image pull secret for accessing the Crunchy Data Customer Portal image
registry in the `postgres-operator` namespace, you can execute the following commands:

```shell
kubectl create ns postgres-operator

kubectl create secret docker-registry crunchy-regcred -n postgres-operator \
  --docker-server=registry.crunchydata.com \
  --docker-username=<YOUR USERNAME> \
  --docker-email=<YOUR EMAIL> \
  --docker-password=<YOUR PASSWORD>
```

This creates an image pull secret named `crunchy-regcred` in the `postgres-operator` namespace.

## Install PGO from a Private Registry

To [install PGO]({{< relref "installation/_index.md" >}}) from a private registry, you will need to
set an image pull secret on the installation manifest.

For example, to set up an image pull secret using the [Kustomize install method]({{< relref "installation/_index.md" >}})
to install PGO from the [Crunchy Data Customer Portal](https://access.crunchydata.com/), you can set
the following in the `install/bases/kustomization.yaml` manifest:

```yaml
images:
- name: postgres-operator
  newName: {{< param operatorRepositoryPrivate >}}
  newTag: {{< param postgresOperatorTag >}}

patchesJson6902:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: pgo
    patch: |-
      - op: remove
        path: /spec/selector/matchLabels/app.kubernetes.io~1name
      - op: remove
        path: /spec/selector/matchLabels/app.kubernetes.io~1version
      - op: add
        path: /spec/template/spec/imagePullSecrets
        value:
          - name: crunchy-regcred
```

If you are using a version of `kubectl` prior to `v1.21.0`, you will have to create an explicit
patch file named `install-ops.yaml`:

```yaml
- op: remove
  path: /spec/selector/matchLabels/app.kubernetes.io~1name
- op: remove
  path: /spec/selector/matchLabels/app.kubernetes.io~1version
- op: add
  path: /spec/template/spec/imagePullSecrets
  value:
    - name: crunchy-regcred
```

and modify the manifest to be the following:

```yaml
images:
- name: postgres-operator
  newName: {{< param operatorRepositoryPrivate >}}
  newTag: {{< param postgresOperatorTag >}}

patchesJson6902:
  - target:
      group: apps
      version: v1
      kind: Deployment
      name: pgo
    path: install-ops.yaml
```

You can then install PGO from the private registry using the standard installation procedure, e.g.:

```shell
kubectl apply -k kustomize/install
```

## Deploy a Postgres cluster from a Private Registry

To deploy a Postgres cluster using images from a private registry, you will need to set the value of
`spec.imagePullSecrets` on a `PostgresCluster` custom resource.

For example, to deploy a Postgres cluster using images from the [Crunchy Data Customer Portal](https://access.crunchydata.com/)
with an image pull secret in the `postgres-operator` namespace, you can use the following manifest:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  imagePullSecrets:
    - name: crunchy-regcred
  image: {{< param imageCrunchyPostgresPrivate >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - name: instance1
      dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrestPrivate >}}
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi
```
