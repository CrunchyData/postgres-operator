---
title: "Installation of PostgreSQL Operator RBAC"
date:
draft: false
weight: 7
---

## Installation of PostgreSQL Operator RBAC

For a list of the RBAC required to install the PostgreSQL Operator, please view the [`postgres-operator.yml`](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml) file:

[https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml)

The first step is to install the PostgreSQL Operator RBAC configuration.  This can be accomplished  by running:

    make installrbac

This script will install the PostreSQL Operator Custom Resource Definitions, CRD’s and creates the following RBAC resources on your Kubernetes cluster:

| Setting |Definition  |
|---|---|
| Custom Resource Definitions | pgclusters|
|  | pgpolicies|
|  | pgreplicas|
|  | pgtasks|
|  | pgupgrades|
| Cluster Roles (cluster-roles.yaml) | pgopclusterrole|
|  | pgopclusterrolecrd|
| Cluster Role Bindings (cluster-roles-bindings.yaml) | pgopclusterbinding|
|  | pgopclusterbindingcrd|
| Service Account (service-accounts.yaml) | postgres-operator|
| | pgo-backrest|
| Roles (rbac.yaml) | pgo-role|
| | pgo-backrest-role|
|Role Bindings  (rbac.yaml) | pgo-backrest-role-binding|
| | pgo-role-binding|

Note that the cluster role bindings have a naming convention of pgopclusterbinding-$PGO_OPERATOR_NAMESPACE and pgopclusterbindingcrd-$PGO_OPERATOR_NAMESPACE.  The PGO_OPERATOR_NAMESPACE environment variable is added to make each cluster role binding name unique and to support more than a single PostgreSQL Operator being deployed on the same Kubernertes cluster.

Also, the specific Cluster Roles installed depends on the Namespace Mode enabled via the `PGO_NAMESPACE_MODE` environment variable when running `make installrbac`.  Please consult the [Namespace documentation](/architecture/namespace/) for more information regarding the Namespace Modes available, including the specific `ClusterRoles` required to enable each mode.
