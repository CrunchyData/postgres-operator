<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->


## Targets

- The `default` target installs the operator in the `postgres-operator`
  namespace and configures it to manage resources in all namespaces.

- The `singlenamespace` target installs the operator in the `postgres-operator`
  namespace and configures it to manage resources in that same namespace.

<!--
- The `dev` target installs the CRD and RBAC in the `postgres-operator`
  namespace while scaling an existing operator Deployment to zero.
-->


## Bases

- The `crd` base creates `CustomResourceDefinition`s that are managed by the
  operator.

- The `manager` base creates the `Deployment` that runs the operator. Do not
  run this as a target.

- The `rbac/cluster` base creates a `ClusterRole` that allows the operator to
  manage resources in all current and future namespaces.

- The `rbac/namespace` base creates a `Role` that limits the operator to
  managing a single namespace. Do not run this as a target.

<!--

| `kubectl` | `kustomize` |
|-----------|-------------|
| v1.16.0   | v2.0.3      |
| v1.17.0   | v2.0.3      |
| v1.18.0   | v2.0.3      |
| v1.19.0   | v2.0.3      |
| v1.20.0   | v2.0.3      |
| v1.21.0   | v4.0.5      |
| v1.22.0   | v4.2.0      |

-->
