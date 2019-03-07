---
title: "Design"
date:
draft: false
weight: 4
---

## Operator Namespaces

The Operator itself knows which namespace it is running
within by referencing the PGO_NAMESPACE environment variable
at startup time from within its Deployment definition.  

The CO_NAMESPACE environment variable a user sets in their 
.bashrc file is used to determine what namespace the Operator 
is deployed into.  The CO_NAMESPACE variable is mapped into
the PGO_NAMESPACE variable that the Operator references.

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


### Single Namespace Example

Prior to version 4.0, the Operator was deployed into
a single namespace and Postgres Clusters created by it were
created in that same namespace. 

To achieve that same deployment model you would use
variable settings as follows:

    export CO_NAMESPACE=pgo
    export NAMESPACE=pgo

![Reference](/Namespace-Single.png)

### Single Operator Namespace and Single Target Namespace

To have the Operator deployed into its own namespace but 
create Postgres Clusters into a different namespace the
variables would be as follows:

    export CO_NAMESPACE=pgo
    export NAMESPACE=pgouser1

![Reference](/Namespace-Single-Single.png)

### Single Operator Namespace and Multiple Target Namespaces

To have the Operator deployed into its own namespace but
create Postgres Clusters into more than one namespace the
variables would be as follows:

    export CO_NAMESPACE=pgo
    export NAMESPACE=pgouser1,pgouser2

![Reference](/Namespace-Single-Multiple.png)

### Single Operator Namespace and Any Target Namespaces

To have the Operator deployed into its own namespace but
create Postgres Clusters into any target namespace the
variables would be as follows:

    export CO_NAMESPACE=pgo
    export NAMESPACE=

Here the empty string value represents *all* namespaces.

![Reference](/Namespace-Single-Any.png)


### RBAC

To support multiple namespace watching, the Operator deployment
process changes a bit from 3.X releases.

Each namespace to be watch requires its own copy of the 
pgo-backrest-repo-config Secret.  When you run the install-rbac.sh
script, it now iterates through the list of namespaces to be
watched and creates this Secret into each of those namespaces.

If after an initial execution of install-rbac.sh, you need to add a 
new namespace, you will need to run the create-pgo-backrest-ssh-secret.sh 
script for that new namespace.

## pgo Clients and Namespaces

The *pgo* CLI now is required to identify which namespace it
wants to use when issuing commands to the Operator.

Users of *pgo* can either create a PGO_NAMESPACE environment
variable to set the namespace in a persistent manner or they
can specify it on the *pgo* command line using the *--namespace*
flag.

If a pgo request doe not contain a valid namespace the request
will be rejected.


