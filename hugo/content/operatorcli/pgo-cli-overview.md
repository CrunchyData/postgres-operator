---
title: "PGO CLI Overview"
date:
draft: false
weight: 1
---

## PGO Command Line Interface (PGO CLI)

One of the suppport methods of interacting with the PostgreSQL Operator is through the command line tool, *pgo* CLI.  

The PGO CLI is downloaded from the GitHub Releases page for the PostgreSQL Operator (https://github.com/crunchydata/postgres-operator/releases).

The *pgo* client is provided in Mac, Windows, and Linux binary formats, download the appropriate client to your local laptop or workstation to work with a remote Operator.

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

To get detailed help information and command flag descriptions on each *pgo* command, enter:

    pgo [command] -h
 
## PGO CLI Operations

The following table shows the *pgo* operations currently implemented:

| Operation   |      Syntax      |  Description |
|:----------|:-------------|:------|
| apply |pgo apply mypolicy  --selector=name=mycluster  | Apply a SQL policy on a Postgres cluster(s) that have a label matching service-name=mycluster|
| backup |pgo backup mycluster  |Perform a backup on a Postgres cluster(s) |
| create |pgo create cluster mycluster  |Create an Operator resource type (e.g. cluster, policy, schedule, user) |
| delete |pgo delete cluster mycluster  |Delete an Operator resource type (e.g. cluster, policy, user, schedule) |
| ls |pgo ls mycluster  |Perform a Linux *ls* command on the cluster. |
| cat |pgo cat mycluster  |Perform a Linux *ls* command on the cluster. |
| df |pgo df mycluster  |Display the disk status/capacity of a Postgres cluster. |
| failover |pgo failover mycluster  |Perform a manual failover of a Postgres cluster. |
| help |pgo help |Display general *pgo* help information. |
| label |pgo label mycluster --label=environment=prod  |Create a metadata label for a Postgres cluster(s). |
| load |pgo load --load-config=load.json --selector=name=mycluster  |Perform a data load into a Postgres cluster(s).|
| reload |pgo reload mycluster  |Perform a pg_ctl reload command on a Postgres cluster(s). |
| restore |pgo restore mycluster |Perform a pgbackrest or pgdump restore on a Postgres cluster. |
| scale |pgo scale mycluster  |Create a Postgres replica(s) for a given Postgres cluster. |
| scaledown |pgo scaledown mycluster --query  |Delete a replica from a Postgres cluster. |
| show |pgo show cluster mycluster  |Display Operator resource information (e.g. cluster, user, policy, schedule). |
| status |pgo status  |Display Operator status. |
| test |pgo test mycluster  |Perform a SQL test on a Postgres cluster(s). |
| update |pgo update cluster --label=autofail=false  |Update a Postgres cluster(s). |
| upgrade |pgo upgrade mycluster  |Perform a minor upgrade to a Postgres cluster(s). |
| user |pgo user --selector=name=mycluster --update-passwords  |Perform Postgres user maintenance on a Postgres cluster(s). |
| version |pgo version  |Display Operator version information. |
