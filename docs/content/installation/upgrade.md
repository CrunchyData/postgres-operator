---
title: "Upgrade"
date:
draft: false
weight: 50
---

# Overview

Upgrading to a new version of PGO is typically as simple as following the various installation
guides defined within the PGO documentation:

- [PGO Kustomize Install]({{< relref "./kustomize.md" >}})
- [PGO Helm Install]({{< relref "./helm.md" >}})

However, when upgrading to or from certain versions of PGO, extra steps may be required in order
to ensure a clean and successful upgrade.  This page will therefore document any additional
steps that must be completed when upgrading PGO.

## Upgrading from PGO 5.0.0 Using Kustomize

Starting with PGO 5.0.1, both the Deployment and ServiceAccount created when installing PGO via
the installers in the
[Postgres Operator examples repository](https://github.com/CrunchyData/postgres-operator-examples)
have been renamed from `postgres-operator` to `pgo`.  As a result of this change, if using
Kustomize to install PGO and upgrading from PGO 5.0.0, the following step must be completed prior
to upgrading.  This will ensure multiple versions of PGO are not installed and running concurrently
within your Kubernetes environment.

Prior to upgrading PGO, first manually delete the PGO 5.0.0 `postgres-operator` Deployment and
ServiceAccount:

```bash
kubectl -n postgres-operator delete deployment,serviceaccount postgres-operator
```

Then, once both the Deployment and ServiceAccount have been deleted, proceed with upgrading PGO
by applying the new version of the Kustomize installer:

```bash
kubectl apply -k kustomize/install/bases
```
