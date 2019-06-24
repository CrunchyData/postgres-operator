
---
title: "Configuration Resources"
Latest Release: 4.0.1 {docdate}
draft: false
weight: 2
---

The operator is template-driven; this makes it simple to configure both the client and the operator.

## conf Directory

The Operator is configured with a collection of files found in the *conf* directory.  These configuration files are deployed to your Kubernetes cluster when the Operator is deployed.  Changes made to any of these configuration files currently require a redeployment of the Operator on the Kubernetes cluster.

The server components of the Operator include Role Based Access Control resources which need to be created a single time by a Kubernetes cluster-admin user.  See the Installation section for details on installing a Postgres Operator server.

The configuration files used by the Operator are found in 2 places:
 * the pgo-config ConfigMap in the namespace the Operator is running in
 * or, a copy of the configuration files are also included by default into the Operator container images themselves to support a very simplistic deployment of the Operator

If the pgo-config ConfigMap is not found by the Operator, it will use
the configuration files that are included in the Operator container
images.

The container included set of configuration files use the most
basic setting values and the image versions of the Operator itself
with the latest Crunchy Container image versions.  The storage
configurations are determined by using the default storage
class on the system you are deploying the Operator into, the
default storage class is one that is labeled as follows:

    pgo-default-sc=true 

If no storage class has that label, then the first storage class
found on the system will be used.  If no storage class is found
on the system, the containers will not run and produce an error
in the log.

## conf/postgres-operator/pgo.yaml
The *pgo.yaml* file sets many different Operator configuration settings and is described in the [pgo.yaml configuration]({{< ref "pgo-yaml-configuration.md" >}}) documentation section.


The *pgo.yaml* file is deployed along with the other Operator configuration files when you run:

    make deployoperator

## conf/postgres-operator Directory

Files within the *conf/postgres-operator* directory contain various templates that are used by the Operator when creating Kubernetes resources.  In an advanced Operator deployment, administrators can modify these templates to add their own custom meta-data or make other changes to influence the Resources that get created on your Kubernetes cluster by the Operator.

Files within this directory are used specifically when creating PostgreSQL Cluster resources. Sidecar components such as pgBouncer and pgPool II templates are also located within this directory.

As with the other Operator templates, administrators can make custom changes to this set of templates to add custom features or metadata into the Resources created by the Operator.

## Security

Setting up pgo users and general security configuration is described in the [Security](/security) section of this documentation.

## Local pgo CLI Configuration

You can specify the default namespace you want to use by
setting the PGO_NAMESPACE environment variable locally
on the host the pgo CLI command is running.

    export PGO_NAMESPACE=pgouser1

When that variable is set, each command you issue with *pgo* will
use that namespace unless you over-ride it using the *--namespace*
command line flag.

    pgo show cluster foo --namespace=pgouser2

## Namespace Configuration

The Design [Design](/Design) section of this documentation talks further about
the use of namespaces within the Operator and configuring different
deployment models of the Operator.

