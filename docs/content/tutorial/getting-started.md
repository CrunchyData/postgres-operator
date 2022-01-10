---
title: "Getting Started"
date:
draft: false
weight: 10
---

If you have not done so, please install PGO by following the [quickstart]({{< relref "quickstart/_index.md" >}}#installation).

As part of the installation, please be sure that you have done the following:

1. [Forked the Postgres Operator examples repository](https://github.com/CrunchyData/postgres-operator-examples/fork) and cloned it to your host machine.
1. Installed PGO to the `postgres-operator` namespace. If you are inside your `postgres-operator-examples` directory, you can run the `kubectl apply -k kustomize/install` command.

Note if you are using this guide in conjunction with images from the [Crunchy Data Customer Portal](https://access.crunchydata.com), please follow the [private registries]({{< relref "guides/private-registries.md" >}}) guide for additional setup instructions.

Throughout this tutorial, we will be building on the example provided in the `kustomize/postgres`.

When referring to a nested object within a YAML manifest, we will be using the `.` format similar to `kubectl explain`. For example, if we want to refer to the deepest element in this yaml file:

```
spec:
  hippos:
    appetite: huge
```

we would say `spec.hippos.appetite`.

`kubectl explain` is your friend. You can use `kubectl explain postgrescluster` to introspect the `postgrescluster.postgres-operator.crunchydata.com` custom resource definition. You can also review the [CRD reference]({{< relref "references/crd.md" >}}).

With PGO, the Postgres Operator installed, let's go and [create a Postgres cluster]({{< relref "./create-cluster.md" >}})!
