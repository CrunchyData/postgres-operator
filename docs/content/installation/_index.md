---
title: "Installation"
date:
draft: false
weight: 40
---

There are several different ways to install and deploy the [PGO, the Postgres Operator](https://www.crunchydata.com/developers/download-postgres/containers/postgres-operator)
based upon your use case.

For the vast majority of use cases, we recommend using the [Postgres Operator Installer]({{< relref "/installation/postgres-operator.md" >}}),
which uses the `pgo-deployer` container to set up all of the objects required to
run the PostgreSQL Operator.

For advanced use cases, such as for development, one may want to set up a
[development environment]({{< relref "/contributing/developer-setup.md" >}})
that is created using a series of scripts controlled by the Makefile.

Before selecting your installation method, it's important that you first read
the [prerequisites]({{< relref "/installation/prerequisites.md" >}}) for your
deployment environment to ensure that your setup meets the needs for installing
the PostgreSQL Operator.
