---
title: "Kubernetes RBAC"
date:
draft: false
weight: 7
---

## Kubernetes RBAC

Install the requisite PostgreSQL Operator RBAC resources, *as a Kubernetes cluster admin user*,  by running a Makefile target:

    make installrbac


This script creates the following RBAC resources on your Kubernetes cluster:

| Setting |Definition  |
|---|---|
| Custom Resource Definitions (crd.yaml) | pgbackups|
|  | pgclusters|
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


Note that the cluster role bindings have a naming convention of pgopclusterbinding-$PGO_OPERATOR_NAMESPACE and pgopclusterbindingcrd-$PGO_OPERATOR_NAMESPACE.  

The PGO_OPERATOR_NAMESPACE environment variable is added to make each cluster role binding name unique and to support more than a single PostgreSQL Operator being deployed on the same Kubernertes cluster.


