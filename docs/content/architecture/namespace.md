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
determining whether or not certain namespaces capabilities are enabled for the Operator
installation. When the PostgreSQL Operator is run, the Kubernetes environment is inspected to
determine what cluster roles are currently assigned to the `pgo-operator` `ServiceAccount`
(i.e. the `ServiceAccount` running the Pod the PostgreSQL Operator is deployed within).  Based
on the `ClusterRoles` identified, one of the namespace operating modes described below will be
enabled for the Operator installation.  Please consult the installation guides for the various
installation methods available to determine the settings required to install the `ClusterRoles`
required for each mode.

### `dynamic`

Enables full dynamic namespace capabilities, in which the Operator can create, delete and update
any namespaces within the Kubernetes cluster, while then also having the ability to create the
`Roles`, `RoleBindings` and `ServiceAccounts` within those namespaces as required for the Operator
to create PostgreSQL clusters.  Additionally, while  in this mode the Operator can listen for
namespace events (e.g. namespace additions, updates and deletions), and then create or remove
controllers for various namespaces as those namespaces are added or removed from the Kubernetes
cluster and/or Operator install.  The mode therefore allows the Operator to dynamically respond
to namespace events in the cluster, and then interact with those namespaces as required to manage
PostgreSQL clusters within them.

The following represents the `ClusterRole` required for the `dynamic` mode to be enabled:

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
  - apiGroups:
      - ''
    resources:
      - serviceaccounts
    verbs:
    - get
    - create
    - delete
  - apiGroups:
      - rbac.authorization.k8s.io
    resources:
      - roles
    verbs:
      - get
      - create
      - delete
      - bind
      - escalate
  - apiGroups:
      - rbac.authorization.k8s.io
    resources:
      - rolebindings
    verbs:
      - get
      - create
      - delete
```

### `readonly`

In this mode the PostgreSQL Operator is still able to listen for  namespace events within the
Kubernetetes cluster, and then create and run and/or remove controllers as namespaces are added,
updated and deleted.  However, while in this mode the Operator is unable to create, delete or
update namespaces itself, nor can it create the RBAC it requires in any of those namespaces to
create PostgreSQL clusters.  Therefore, while in a `readonly` mode namespaces must be
preconfigured with the proper RBAC, since the Operator cannot create the RBAC itself (unless
it has permission to do so in its ServiceAccount, as described further on in this document).

The following represents the `ClusterRole` required for the `readonly` mode to be enabled:

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

Disables namespace capabilities within the Operator altogether.  While in this mode the Operator
will simply attempt to work with the target namespaces specified during installation.  If no
target namespaces are specified, then the Operator will be configured to work within the namespace
in which it is deployed.  As with `readonly`, while in this mode namespaces must be pre-configured
with the proper RBAC, since the Operator cannot create the RBAC itself.  Additionally, in the event
that target namespaces are deleted or the required RBAC within those namespaces are modified, the
Operator will need to be re-deployed to ensure it no longer attempts to listen for events in those
namespaces (specifically because while in this mode, the Operator is unable to listen for namespace
events, and therefore cannot detect whether to watch or stop watching namespaces as they are added
and/or removed).

Mode `disabled` is enabled when no `ClusterRoles` have been installed.

## Dynamic RBAC Creation for `readonly` and `disabled` Namespace Operating Modes

_Please note that this section is only applicable when using the `readonly` or `disabled` namespace
operating modes._

As described in the Namespace Operating Mode section above, when using either the `readonly` or 
`disabled` operating modes, all target name namespaces must be pre-configured with the proper RBAC
(ServiceAccounts, Roles and RoleBindings) as required for the PostgreSQL Operator to create PostgreSQL
clusters within those namespaces.  However,  this can done in one of the following two ways:

1. Assign the `postgres-operator` ServiceAccount the permissions required to create the RBAC it requires
    within the namespace to create PostgreSQL clusters.  This is specifically be done by creating the following
    Role and RoleBinding within the target namespace:

        ---
        kind: Role
        apiVersion: rbac.authorization.k8s.io/v1
        metadata:
          name: pgo-local-ns
          namespace: <target-namespace>
        rules:
          - apiGroups:
              - ''
            resources:
              - serviceaccounts
            verbs:
              - get
              - create
              - delete
          - apiGroups:
              - rbac.authorization.k8s.io
            resources:
              - roles
              - rolebindings
            verbs:
              - get
              - create
              - delete
              - bind
              - escalate
        ---
        apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        metadata:
          name: pgo-local-ns
          namespace: $TARGET_NAMESPACE
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: pgo-local-ns
        subjects:
        - kind: ServiceAccount
          name: postgres-operator
          namespace: <postgresql-operator-namespace>

    When the PostgreSQL Operator detects that it has the permissions defined in the `pgo-local-ns`
    during initialization, it will create any RBAC it requires within that namespace  (recreating 
    it if it already exists).  And if using the `readonly` namespace operating mode, the operator
    will also create/recreate the RBAC for a namespace when it detects that a new target namespace
    has been created.

2. Manually create the ServiceAccounts, Roles and RoleBindings required for the Operator to create PostgreSQL clusters in a target namespace.

All installation methods provided for installing the PostgreSQL Operator include configuration settings for determining whether or not
the PostgreSQL Operator is assigned the permissions needed to dynamically create RBAC within a target namespace.  Therefore, when using the
`readonly` and `disabled` namespace operating modes, please consult the proper installation guide for guidance on the proper configuration
settings.

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
