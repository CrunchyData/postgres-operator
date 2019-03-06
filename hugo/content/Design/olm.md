---
title: "Design"
date:
draft: false
weight: 4
---

## Operator Lifecycle Management

The Postgres Operator supports Redhats OLM (operator lifecycle manager)
to a degree starting with pgo 4.0.

### Operator Namespace

The Operator itself knows which namespace it is running
within by referencing the PGO_NAMESPACE environment variable
at startup time from within its Deployment definition.  

The CO_NAMESPACE environment variable a user sets in their 
.bashrc file is still used to determine what namespace the Operator 
is deployed into and should be the same as what PGO_NAMESPACE resolves
into.

The new NAMESPACE environment variable a user sets determines
which namespaces the Operator will watch.

### Namespace Watching

The Operator at startup time determines which namespaces it will
service based on what it finds in the NAMESPACE environment variable
that is passed into the Operator containers within the deployment.json file.

The NAMESPACE variable can hold different values which determine
the namespaces which will be *watched* by the Operator.

#### Examples

To specify that *all* namespaces are watched:
 * export NAMESPACE=

To specify a single namespace be watched:
 * (single namespace) export NAMESPACE=example1

To specify multiple namespaces be watched:
 * (multiple namespaces) export NAMESPACE=example1,example2

The format of the NAMESPACE value is modeled after the following
document:

https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/operatorgroups.md

The following diagram depicts the various deployment models:

![Reference](/OperatorReferenceDiagram.png)


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

## Operator Hub

The Operator shows up on the Redhat Operator Hub at the following
location:

https://www.operatorhub.io/operator/postgres-operator.v3.5.0

