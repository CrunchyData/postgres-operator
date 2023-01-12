---
title: "Upgrading PGO v5 Using Helm"
date:
draft: false
weight: 70
---

Once PGO v5 has been installed with Helm, it can then be upgraded using the `helm upgrade` command.
However, before running the `upgrade` command, any CustomResourceDefinitions (CRDs) must first be
manually updated (this is specifically due to a [design decision in Helm v3][helm-crd-limits],
in which any CRDs in the Helm chart are only applied when using the `helm install` command).

[helm-crd-limits]: https://helm.sh/docs/topics/charts/#limitations-on-crds

If you would like, before upgrading the CRDs, you can review the changes with
`kubectl diff`. They can be verbose, so a pager like `less` may be useful:

```shell
kubectl diff -f helm/install/crds | less
```

Use the following command to update the CRDs using
[server-side apply](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
_before_ running `helm upgrade`. The `--force-conflicts` flag tells Kubernetes that you recognize
Helm created the CRDs during `helm install`.

```shell
kubectl apply --server-side --force-conflicts -f helm/install/crds
```

Then, perform the upgrade using Helm:

```shell
helm upgrade <name> -n <namespace> helm/install
```
PGO versions earlier than v5.4.0 include a pgo-upgrade deployment. When upgrading to v5.4.x, users
should expect the pgo-upgrade deployment to be deleted automatically.
