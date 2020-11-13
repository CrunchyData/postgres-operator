---
title: "Configuration of PostgreSQL Operator RBAC"
date:
draft: false
weight: 7
---


## PostreSQL Operator RBAC

The *conf/postgres-operator/pgorole* file is read at start up time when the operator is deployed to the Kubernetes cluster.  This file defines the PostgreSQL Operator roles whereby PostgreSQL Operator API users can be authorized.

The *conf/postgres-operator/pgouser* file is read at start up time also and contains username, password, role, and namespace information as follows:

    username:password:pgoadmin:
    pgouser1:password:pgoadmin:pgouser1
    pgouser2:password:pgoadmin:pgouser2
    pgouser3:password:pgoadmin:pgouser1,pgouser2
    readonlyuser:password:pgoreader:

The format of the pgouser server file is:

    <username>:<password>:<role>:<namespace,namespace>

The namespace is a comma separated list of namespaces that user has access to.  If you do not specify a namespace, then all namespaces is assumed, meaning this user can access any namespace that the Operator is watching.

A user creates a *.pgouser* file in their $HOME directory to identify themselves to the Operator.  An entry in .pgouser will need to match entries in the *conf/postgres-operator/pgouser* file.  A sample *.pgouser* file contains the following:

    username:password

The format of the .pgouser client file is:

    <username>:<password>

The users pgouser file can also be located at:

*/etc/pgo/pgouser*

or it can be found at a path specified by the PGOUSER environment variable.

If the user tries to access a namespace that they are not configured for within the server side *pgouser* file then they will get an error message as follows:

    Error: user [pgouser1] is not allowed access to namespace [pgouser2]


If you wish to add all available permissions to a *pgorole*, you can specify it by using a single `*` in your configuration. Note that if you are editing your YAML file directly, you will need to ensure to write it as `"*"` to ensure it is recognized as a string.

The following list shows the current complete list of possible pgo permissions that you can specify within the *pgorole* file when creating roles:

|Permission|Description  |
|---|---|
|ApplyPolicy | allow *pgo apply*|
|Cat | allow *pgo cat*|
|CreateBackup | allow *pgo backup*|
|CreateCluster | allow *pgo create cluster*|
|CreateDump | allow *pgo create pgdump*|
|CreateFailover | allow *pgo failover*|
|CreatePgAdmin | allow *pgo create pgadmin*|
|CreatePgbouncer | allow *pgo create pgbouncer*|
|CreatePolicy | allow *pgo create policy*|
|CreateUpgrade | allow *pgo upgrade*|
|CreateUser | allow *pgo create user*|
|DeleteBackup | allow *pgo delete backup*|
|DeleteCluster | allow *pgo delete cluster*|
|DeletePgAdmin | allow *pgo delete pgadmin*|
|DeletePgbouncer | allow *pgo delete pgbouncer*|
|DeletePolicy | allow *pgo delete policy*|
|DeleteUpgrade | allow *pgo delete upgrade*|
|DeleteUser | allow *pgo delete user*|
|DfCluster | allow *pgo df*|
|Label | allow *pgo label*|
|Reload | allow *pgo reload*|
|Restore | allow *pgo restore*|
|RestoreDump | allow *pgo restore* for pgdumps|
|ShowBackup | allow *pgo show backup*|
|ShowCluster | allow *pgo show cluster*|
|ShowConfig | allow *pgo show config*|
|ShowPgAdmin | allow *pgo show pgadmin*|
|ShowPgBouncer | allow *pgo show pgbouncer*|
|ShowPolicy | allow *pgo show policy*|
|ShowPVC | allow *pgo show pvc*|
|ShowNamespace | allow *pgo show namespace*|
|ShowSystemAccounts | allows commands with the `--show-system-accounts` flag to return system account information (e.g. the `postgres` superuser)|
|ShowUpgrade | allow *pgo show upgrade*|
|ShowWorkflow | allow *pgo show workflow*|
|Status | allow *pgo status*|
|TestCluster | allow *pgo test*|
|UpdatePgBouncer | allow *pgo update pgbouncer*|
|UpdateCluster | allow *pgo update cluster*|
|User | allow *pgo user*|
|Version | allow *pgo version*|


If the user is unauthorized for a pgo command, the user will get back this response:

    Error:  Authentication Failed: 403

## Making Security Changes

Importantly, it is necesssary to redeploy the PostgreSQL Operator prior to giving effect to the user security changes in the pgouser and pgorole files:

    make deployoperator

Performing this command will recreate the *pgo-config* ConfigMap that stores these files and is mounted by the Operator during its initialization.
