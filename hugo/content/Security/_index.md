---
title: "Security"
date:
draft: false
weight: 7
---

## Kubernetes RBAC

Install the requisite Operator RBAC resources, *as a Kubernetes cluster admin user*,  by running a Makefile target:

    make installrbac


This script creates the following RBAC cluster-wide resources on your Kubernetes cluster:

| Setting |Definition  |
|---|---|
| Custom Resource Definitions (crd.yaml) | pgbackups|
|  | pgclusters|
|  | pgpolicies|
|  | pgreplicas|
|  | pgtasks|
|  | pgupgrades|
| Cluster Roles (cluster-roles.yaml) (cluster-roles-readonly.yaml) | pgo-cluster-role|
| Cluster Role Bindings (cluster-roles-bindings.yaml) | pgo-cluster-role|

The above cluster role/binding is necessary to list and watch 
namespaces at a minimum (cluster-roles-readonly.yaml).  This role lets
the Operator watch namespaces and run the following pgo CLI command:
    pgo show namespace --all

The default cluster role (cluster-roles.yaml) includes the permission to create and delete namespaces using the following pgo CLI commands:
    pgo create namespace mynamespace
    pgo update namespace mynamespace
    pgo delete namespace mynamespace

If you do not allow create/update of namespaces, you can manually
create Operator target namespaces using the following script:
    deploy/add-targeted-namespace.sh

WARNING: Unless the `cluster-admin` role has been assigned to the PGO service account during
installation (specifically using the `pgo_cluster_admin` variable, which is available for Ansible
installs only), when running version 4.1.0 of the Operator on OCP 3.11, you are REQUIRED to use the
`deploy/add-targeted-namespace.sh` script to add new targeted namespaces.  This is a bug that will
be fixed in a later version of the Operator.

This script creates the following RBAC namespace resources in the Operator
namespace (e.g. pgo namespace):

| Setting |Definition  |
|---|---|
| Roles (roles.yaml) | pgo-role|
| Role Bindings (role-bindings.yaml) | pgo-role|
| Service Account (service-accounts.yaml) | postgres-operator|

Targeted namespaces (e.g. pgouser1, pgouser2), used by the Operator to run Postgres clusters, include the following RBAC resources:

| Setting |Definition  |
|---|---|
| Roles (roles.yaml) | pgo-backrest-role|
|  | pgo-target-role|
| Role Bindings (role-bindings.yaml) | pgo-backrest-role-binding|
|  | pgo-target-role-binding|
| Service Account (service-accounts.yaml) | pgo-backrest|

These target namespace RBAC resources are created either dynamically
using the *pgo create namespace* command or via the manual namespace
setup script.

## Operator RBAC

Operator user roles (pgouser) are defined as Secrets starting in version 4.1 of the Operator.  Likewise, Operator users (pgouser) are also defined as Secrets.

The bootstrap Operator credential is created when you run:

    make installrbac

This creates an Operator role and user with the following
names:
    
    pgoadmin

The roles and user Secrets are created in the PGO_OPERATOR_NAMESPACE.

The default roles, role name, user name, and password are defined in the following script and should be modified for a production environment:

    deploy/install-bootstrap-creds.sh
 
These Secrets (pgouser/pgorole) control access to the Operator API.

## Target Namespaces

Starting in Operator version 4.1, targeted namespaces are created
by the Operator itself instead as configuration or user actions
dictate.

Targeted namespaces are namespaces which are configured for the
Operator to deploy Postgres clusters into and manage.

Targeted namespaces are those which have the following labels:

    vendor=crunchydata
    pgo-installation-name=devtest

In the above example, *devtest* is the default name for an Operator
installation.  This value is specified using the PGO_INSTALLATION_NAME
environment variable, when installing the Operator, you will
need to set this environment variable to be unique across your Kube
cluster.  This setting determines what namespaces an Operator
installation can access. 

Users can create a new Namespace as follows:
    pgo create namespace mynamespace

If you want to create the Namespace prior to the Operator using
them, you can do the following:
    kubectl create namespace mynamespace
    pgo update namespace mynamespace

The *update namespace* command will apply the required Operator RBAC
rules to that namespace.

You can manually create a targeted namespace by running
the *deploy/add-targeted-namespace.sh* script.  That script
will create the namespace, add the required labels, and apply
the necessary RBAC resources into that namespaces.  This can
be used for installations that do not want to install the
cluster role/bindings which enable dynamic namespace creation.

When you configure the Operator, you can still specify the NAMESPACE
environment variable with a list of namespaces, if they exist and
have the correct labels, the Operator will recognize and watch 
those namespaces.  If the NAMESPACE environment variable has names
that are not on your Kube system, the Operator will create the namespaces
at boot up time.

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
|CreateNamespace | allow *pgo create namespace*|
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
|DeleteNamespace | allow *pgo delete namespace*|
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
|ShowUser | allow *pgo show user*|
|ShowWorkflow | allow *pgo show workflow*|
|Status | allow *pgo status*|
|TestCluster | allow *pgo test*|
|UpdateCluster | allow *pgo update cluster*|
|UpdateNamespace | allow *pgo update namespace*|
|UpdatePgouser | allow *pgo update pgouser*|
|UpdatePgorole | allow *pgo update pgorole*|
|UpdateUser | allow *pgo update user*|
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
