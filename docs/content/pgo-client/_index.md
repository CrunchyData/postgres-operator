---
title: "Using the pgo Client"
date:
draft: false
weight: 50
---

The PostgreSQL Operator Client, aka `pgo`, is the most convenient way to
interact with the PostgreSQL Operator. `pgo` provides many convenience methods
for creating, managing, and deleting PostgreSQL clusters through a series of
simple commands. The `pgo` client interfaces with the API that is provided by
the PostgreSQL Operator and can leverage the RBAC and TLS systems that are
provided by the PostgreSQL Operator

![Architecture](/Operator-Architecture.png)

The `pgo` client is available for Linux, macOS, and Windows, as well as a
`pgo-client` container that can be deployed alongside the PostgreSQL Operator.

You can download `pgo` from the [releases page](https://github.com/crunchydata/postgres-operator/releases),
or have it installed in your preferred binary format or as a container in your
Kubernetes cluster using the [Ansible Installer](/installation/install-with-ansible/).

## General Notes on Using the `pgo` Client

Many of the `pgo` client commands require you to specify a namespace via the
`-n` or `--namespace` flag. While this is a very helpful tool when managing
PostgreSQL deployments across many Kubernetes namespaces, this can become
onerous for the intents of this guide.

If you install the PostgreSQL Operator using the [quickstart](/quickstart/)
guide, you will install the PostgreSQL Operator to a namespace called `pgo`. We
can choose to always use one of these namespaces by setting the `PGO_NAMESPACE`
environmental variable, which is detailed in the global [`pgo` Client](/pgo-client/)
reference,

For convenience, we will use the `pgo` namespace in the examples below.
For even more convenience, we recommend setting `pgo` to be the value of
the `PGO_NAMESPACE` variable. In the shell that you will be executing the `pgo`
commands in, run the following command:

```shell
export PGO_NAMESPACE=pgo
```

If you do not wish to set this environmental variable, or are in an environment
where you are unable to use environmental variables, you will have to use the
`--namespace` (or `-n`) flag for most commands, e.g.

`pgo version -n pgo`

## Syntax

The syntax for `pgo` is similar to what you would expect from using the
`kubectl` or `oc` binaries. This is by design: one of the goals of the
PostgreSQL Operator project is to allow for seamless management of PostgreSQL
clusters in Kubernetes-enabled environments, and by following the command
patterns that users are familiar with, the learning curve is that much easier!

To get an overview of everything that is available at the top-level of `pgo`,
execute:

```shell
pgo
```

The syntax for the commands that `pgo` executes typicall follow this format:

```
pgo [command] ([TYPE] [NAME]) [flags]
```

Where *command* is a verb like:

- `create`
- `show`
- `delete`

And *type* is a resource type like:

- `cluster`
- `backup`
- `user`

And *name* is the name of the resource type like:

- hacluster
- gisdba

There are several global flags that are available to every `pgo` command as well
as flags that are specific to particular commands. To get a list of all the
options and flags available to a command, you can use the `--help` flag. For
example, to see all of the options available to the `pgo create cluster`
command, you can run the following:

```shell
pgo create cluster --help
```

## Command Overview

The following table provides an overview of the commands that the `pgo` client
provides:

| Operation   | Syntax                                                       | Description                                                                                     |
| :---------- | :-------------                                               | :------                                                                                         |
| apply       | `pgo apply mypolicy --selector=name=mycluster`               | Apply a SQL policy on a Postgres cluster(s) that have a label matching `service-name=mycluster` |
| backup      | `pgo backup mycluster`                                       | Perform a backup on a Postgres cluster(s)                                                       |
| cat         | `pgo cat mycluster filepath`                                 | Perform a Linux `cat` command on the cluster.                                                   |
| create      | `pgo create cluster mycluster`                               | Create an Operator resource type (e.g. cluster, policy, user, namespace, pgouser, pgorole)                         |
| delete      | `pgo delete cluster mycluster`                               | Delete an Operator resource type (e.g. cluster, policy, user, namespace, pgouser, pgorole)                         |
| df          | `pgo df mycluster`                                           | Display the disk status/capacity of a Postgres cluster.                                         |
| failover    | `pgo failover mycluster`                                     | Perform a manual failover of a Postgres cluster.                                                |
| help        | `pgo help`                                                   | Display general `pgo` help information.                                                         |
| label       | `pgo label mycluster --label=environment=prod`               | Create a metadata label for a Postgres cluster(s).                                              |
| reload      | `pgo reload mycluster`                                       | Perform a `pg_ctl` reload command on a Postgres cluster(s).                                     |
| restore     | `pgo restore mycluster`                                      | Perform a `pgbackrest` or `pgdump` restore on a Postgres cluster.                               |
| scale       | `pgo scale mycluster`                                        | Create a Postgres replica(s) for a given Postgres cluster.                                      |
| scaledown   | `pgo scaledown mycluster --query`                            | Delete a replica from a Postgres cluster.                                                       |
| show        | `pgo show cluster mycluster`                                 | Display Operator resource information (e.g. cluster, user, policy, namespace, pgouser, pgorole).                   |
| status      | `pgo status`                                                 | Display Operator status.                                                                        |
| test        | `pgo test mycluster`                                         | Perform a SQL test on a Postgres cluster(s).                                                    |
| update      | `pgo update cluster mycluster --disable-autofail`            | Update a Postgres cluster(s), pgouser, pgorole, user, or namespace.                             |
| upgrade     | `pgo upgrade mycluster`                                      | Perform a minor upgrade to a Postgres cluster(s).                                               |
| version     | `pgo version`                                                | Display Operator version information.                                                           |


### Global Flags

There are several global flags available to the `pgo` client.

**NOTE**: Flags take precedence over environmental variables.

| Flag                 | Description |
| :--                  | :-- |
| `--apiserver-url`    | The URL for the PostgreSQL Operator apiserver that will process the request from the pgo client. Note that the URL should **not** end in a `/`. |
| `--debug`            | Enable additional output for debugging. |
| `--disable-tls`      | Disable TLS authentication to the Postgres Operator. |
| `--exclude-os-trust` | Exclude CA certs from OS default trust store. |
| `-h`, `--help`       | Print out help for a command command.  |
| `-n`, `--namespace`  | The namespace to execute the `pgo` command in. This is required for most `pgo` commands. |
| `--pgo-ca-cert`      | The CA certificate file path for authenticating to the PostgreSQL Operator apiserver. |
| `--pgo-client-cert`  | The client certificate file path for authenticating to the PostgreSQL Operator apiserver. |
| `--pgo-client-key`   | The client key file path for authenticating to the PostgreSQL Operator apiserver. |

### Global Environment Variables

There are several environmental variables that can be used with the `pgo`
client.

**NOTE** Flags take precedence over environmental variables.


| Name                | Description                                                  |
| :--                 | :--                                                          |
| `EXCLUDE_OS_TRUST`  | Exclude CA certs from OS default trust store.                |
| `GENERATE_BASH_COMPLETION` | If set, will allow `pgo` to leverage "bash completion" to help complete commands as they are typed. |
| `PGO_APISERVER_URL` | The URL for the PostgreSQL Operator apiserver that will process the request from the pgo client. Note that the URL should **not** end in a `/`. |
| `PGO_CA_CERT`       | The CA certificate file path for authenticating to the PostgreSQL Operator apiserver. |
| `PGO_CLIENT_CERT`   | The client certificate file path for authenticating to the PostgreSQL Operator apiserver. |
| `PGO_CLIENT_KEY`    | The client key file path for authenticating to the PostgreSQL Operator apiserver. |
| `PGO_NAMESPACE`     | The namespace to execute the `pgo` command in. This is required for most `pgo` commands. |
| `PGOUSER`           | The path to the pgouser file. Will be ignored if either `PGOUSERNAME` or `PGOUSERPASS` are set. |
| `PGOUSERNAME`       | The username (role) used for auth on the operator apiserver. Requires that `PGOUSERPASS` be set. |
| `PGOUSERPASS`       | The password for used for auth on the operator apiserver. Requires that `PGOUSERNAME` be set. |

## Additional Information

How can you use the `pgo` client to manage your day-to-day PostgreSQL
operations? The next section covers many of the common types of tasks that
one needs to perform when managing production PostgreSQL clusters. Beyond that
is the full reference for all the available commands and flags for the `pgo`
client.

- [Common `pgo` Client Tasks](/pgo-client/common-tasks/)
- [`pgo` Client Reference](/pgo-client/common-reference/)
