---
title: "Overview"
date:
draft: false
weight: 10
---

## PGO Command Line Interface (PGO CLI)

One of the suppport methods of interacting with the PostgreSQL Operator is through the command line tool, `pgo` CLI.  

The PGO CLI is downloaded from the GitHub Releases page for the PostgreSQL Operator (https://github.com/crunchydata/postgres-operator/releases).

The `pgo` client is provided in Mac, Windows, and Linux binary formats, download the appropriate client to your local laptop or workstation to work with a remote Operator.

## PGO CLI Syntax

Use the following syntax to run  `pgo`  commands from your terminal window:

    pgo [command] ([TYPE] [NAME]) [flags]

Where *command* is a verb like:

 * show
 * create
 * delete

And *type* is a resource type like:

 * cluster
 * policy
 * user

And *name* is the name of the resource type like:

 * mycluster
 * somesqlpolicy
 * john

To get detailed help information and command flag descriptions on each `pgo` command, enter:

    pgo [command] -h

## PGO CLI Operations

The following table shows the `pgo` operations currently implemented:

| Operation   | Syntax                                                       | Description                                                                                     |
| :---------- | :-------------                                               | :------                                                                                         |
| apply       | `pgo apply mypolicy --selector=name=mycluster`               | Apply a SQL policy on a Postgres cluster(s) that have a label matching `service-name=mycluster` |
| backup      | `pgo backup mycluster`                                       | Perform a backup on a Postgres cluster(s)                                                       |
| cat         | `pgo cat mycluster filepath`                                 | Perform a Linux `cat` command on the cluster.                                                   |
| clone      | `pgo clone oldcluster newcluster`                             | Copies the primary database of an existing cluster to a new cluster                         |
| create      | `pgo create cluster mycluster`                               | Create an Operator resource type (e.g. cluster, policy, schedule, user, namespace, pgouser, pgorole)                         |
| delete      | `pgo delete cluster mycluster`                               | Delete an Operator resource type (e.g. cluster, policy, user, schedule, namespace, pgouser, pgorole)                         |
| ls          | `pgo ls mycluster filepath`                                  | Perform a Linux `ls` command on the cluster.                                                    |
| df          | `pgo df mycluster`                                           | Display the disk status/capacity of a Postgres cluster.                                         |
| failover    | `pgo failover mycluster`                                     | Perform a manual failover of a Postgres cluster.                                                |
| help        | `pgo help`                                                   | Display general `pgo` help information.                                                         |
| label       | `pgo label mycluster --label=environment=prod`               | Create a metadata label for a Postgres cluster(s).                                              |
| load        | `pgo load --load-config=load.json --selector=name=mycluster` | Perform a data load into a Postgres cluster(s).                                                 |
| reload      | `pgo reload mycluster`                                       | Perform a `pg_ctl` reload command on a Postgres cluster(s).                                     |
| restore     | `pgo restore mycluster`                                      | Perform a `pgbackrest`, `pgbasebackup` or `pgdump` restore on a Postgres cluster.                               |
| scale       | `pgo scale mycluster`                                        | Create a Postgres replica(s) for a given Postgres cluster.                                      |
| scaledown   | `pgo scaledown mycluster --query`                            | Delete a replica from a Postgres cluster.                                                       |
| show        | `pgo show cluster mycluster`                                 | Display Operator resource information (e.g. cluster, user, policy, schedule, namespace, pgouser, pgorole).                   |
| status      | `pgo status`                                                 | Display Operator status.                                                                        |
| test        | `pgo test mycluster`                                         | Perform a SQL test on a Postgres cluster(s).                                                    |
| update      | `pgo update cluster mycluster --disable-autofail`            | Update a Postgres cluster(s), pgouser, pgorole, user, or namespace.                             |
| upgrade     | `pgo upgrade mycluster`                                      | Perform a minor upgrade to a Postgres cluster(s).                                               |
| version     | `pgo version`                                                | Display Operator version information.                                                           |


## pgo Global Flags
`pgo` global command flags include:

| Flag                | Description                                                                                                                                     |
| :--                 | :--                                                                                                                                             |
| `-n`                | namespace targeted for the command                                                                                                              |
| `--apiserver-url`   | URL of the Operator REST API service, override with `CO_APISERVER_URL` environment variable                                                     |
| `--debug`           | Enable debug messages                                                                                                                           |
| `--pgo-ca-cert`     | The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver. Override with `PGO_CA_CERT` environment variable          |
| `--pgo-client-cert` | The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.  Override with `PGO_CLIENT_CERT` environment variable |
| `--pgo-client-key`  | The Client Key file path for authenticating to the PostgreSQL Operator apiserver.  Override with `PGO_CLIENT_KEY` environment variable          |

## pgo Global Environment Variables
`pgo` will pick up these settings if set in your environment:

| Name          | Description                                                  | NOTES                                                             |
| :--           | :--                                                          | :--                                                               |
| `PGOUSERNAME` | The username (role) used for auth on the operator apiserver. | Requires that `PGOUSERPASS` be set.                               |
| `PGOUSERPASS` | The password for used for auth on the operator apiserver.    | Requires that `PGOUSERNAME` be set.                               |
| `PGOUSER`     | The path to the pgouser file.                                | Will be ignored if either `PGOUSERNAME` or `PGOUSERPASS` are set. |
