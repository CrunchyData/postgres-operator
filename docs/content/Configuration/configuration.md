---
title: "Configuration Resources"
draft: false
weight: 2
---

The operator is template-driven; this makes it simple to configure both the client and the operator.

## conf Directory

The Operator is configured with a collection of files found in the *conf* directory.  These configuration files are deployed to your Kubernetes cluster when the Operator is deployed.  Changes made to any of these configuration files currently require a redeployment of the Operator on the Kubernetes cluster.

The server components of the Operator include Role Based Access Control resources which need to be created a single time by a privileged Kubernetes user.  See the Installation section for details on installing a Postgres Operator server.

The configuration files used by the Operator are found in 2 places:
 * the pgo-config ConfigMap in the namespace the Operator is running in
 * or, a copy of the configuration files are also included by default into the Operator container images themselves to support a very simplistic deployment of the Operator

If the `pgo-config` ConfigMap is not found by the Operator, it will create a
`pgo-config` ConfigMap using the configuration files that are included in the
Operator container.

## conf/postgres-operator/pgo.yaml
The *pgo.yaml* file sets many different Operator configuration settings and is described in the [pgo.yaml configuration]({{< ref "pgo-yaml-configuration.md" >}}) documentation section.


The *pgo.yaml* file is deployed along with the other Operator configuration files when you run:

    make deployoperator

## Config Directory

Files within the [*PGO_CONF_DIR*](/developer-setup/) directory contain various templates that are used by the Operator when creating Kubernetes resources.  In an advanced Operator deployment, administrators can modify these templates to add their own custom meta-data or make other changes to influence the Resources that get created on your Kubernetes cluster by the Operator.

Files within this directory are used specifically when creating PostgreSQL Cluster resources. Sidecar components such as pgBouncer templates are also located within this directory.

As with the other Operator templates, administrators can make custom changes to this set of templates to add custom features or metadata into the Resources created by the Operator.

## Operator API Server

The Operator's API server can be configured to allow access to select URL routes
without requiring TLS authentication from the client and without
the HTTP Basic authentication used for role-based-access.

This configuration is performed by defining the `NOAUTH_ROUTES` environment
variable for the apiserver container within the Operator pod.

Typically, this configuration is made within the `deploy/deployment.json`
file for bash-based installations and
`ansible/roles/pgo-operator/templates/deployment.json.j2` for ansible installations.

For example:
```
...
    containers: [
        {
        	"name": "apiserver"
        	"env": [
                {
                	"name": "NOAUTH_ROUTES",
                	"value": "/health"
                }
        	]
        	...
        }
        ...
    ]
...
```

The `NOAUTH_ROUTES` variable must be set to a comma-separated list of
URL routes. For example: `/health,/version,/example3` would opt to **disable**
authentication for `$APISERVER_URL/health`, `$APISERVER_URL/version`, and
`$APISERVER_URL/example3` respectively.

Currently, only the following routes may have authentication disabled using
this setting:

```
/health
```

The `/healthz` route is used by kubernetes probes and has its authentication
disabed without requiring NOAUTH_ROUTES.


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
