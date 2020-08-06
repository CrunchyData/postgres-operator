---
title: "PostgreSQL Operator Metrics"
date:
draft: false
weight: 60
---

There are several different ways to install and deploy the
[PostgreSQL Operator Metrics infrastructure](https://www.crunchydata.com/developers/download-postgres/containers/postgres-operator)
based upon your use case.

For the vast majority of use cases, we recommend using the
[PostgreSQL Operator Metrics Installer]({{< relref "/installation/metrics/postgres-operator-metrics.md" >}}),
which uses the `pgo-deployer` container to set up all of the objects required to
run the PostgreSQL Operator Metrics infrastructure.  
Additionally, [Ansible](<{{< relref "/installation/metrics/metrics-configuration.md">}}>) and
[Helm](<{{< relref "/installation/metrics/other/ansible">}}>) installers are available.

Before selecting your installation method, it's important that you first read
the [prerequisites]({{< relref "/installation/metrics/metrics-prerequisites.md" >}}) for your
deployment environment to ensure that your setup meets the needs for installing
the PostgreSQL Operator Metrics infrastructure.
