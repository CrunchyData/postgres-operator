
---
title: "Configuration Resources"
Latest Release: 3.5.3 {docdate}
draft: false
weight: 30
---

The operator is template-driven; this makes it simple to configure both the client and the operator.

## conf Directory
The Operator is configured with a collection of files found in the *conf* directory.  These configuration files are deployed to your Kubernetes cluster when the Operator is deployed.  Changes made to any of these configuration files currently require a redeployment of the Operator on the Kubernetes cluster.

The server components of the Operator include Role Based Access Control resources which need to be created a single time by a Kubernetes cluster-admin user.  See the Installation section for details on installing a Postgres Operator server.

## conf/postgres-operator/pgo.yaml
The *pgo.yaml* file sets many different Operator configuration settings and is described in the [pgo.yaml configuration]({{< ref "pgo-yaml-configuration.md" >}}) documentation section.


The *pgo.yaml* file is deployed along with the other Operator configuration files when you run:

    make deployoperator

## conf/postgres-operator Directory
Files within the *conf/postgres-operator* directory contain various templates that are used by the Operator when creating Kubernetes resources.  In an advanced Operator deployment, administrators can modify these templates to add their own custom meta-data or make other changes to influence the Resources that get created on your Kubernetes cluster by the Operator.

## conf/postgres-operator/cluster
Files within this directory are used specifically when creating PostgreSQL Cluster resources. Sidecar components such as pgBouncer and pgPool II templates are also located within this directory.

As with the other Operator templates, administrators can make custom changes to this set of templates to add custom features or metadata into the Resources created by the Operator.

## Security
Security configuration is described in the [Security](/security) section of this documentation.
