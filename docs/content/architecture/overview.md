---
title: "Overview"
date:
draft: false
weight: 100
---

The goal of PGO, the Postgres Operator from Crunchy Data is to provide a means to quickly get
your applications up and running on Postgres for both development and
production environments. To understand how PGO does this, we
want to give you a tour of its architecture, with explains both the architecture
of the PostgreSQL Operator itself as well as recommended deployment models for
PostgreSQL in production!

# PGO Architecture

The Crunchy PostgreSQL Operator extends Kubernetes to provide a higher-level
abstraction for rapid creation and management of PostgreSQL clusters.  The
Crunchy PostgreSQL Operator leverages a Kubernetes concept referred to as
"[Custom Resources](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)”
to create several
[custom resource definitions (CRDs)](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions)
that allow for the management of PostgreSQL clusters.

The main custom resource definition is [`postgresclusters.postgres-operator.crunchydata.com`]({{< relref "references/crd.md" >}}). This allows you to control all the information about a Postgres cluster, including:

- General information
- Resource allocation
- High availability
- Backup management
- Where and how it is deployed (affinity, tolerations, topology spread constraints)
- Disaster Recovery / standby clusters
- Monitoring

and more.

PGO itself runs as a Deployment and is composed of a single container.

- `operator` (image: postgres-operator) - This is the heart of the PostgreSQL
Operator. It contains a series of Kubernetes
[controllers](https://kubernetes.io/docs/concepts/architecture/controller/) that
place watch events on a series of native Kubernetes resources (Jobs, Pods) as
well as the Custom Resources that come with the PostgreSQL Operator (Pgcluster,
Pgtask)

The main purpose of PGO is to create and update information
around the structure of a Postgres Cluster, and to relay information about the
overall status and health of a PostgreSQL cluster. The goal is to also simplify
this process as much as possible for users. For example, let's say we want to
create a high-availability PostgreSQL cluster that has a single replica,
supports having backups in both a local storage area and Amazon S3 and has
built-in metrics and connection pooling, similar to:

![PostgreSQL HA Cluster](/images/postgresql-cluster-ha-s3.png)

This can be accomplished with a relatively simple manifest. Please refer to the [tutorial]({{< relref "tutorial/_index.md" >}}) for how to accomplish this, or see the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repo.

The Postgres Operator handles setting up all of the various StatefulSets, Deployments, Services and other Kubernetes objects.

You will also notice that **high-availability is enabled by default** if you deploy at least one Postgres replica. The
Crunchy PostgreSQL Operator uses a distributed-consensus method for PostgreSQL
cluster high-availability, and as such delegates the management of each
cluster's availability to the clusters themselves. This removes the PostgreSQL
Operator from being a single-point-of-failure, and has benefits such as faster
recovery times for each PostgreSQL cluster. For a detailed discussion on
high-availability, please see the [High-Availability]({{< relref "architecture/high-availability.md" >}})
section.

## Kubernetes StatefulSets: The PGO Deployment Model

PGO, the Postgres Operator from Crunchy Data, uses [Kubernetes StatefulSets](https://kubernetes.io/docs/concepts/workloads/controllers/statefulset/)
for running Postgres instances, and will use [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) for more ephemeral services.

PGO deploys Kubernetes Statefulsets in a way to allow for creating both different Postgres instance groups and be able to support advanced operations such as rolling updates that minimize or eliminate Postgres downtime. Additional components in our
PostgreSQL cluster, such as the pgBackRest repository or an optional PgBouncer,
are deployed with Kubernetes Deployments.

With the PGO architecture, we can also leverage Statefulsets to apply affinity and toleration rules across every Postgres instance or individual ones. For instance, we may want to force one or more of our PostgreSQL replicas to run on Nodes in a different region than
our primary PostgreSQL instances.

What's great about this is that PGO manages this for you so you don't have to worry! Being aware of
this model can help you understand how the Postgres Operator gives you maximum
flexibility for your PostgreSQL clusters while giving you the tools to
troubleshoot issues in production.

The last piece of this model is the use of [Kubernetes Services](https://kubernetes.io/docs/concepts/services-networking/service/)
for accessing your PostgreSQL clusters and their various components. The
PostgreSQL Operator puts services in front of each Deployment to ensure you have
a known, consistent means of accessing your PostgreSQL components.

Note that in some production environments, there can be delays in accessing
Services during transition events. The PostgreSQL Operator attempts to mitigate
delays during critical operations (e.g. failover, restore, etc.) by directly
accessing the Kubernetes Pods to perform given actions.

# Additional Architecture Information

There is certainly a lot to unpack in the overall architecture of PGO. Understanding the architecture will help you to plan
the deployment model that is best for your environment. For more information on
the architectures of various components of the PostgreSQL Operator, please read
onward!
