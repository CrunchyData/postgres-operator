---
title: "Upgrade"
date:
draft: false
weight: 33
---

# Overview

Upgrading to a new version of PGO is typically as simple as following the various installation
guides defined within the PGO documentation:

- [PGO Kustomize Install]({{< relref "./kustomize.md" >}})
- [PGO Helm Install]({{< relref "./helm.md" >}})

However, when upgrading to or from certain versions of PGO, extra steps may be required in order
to ensure a clean and successful upgrade.

This section provides detailed instructions for upgrading PGO 5.x using Kustomize or Helm, along with information for upgrading from PGO v4 to PGO v5.

{{% notice info %}}
Depending on version updates, upgrading PGO may automatically rollout changes to managed Postgres clusters. This could result in downtime--we cannot guarantee no interruption of service, though PGO attempts graceful incremental rollouts of affected pods, with the goal of zero downtime.
{{% /notice %}}

## Upgrading PGO 5.x

- [PGO Kustomize Upgrade]({{< relref "./kustomize.md" >}})
- [PGO Helm Upgrade]({{< relref "./helm.md" >}})

## Upgrading from PGO v4 to PGO v5

- [V4 to V5 Upgrade Methods]({{< relref "./v4tov5" >}})
