---
title: "Namespace"
date:
draft: false
weight: 6
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
the namespaces which will be *watched* by the Operator.

The format of the NAMESPACE value is modeled after the following
document:

https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/operatorgroups.md


### OwnNamespace Example

Prior to version 4.0, the Operator was deployed into
a single namespace and Postgres Clusters created by it were
created in that same namespace. 

To achieve that same deployment model you would use
variable settings as follows:

    export PGO_OPERATOR_NAMESPACE=pgo
    export NAMESPACE=pgo

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

### AllNamespaces Example

To have the Operator deployed into its own namespace but
create Postgres Clusters into any target namespace the
variables would be as follows:

    export PGO_OPERATOR_NAMESPACE=pgo
    export NAMESPACE=

Here the empty string value represents *all* namespaces.

![Reference](/Namespace-Single-Any.png)


### RBAC

To support multiple namespace watching, the Operator deployment
process changes somewhat from 3.X releases.

Each namespace to be watched requires its own copy of the 
the following resources for working with the Operator:

    serviceaccount/pgo-backrest
    secret/pgo-backrest-repo-config
    role/pgo-role
    rolebinding/pgo-role-binding
    role/pgo-backrest-role
    rolebinding/pgo-backrest-role-binding

When you run the install-rbac.sh script, it iterates through the 
list of namespaces to be watched and creates these resources into 
each of those namespaces.

If you need to add a new namespace that the Operator will watch
after an initial execution of install-rbac.sh, you will need to run 
the following for each new namespace:

    create-target-rbac.sh YOURNEWNAMESPACE $PGO_OPERATOR_NAMESPACE

The example deployment creates the following RBAC structure
on your Kube system after running the install scripts:

![Reference](/Operator-RBAC-Diagram.png)


## pgo Clients and Namespaces

The *pgo* CLI now is required to identify which namespace it
wants to use when issuing commands to the Operator.

Users of *pgo* can either create a PGO_NAMESPACE environment
variable to set the namespace in a persistent manner or they
can specify it on the *pgo* command line using the *--namespace*
flag.

If a pgo request doe not contain a valid namespace the request
will be rejected.


