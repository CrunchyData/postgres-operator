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
at startup time.

### Namespace Servicing

The Operator at startup time determines which namespaces it will
service based on what it finds in the NAMESPACE environment variable
that is passed into the Operator containers within the deployment.json file.

The NAMESPACE variable can hold different values which determine
the namespaces which will be *watched* by the Operator.

The following examples are supported:

 * (all namespace) export NAMESPACE=
 * (single namespace) export NAMESPACE=example1
 * (multiple namespaces) export NAMESPACE=example1,example2

The format of the NAMESPACE value is modeled after the following
document:

https://github.com/operator-framework/operator-lifecycle-manager/blob/master/Documentation/design/operatorgroups.md

The following diagram depicts the various deployment models:

![Reference](/OperatorReferenceDiagram.png)


## Operator Hub

The Operator shows up on the Redhat Operator Hub at the following
location:

https://www.operatorhub.io/operator/postgres-operator.v3.5.0

