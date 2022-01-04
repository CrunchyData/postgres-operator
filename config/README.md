<!--
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
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
