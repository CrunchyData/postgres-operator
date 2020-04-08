---
title: "Namespace Management"
date:
draft: false
weight: 400
---

## Operator Namespaces

The Operator itself knows which namespace it is running
within by referencing the PGO_OPERATOR_NAMESPACE environment variable
at startup time from within its Deployment definition.  

The PGO_OPERATOR_NAMESPACE environment variable a user sets in their
.bashrc file is used to determine what namespace the Operator
is deployed into.  The PGO_OPERATOR_NAMESPACE variable is referenced
by the Operator during deployment.

The .bashrc NAMESPACE environment variable a user sets determines
which namespaces the Operator will watch.

## Namespace Watching

The Operator at startup time determines which namespaces it will
service based on what it finds in the NAMESPACE environment variable
that is passed into the Operator containers within the deployment.json file.

The NAMESPACE variable can hold different values which determine
the namespaces which will be *watched* by the Operator.  If NAMESPACE is
empty, then the Operator will simply default to watching the namespace in
which it is deployed.

### OwnNamespace Example

Prior to version 4.0, the Operator was deployed into
a single namespace and Postgres Clusters created by it were
created in that same namespace.

To achieve that same deployment model you would use
variable settings as follows:

    export PGO_OPERATOR_NAMESPACE=pgo

![Reference](/Namespace-Single.png)

### SingleNamespace Example

To have the Operator deployed into its own namespace but
create Postgres Clusters into a different namespace the
variables would be as follows:

    export PGO_OPERATOR_NAMESPACE=pgo
    export NAMESPACE=pgouser1

![Reference](/Namespace-Single-Single.png)

### MultiNamespace Example

To have the Operator deployed into its own namespace but
create Postgres Clusters into more than one namespace the
variables would be as follows:

    export PGO_OPERATOR_NAMESPACE=pgo
    export NAMESPACE=pgouser1,pgouser2

![Reference](/Namespace-Single-Multiple.png)

### RBAC

To support multiple namespace watching, each namespace that the PostgreSQL
Operator watches requires its own copy of the following resources:

- `role/pgo-pg-role`
- `role/pgo-backrest-role`
- `role/pgo-target-role`
- `rolebinding/pgo-pg-role-binding`
- `rolebinding/pgo-backrest-role-binding`
- `rolebinding/pgo-target-role-binding`
- `serviceaccount/pgo-pg`
- `serviceaccount/pgo-backrest`
- `serviceaccount/pgo-target`

When you run the `install-rbac.sh` script, it iterates through the
list of namespaces to be watched and creates these resources into
each of those namespaces.

If you need to add a new namespace that the Operator will watch
after an initial execution of `install-rbac.sh`, you will need to run
the following for each new namespace:

    add-targeted-namespace.sh YOURNEWNAMESPACE

The example deployment creates the following RBAC structure
on your Kubernetes system after running the install scripts:

![Reference](/Operator-RBAC-Diagram.png)

## Namespace Operating Modes

The PostgreSQL Operator can be run with various Namespace Operating Modes, with each mode
determining whether or not certain namespaces capabilities are enabled for the Operator
installation. When the PosgreSQL Operator is run, the Kubernetes environment is inspected to 
determine what cluster roles are currently assigned to the `pgo-operator` `ServiceAccount` 
(i.e. the `ServiceAccount` running the Pod the PostgreSQL Operator is deployed within).  Based
on the `ClusterRoles` identified, one of the namespace operating modes described below will be
enabled for the Operator installation.  Please consult the installation guides for the various
installation methods available to determine the settings required to install the `ClusterRoles`
required for each mode.

### dynamic

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

### readonly

In this mode the PostgreSQL Operator is still able to listen for  namespace events within the 
Kubernetetes cluster, and then create and run and/or remove controllers as namespaces are added,
updated and deleted.  However, while in this mode the Operator is unable to create, delete or
update namespaces itself, nor can it create the RBAC it requires in any of those namespaces to
create PostgreSQL clusters.  Therefore, while in a `readonly` mode namespaces must be
pre-configured with the proper RBAC, since the Operator cannot create the RBAC itself.

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

### disabled

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

## pgo Clients and Namespaces

The *pgo* CLI now is required to identify which namespace it
wants to use when issuing commands to the Operator.

Users of *pgo* can either create a PGO_NAMESPACE environment
variable to set the namespace in a persistent manner or they
can specify it on the *pgo* command line using the *--namespace*
flag.

If a pgo request doe not contain a valid namespace the request
will be rejected.
