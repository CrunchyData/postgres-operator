---
title: "Getting Started"
draft: false
weight: 100
---

## Installation

If you have not installed the PostgreSQL Operator yet, we recommend you take a look at our [quickstart]({{< relref "quickstart/_index.md" >}}) or the [installation]({{< relref "installation/_index.md" >}}) sections.

### Customizing an Installation

How to customize a PostgreSQL Operator installation is a lengthy topic. The details are covered in the [installation]({{< relref "installation/postgres-operator.md" >}}) section, as well as a list of all the [configuration variables]({{< relref "installation/configuration.md" >}}) available.

## Setup the `pgo` Client

This tutorial will be using the [`pgo` client]({{< relref "pgo-client/_index.md" >}}) to interact with the PostgreSQL Operator. Please follow the instructions in the [quickstart]({{< relref "quickstart/_index.md" >}}) or the [installation]({{< relref "installation/pgo-client.md" >}}) sections for how to configure the `pgo` client.

The PostgreSQL Operator and `pgo` client are designed to work in a [multi-namespace deployment environment]({{< relref "architecture/namespace.md" >}}) and many `pgo` commands require that the namespace flag (`-n`) are passed into it. You can use the `PGO_NAMESPACE` environmental variable to set which namespace a `pgo` command can use. For example:

```
export PGO_NAMESPACE=pgo
pgo show cluster --all
```

would show all of the PostgreSQL clusters deployed to the `pgo` namespace. This is equivalent to:

```
pgo show cluster -n pgo --all
```

(Note: `-n` takes precedence over `PGO_NAMESPACE`.)

For convenience, we will use the `pgo` namespace created as part of the [quickstart]({{< relref "quickstart/_index.md" >}}) in this tutorial. In the shell that you will be executing the `pgo` commands in, run the following command:

```
export PGO_NAMESPACE=pgo
```

## Next Steps

Before proceeding, please make sure that your `pgo` client setup can communicate with your PostgreSQL Operator. In a separate terminal window, set up a port forward to your PostgreSQL Operator:

```
kubectl port-forward -n pgo svc/postgres-operator 8443:8443
```

The [`pgo version`]({{< relref "pgo-client/reference/pgo_version.md" >}}) command is a great way to check connectivity with the PostgreSQL Operator, as it is a very simple, safe operation. Try it out:

```
pgo version
```

If it is working, you should see results similar to:

```
pgo client version {{< param operatorVersion >}}
pgo-apiserver version {{< param operatorVersion >}}
```

Note that the version of the `pgo` client **must** match that of the PostgreSQL Operator.

You can also use the `pgo version` command to check the version specifically for the `pgo` client. This command only runs locally, i.e. it does not make any requests to the PostgreSQL Operator. For example:

```
pgo version --client
```

which yields results similar to:

```
pgo client version {{< param operatorVersion >}}
```

Alright, we're now ready to start our journey with the PostgreSQL Operator!
