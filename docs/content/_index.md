---
title: "PGO, the Postgres Operator from Crunchy Data"
date:
draft: false
---

# PGO, the Postgres Operator from Crunchy Data

 <img width="25%" src="logos/pgo.svg" alt="PGO: The Postgres Operator from Crunchy Data" />

Latest Release: {{< param operatorVersion >}}

# Production Postgres Made Easy

[PGO](https://github.com/CrunchyData/postgres-operator), the [Postgres Operator]((https://github.com/CrunchyData/postgres-operator)) from [Crunchy Data](https://www.crunchydata.com), gives you a **declarative Postgres** solution that automatically manages your [PostgreSQL](https://www.postgresql.org) clusters.

Designed for your GitOps workflows, it is [easy to get started]({{< relref "quickstart/_index.md" >}}) with Postgres on Kubernetes with PGO. Within a few moments, you can have a production grade Postgres cluster complete with high availability, disaster recovery, and monitoring, all over secure TLS communications.Even better, PGO lets you easily customize your Postgres cluster to tailor it to your workload!

With conveniences like cloning Postgres clusters to using rolling updates to roll out disruptive changes with minimal downtime, PGO is ready to support your Postgres data at every stage of your release pipeline. Built for resiliency and uptime, PGO will keep your desired Postgres in a desired state so you do not need to worry about it.

PGO is developed with many years of production experience in automating Postgres management on Kubernetes, providing a seamless cloud native Postgres solution to keep your data always available.

## Supported Platforms

PGO, the Postgres Operator from Crunchy Data, is tested on the following platforms:

- Kubernetes 1.20+
- OpenShift 4.6+
- Rancher
- Google Kubernetes Engine (GKE), including Anthos
- Amazon EKS
- Microsoft AKS
- VMware Tanzu

This list only includes the platforms that the Postgres Operator is specifically
tested on as part of the release process: PGO works on other Kubernetes
distributions as well, such as Rancher.

The PGO Postgres Operator project source code is available subject to the [Apache 2.0 license](https://raw.githubusercontent.com/CrunchyData/postgres-operator/master/LICENSE.md) with the PGO logo and branding assets covered by [our trademark guidelines](/logos/TRADEMARKS.md).
