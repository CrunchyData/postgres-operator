---
title: "Namespace Management"
date:
draft: false
weight: 400
---

# Kubernetes Namespaces and the PostgreSQL Operator

The PostgreSQL Operator leverages Kubernetes Namespaces to react to actions
taken within a Namespace to keep its PostgreSQL clusters deployed as requested.
Early on, the PostgreSQL Operator was scoped to a single namespace and would
only watch PostgreSQL clusters in that Namspace, but since version 4.0, it has
been expanded to be able to manage PostgreSQL clusters across multiple
namespaces.

The following provides more information about how the PostgreSQL Operator works
with namespaces, and presents several deployment patterns that can be used to
deploy the PostgreSQL Operator.

## Namespace Operating Modes

The PostgreSQL Operator can be run with various Namespace Operating Modes, with each mode
determining whether or not certain namespace capabilities are enabled for the PostgreSQL Operator
installation. When the PostgreSQL Operator is run, the Kubernetes environment is inspected to
determine what cluster roles are currently assigned to the `pgo-operator` `ServiceAccount`
(i.e. the `ServiceAccount` running the Pod the PostgreSQL Operator is deployed within).  Based
on the `ClusterRoles` identified, one of the namespace operating modes described below will be
enabled for the [PostgreSQL Operator Installation]({{< relref "installation" >}}). Please consult
the [installation](({{< relref "installation" >}})) section for more information on the available
settings.

### `dynamic`

Enables full dynamic namespace capabilities, in which the Operator can create, delete and update
any namespaces within a Kubernetes cluster.  With `dynamic` mode enabled, the PostgreSQL Operator
can respond to namespace events in a Kubernetes cluster, such as when a namespace is created, and
take an appropriate action, such as adding the PostgreSQL Operator controllers for the newly
created namespace.

The following defines the namespace permissions required for the `dynamic` mode to be enabled:

```yaml
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pgo-cluster-role
rules:
  - apiGroups:
      - ''
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
      - create
      - update
      - delete
```

### `readonly`

In `readonly` mode, the PostgreSQL Operator is still able to listen to namespace events within a
Kubernetes cluster, but it can no longer modify (create, update, delete) namespaces. For example,
if a Kubernetes administrator creates a namespace, the PostgreSQL Operator can respond and create
controllers for that namespace.

The following defines the namespace permissions required for the `readonly` mode to be enabled:

```yaml
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pgo-cluster-role
rules:
  - apiGroups:
      - ''
    resources:
      - namespaces
    verbs:
      - get
      - list
      - watch
```

### `disabled`

`disabled` mode disables namespace capabilities namespace capabilities within the PostgreSQL
Operator altogether.  While in this mode the PostgreSQL Operator will simply attempt to work with
the target namespaces specified during installation.  If no target namespaces are specified, then
the Operator will be configured to work within the namespace in which it is deployed.  Since the
Operator is unable to dynamically respond to namespace events in  the cluster, in the event that
target namespaces are deleted or new target namespaces need to be added, the PostgreSQL Operator
will need to be re-deployed.

Please note that it is important to redeploy the PostgreSQL Operator following the deletion of a
target namespace to ensure it no longer attempts to listen for events in that namespace.

The `disabled` mode is enabled the when the PostgreSQL Operator has not been assigned namespace
permissions.

## RBAC Reconciliation

By default, the PostgreSQL Operator will attempt to reconcile RBAC resources (ServiceAccounts,
Roles and RoleBindings) within each namespace configured for the PostgreSQL Operator installation.
This allows the PostgreSQL Operator to create, update and delete the various RBAC resources it
requires in order to properly create and manage PostgreSQL clusters within each targeted namespace
(this includes self-healing RBAC resources as needed if removed and/or misconfigured).

In order for RBAC reconciliation to function properly, the PostgreSQL Operator ServiceAccount must
be assigned a certain set of permissions.  While the PostgreSQL Operator is not concerned with
exactly how it has been assigned the permissions required to reconcile RBAC in each target
namespace, the various [installation methods]({{< relref "installation" >}}) supported by the
PostgreSQL Operator install a recommended set permissions based on the specific Namespace Operating
Mode enabled (see section [Namespace Operating Modes]({{< relref "#namespace-operating-modes" >}})
above for more information regarding the various Namespace Operating Modes available).

The following section defines the recommended set of permissions that should be assigned to the
PostgreSQL Operator ServiceAccount in order to ensure proper RBAC reconciliation based on the
specific Namespace Operating Mode enabled.  Please note that each PostgreSQL Operator installation
method handles the initial configuration and setup of the permissions shown below based on the
Namespace Operating Mode configured during installation.

### `dynamic` Namespace Operating Mode

When using the `dynamic` Namespace Operating Mode, it is recommended that the PostgreSQL Operator
ServiceAccount be granted permissions to manage RBAC inside any namespace in the Kubernetes cluster
via a ClusterRole.  This allows for a fully-hands off approach to managing RBAC within each
targeted namespace space.  In other words, as namespaces are added and removed post-installation of
the PostgreSQL Operator (e.g. using `pgo create namespace` or `pgo delete namespace`), the Operator
is able to automatically reconcile RBAC in those namespaces without the need for any external
administrative action and/or resource creation.

The following defines ClusterRole permissions that are assigned to the PostgreSQL Operator
ServiceAccount via the various Operator installation methods when the `dynamic` Namespace Operating
Mode is configured:

```yaml
---
kind: ClusterRole
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pgo-cluster-role
rules:
  - apiGroups:
      - ''
    resources:
      - serviceaccounts
    verbs:
      - get
      - create
      - update
      - delete
  - apiGroups:
      - rbac.authorization.k8s.io
    resources:
      - roles
      - rolebindings
    verbs:
      - get
      - create
      - update
      - delete
  - apiGroups:
      - ''
    resources:
      - configmaps
      - endpoints
      - pods
      - pods/exec
      - secrets
      - services
      - persistentvolumeclaims
    verbs:
      - get
      - list
      - watch
      - create
      - patch
      - update
      - delete
      - deletecollection
  - apiGroups:
    - ''
    resources:
      - pods/log
    verbs:
      - get
      - list
      - watch
  - apiGroups:
      - apps
    resources:
      - deployments
      - replicasets
    verbs:
      - get
      - list
      - watch
      - create
      - patch
      - update
      - delete
      - deletecollection
  - apiGroups:
      - batch
    resources:
      - jobs
    verbs:
      - get
      - list
      - watch
      - create
      - patch
      - update
      - delete
      - deletecollection
  - apiGroups:
      - crunchydata.com
    resources:
      - pgclusters
      - pgpolicies
      - pgreplicas
      - pgtasks
    verbs:
      - get
      - list
      - watch
      - create
      - patch
      - update
      - delete
      - deletecollection
```

### `readonly` & `disabled` Namespace Operating Modes

When using the `readonly` or `disabled` Namespace Operating Modes, it is recommended that the
PostgreSQL Operator ServiceAccount be granted permissions to manage RBAC inside of any configured
namespaces using local Roles within each targeted namespace.  This means that as new namespaces
are added and removed post-installation of the PostgreSQL Operator, an administrator must manually
assign the PostgreSQL Operator ServiceAccount the permissions it requires within each target
namespace in order to successfully reconcile RBAC within those namespaces.

The following defines the permissions that are assigned to the PostgreSQL Operator ServiceAccount
in each configured namespace via the various Operator installation methods when the `readonly` or
`disabled` Namespace Operating Modes are configured:

```yaml
---
kind: Role
apiVersion: rbac.authorization.k8s.io/v1
metadata:
  name: pgo-local-ns
  namespace: targetnamespace
rules:
  - apiGroups:
      - ''
    resources:
      - serviceaccounts
    verbs:
      - get
      - create
      - update
      - delete
  - apiGroups:
      - rbac.authorization.k8s.io
    resources:
      - roles
      - rolebindings
    verbs:
      - get
      - create
      - update
      - delete
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: pgo-target-role
  namespace: targetnamespace
rules:
- apiGroups:
    - ''
  resources:
    - configmaps
    - endpoints
    - pods
    - pods/exec
    - pods/log
    - replicasets
    - secrets
    - services
    - persistentvolumeclaims
  verbs:
    - get
    - list
    - watch
    - create
    - patch
    - update
    - delete
    - deletecollection
- apiGroups:
    - apps
  resources:
    - deployments
  verbs:
    - get
    - list
    - watch
    - create
    - patch
    - update
    - delete
    - deletecollection
- apiGroups:
    - batch
  resources:
    - jobs
  verbs:
    - get
    - list
    - watch
    - create
    - patch
    - update
    - delete
    - deletecollection
- apiGroups:
    - crunchydata.com
  resources:
    - pgclusters
    - pgpolicies
    - pgtasks
    - pgreplicas
  verbs:
    - get
    - list
    - watch
    - create
    - patch
    - update
    - delete
    - deletecollection
```

### Disabling RBAC Reconciliation

In the event that the reconciliation behavior discussed above is not desired, it can be fully
disabled by setting `DisableReconcileRBAC` to `true` in the `pgo.yaml` configuration file.  When
reconciliation is disabled using this setting, the PostgreSQL Operator will not attempt to
reconcile RBAC in any configured namespace.  As a result, any RBAC required by the PostreSQL
Operator a targeted namespace must be manually created by an administrator.

Please see the the
[`pgo.yaml` configuration guide]({{< relref "configuration/pgo-yaml-configuration.md" >}}), as well
as the documentation for the various [installation methods]({{< relref "installation" >}})
supported by the PostgreSQL Operator, for guidance on how to properly configure this setting and
therefore disable RBAC reconciliation.

## Namespace Deployment Patterns

There are several different ways the PostgreSQL Operator can be deployed in
Kubernetes clusters with respect to Namespaces.

### One Namespace: PostgreSQL Operator + PostgreSQL Clusters

![PostgreSQL Operator Own Namespace Deployment](/images/namespace-own.png)

This patterns is great for testing out the PostgreSQL Operator in development
environments, and can also be used to keep your entire PostgreSQL workload
within a single Kubernetes Namespace.

This can be set up with the `disabled` Namespace mode.

### Single Tenant: PostgreSQL Operator Separate from PostgreSQL Clusters

![PostgreSQL Operator Single Namespace Deployment](/images/namespace-single.png)

The PostgreSQL Operator can be deployed into its own namespace and manage
PostgreSQL clusters in a separate namespace.

This can be set up with either the `readonly` or `dynamic` Namespace modes.

### Multi Tenant: PostgreSQL Operator Managing PostgreSQL Clusters in Multiple Namespaces

![PostgreSQL Operator Multi Namespace Deployment](/images/namespace-multi.png)

The PostgreSQL Operator can manage PostgreSQL clusters across multiple
namespaces which allows for multi-tenancy.

This can be set up with either the `readonly` or `dynamic` Namespace modes.

## [`pgo` client]({{< relref "/pgo-client/_index.md" >}}) and Namespaces

The [`pgo` client]({{< relref "/pgo-client/_index.md" >}}) needs to be aware of
the Kubernetes Namespaces it is issuing commands to. This can be accomplish with
the `-n` flag that is available on most PostgreSQL Operator commands. For
example, to create a PostgreSQL cluster called `hippo` in the `pgo` namespace,
you would execute the following command:

```
pgo create cluster -n pgo hippo
```

For convenience, you can set the `PGO_NAMESPACE` environmental variable to
automatically use the desired namespace with the commands.

For example, to create a cluster named `hippo` in the `pgo` namespace, you could
do the following

```
# this export only needs to be run once per session
export PGO_NAMESPACE=pgo

pgo create cluster hippo
```
