---
title: "Security"
date:
draft: false
weight: 7
---

## Kubernetes RBAC

Install the requisite Operator RBAC resources, *as a Kubernetes cluster admin user*,  by running a Makefile target:

    make installrbac


This script creates the following RBAC resources on your Kubernetes cluster:

| Setting |Definition  |
|---|---|
| Custom Resource Definitions (crd.yaml) | pgbackups|
|  | pgclusters|
|  | pgpolicies|
|  | pgreplicas|
|  | pgtasks|
|  | pgupgrades|
| Cluster Roles (cluster-roles.yaml) | pgopclusterrole|
|  | pgopclusterrolecrd|
| Cluster Role Bindings (cluster-roles-bindings.yaml) | pgopclusterbinding|
|  | pgopclusterbindingcrd|
| Service Account (service-accounts.yaml) | postgres-operator|
| | pgo-backrest|
| Roles (rbac.yaml) | pgo-role|
| | pgo-backrest-role|
|Role Bindings  (rbac.yaml) | pgo-backrest-role-binding|
| | pgo-role-binding|


Note that the cluster role bindings have a naming convention of
pgopclusterbinding-$PGO_OPERATOR_NAMESPACE and
pgopclusterbindingcrd-$PGO_OPERATOR_NAMESPACE.  The PGO_OPERATOR_NAMESPACE
environment variable is added to make each cluster role binding
name unique and to support more than a single Operator being deployed
on the same Kube cluster.




## Operator RBAC

Operator roles are defined as Secrets starting in version 4.1 of the 
Operator.  Likewise, Operator users are also defined as Secrets.

The bootstrap Operator credential is created when you run:

    make installrbac

This creates an Operator role and user with the following
names:
    
    pgoadmin

The roles and user Secrets are created in the PGO_OPERATOR_NAMESPACE.

The default roles, role name, user name, and password are defined in the following script and should be modified for a production environment:

    deploy/install-bootstrap-creds.sh
 
These Secrets (pgouser/pgorole) control access to the Operator API.

### Managing Operator Roles

After installation, users can create additional Operator roles
as follows:

    pgo create pgorole somerole --permissions="Cat,Ls"

The above command creates a role named *somerole* with permissions to
execute the *cat* and *ls* API commands.  Permissions are comma separated.

The full set of permissions that can be used in a role are as follows:

|Permission|Description  |
|---|---|
|ApplyPolicy | allow *pgo apply*|
|Cat | allow *pgo cat*|
|CreateBackup | allow *pgo backup*|
|CreateBenchmark | allow *pgo create benchmark*|
|CreateCluster | allow *pgo create cluster*|
|CreateDump | allow *pgo create pgdump*|
|CreateFailover | allow *pgo failover*|
|CreatePgbouncer | allow *pgo create pgbouncer*|
|CreatePgouser | allow *pgo create pgouser*|
|CreatePgorole | allow *pgo create pgorole*|
|CreatePgpool | allow *pgo create pgpool*|
|CreatePolicy | allow *pgo create policy*|
|CreateSchedule | allow *pgo create schedule*|
|CreateUpgrade | allow *pgo upgrade*|
|CreateUser | allow *pgo create user*|
|DeleteBackup | allow *pgo delete backup*|
|DeleteBenchmark | allow *pgo delete benchmark*|
|DeleteCluster | allow *pgo delete cluster*|
|DeletePgbouncer | allow *pgo delete pgbouncer*|
|DeletePgouser | allow *pgo delete pgouser*|
|DeletePgorole | allow *pgo delete pgorole*|
|DeletePgpool | allow *pgo delete pgpool*|
|DeletePolicy | allow *pgo delete policy*|
|DeleteSchedule | allow *pgo delete schedule*|
|DeleteUpgrade | allow *pgo delete upgrade*|
|DeleteUser | allow *pgo delete user*|
|DfCluster | allow *pgo df*|
|Label | allow *pgo label*|
|Load | allow *pgo load*|
|Ls | allow *pgo ls*|
|Reload | allow *pgo reload*|
|Restore | allow *pgo restore*|
|RestoreDump | allow *pgo restore* for pgdumps|
|ShowBackup | allow *pgo show backup*|
|ShowBenchmark | allow *pgo show benchmark*|
|ShowCluster | allow *pgo show cluster*|
|ShowConfig | allow *pgo show config*|
|ShowPolicy | allow *pgo show policy*|
|ShowPgouser | allow *pgo show pgouser*|
|ShowPgorole | allow *pgo show pgorole*|
|ShowPVC | allow *pgo show pvc*|
|ShowSchedule | allow *pgo show schedule*|
|ShowNamespace | allow *pgo show namespace*|
|ShowUpgrade | allow *pgo show upgrade*|
|ShowWorkflow | allow *pgo show workflow*|
|Status | allow *pgo status*|
|TestCluster | allow *pgo test*|
|UpdateCluster | allow *pgo update cluster*|
|UpdatePgouser | allow *pgo update pgouser*|
|UpdatePgorole | allow *pgo update pgorole*|
|User | allow *pgo user*|
|Version | allow *pgo version*|


Roles can be viewed with the following command:

    pgo show pgorole --all

Roles can be removed with the following command:

    pgo delete pgorole somerole

Roles can be updated with the following command:

    pgo update pgorole somerole --permissions="Cat,Ls,ShowPolicy"

### Managing Operator Users

After installation, users can create additional Operator users
as follows:

    pgo create pgouser someuser --pgouser-namespaces="pgouser1,pgouser2" --pgouser-password=somepassword --pgouser-roles="somerole,someotherrole"

Mutliple roles and namespaces can be associated with a user by
specifying a comma separated list of values.

The namespace is a comma separated list of namespaces that
user has access to.  If you specify the flag "--all-namespaces", then
all namespaces is assumed, meaning this user can access any
namespace that the Operator is watching.

A user creates a *.pgouser* file in their $HOME directory to identify
themselves to the Operator.  A sample *.pgouser* file contains the following:

    pgoadmin:examplepassword

The format of the .pgouser client file is:

    <username>:<password>

If the user tries to access a namespace that they are not
configured for they will get an error message as follows:

    Error: user [pgouser1] is not allowed access to namespace [pgouser2]

If the user is unauthorized for a pgo command, the user will
get back this response:

    Error:  Authentication Failed: 401 

## API Security

The Operator REST API is encrypted with keys stored in the *pgo.tls* Secret.  

The pgo.tls Secret can be generated prior to starting the Operator or
you can let the Operator generate the Secret for you if the Secret
does not exist.

Adjust the default keys to meet your security requirements using your own keys.  The *pgo.tls* Secret is created when you run:

    make deployoperator

The keys are generated when the RBAC script is executed by the cluster admin:

    make installrbac

In some scenarios like an OLM deployment, it is preferable for the Operator to generate
the Secret keys at runtime, if the pgo.tls Secret does not exit
when the Operator starts, a new TLS Secret will be generated.
In this scenario, you can extract the generated Secret TLS keys using:

    kubectl cp <pgo-namespace>/<pgo-pod>:/tmp/server.key /tmp/server.key -c apiserver
    kubectl cp <pgo-namespace>/<pgo-pod>:/tmp/server.crt /tmp/server.crt -c apiserver
    
example of the command below:
    
    kubectl cp pgo/postgres-operator-585584f57d-ntwr5:tmp/server.key /tmp/server.key -c apiserver
    kubectl cp pgo/postgres-operator-585584f57d-ntwr5:tmp/server.crt /tmp/server.crt -c apiserver

This server.key and server.crt can then be used to access the *pgo-apiserver*
from the pgo CLI by setting the following variables in your client environment:

    export PGO_CA_CERT=/tmp/server.crt
    export PGO_CLIENT_CERT=/tmp/server.crt
    export PGO_CLIENT_KEY=/tmp/server.key

You can view the TLS secret using:

    kubectl get secret pgo.tls -n pgo
or

    oc get secret pgo.tls -n pgo

If you create the Secret outside of the Operator, for example using
the default installation script, the key and cert that are generated by the default installation are found here:

    $PGOROOT/conf/postgres-operator/server.crt 
    $PGOROOT/conf/postgres-operator/server.key 

The key and cert are generated using the *deploy/gen-api-keys.sh* script.
That script gets executed when running:

    make installrbac

You can extract the server.key and server.crt from the Secret using the
following:

    oc get secret pgo.tls -n $PGO_OPERATOR_NAMESPACE -o jsonpath='{.data.tls\.key}' | base64 --decode > /tmp/server.key
    oc get secret pgo.tls -n $PGO_OPERATOR_NAMESPACE -o jsonpath='{.data.tls\.crt}' | base64 --decode > /tmp/server.crt

This server.key and server.crt can then be used to access the *pgo-apiserver*
REST API from the pgo CLI on your client host.
