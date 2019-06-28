---
title: "Install Operator Using Bash"
date:
draft: false
weight: 300
---

A full installation of the Operator includes the following steps:

 - create a project structure
 - configure your environment variables
 - configure Operator templates
 - create security resources
 - deploy the operator
 - install pgo CLI (end user command tool)

Operator end-users are only required to install the pgo CLI client on their host and can skip the server-side installation steps.  pgo CLI clients are provided for Linux, Mac, and Windows clients.

The Operator can be deployed by multiple methods including:

 * default installation 
 * Ansible playbook installation 
 * Openshift Console installation using OLM


## Default Installation - Create Project Structure

The Operator follows a golang project structure, you can create a structure as follows on your local Linux host:

    mkdir -p $HOME/odev/src/github.com/crunchydata $HOME/odev/bin $HOME/odev/pkg
    cd $HOME/odev/src/github.com/crunchydata
    git clone https://github.com/CrunchyData/postgres-operator.git
    cd postgres-operator
	git checkout 4.0.1


This creates a directory structure under your HOME directory name *odev* and clones the current Operator version to that structure.  

## Default Installation - Configure Environment

Environment variables control aspects of the Operator installation.  You can copy a sample set of Operator environment variables and aliases to your *.bashrc* file to work with.

    cat $HOME/odev/src/github.com/crunchydata/postgres-operator/examples/envs.sh >> $HOME/.bashrc
    source $HOME/.bashrc

For various scripts used by the Operator, the *expenv* utility is required, download this utility from the Github Releases page, and place it into your PATH (e.g. $HOME/odev/bin).
{{% notice tip %}}There is also a Makefile target that includes is *expenv* and several other dependencies that are only needed if you plan on building from source: 

    make setup

{{% /notice %}}

## Default Installation - Namespace Creation

The default installation will create 3 namespaces to use
for deploying the Operator into and for holding Postgres clusters
created by the Operator.

Creating Kube namespaces is typically something that only a 
priviledged Kube user can perform so log into your Kube cluster as a user 
that has the necessary priviledges.

On Openshift if you do not want to install the Operator as the system
administrator, you can grant cluster-admin priviledges to a user
as follows:

    oc adm policy add-cluster-role-to-user cluster-admin pgoinstaller

In the above command, you are granting cluster-admin priviledges
to a user named pgoinstaller.  

The *NAMESPACE* environment variable is a comma separated list
of namespaces that specify where the Operator will be provisioing
PG clusters into, specifically, the namespaces the Operator is watching
for Kube events.  This value is set as follows:

    export NAMESPACE=pgouser1,pgouser2

This means namespaces called *pgouser1* and *pgouser2* will be
created as part of the default installation.  

The *PGO_OPERATOR_NAMESPACE* environment variable is a comma separated list
of namespace values that the Operator itself will be deployed into.  For
the installation example, this value is set as follows:

    export PGO_OPERATOR_NAMESPACE=pgo

This means a *pgo* namespace will be created and the Operator will
be deployed into that namespace.

Create the Operator namespaces using the Makefile target:

    make setupnamespaces

The [Design](/design) section of this documentation talks further about
the use of namespaces within the Operator.

## Default Installation - Configure Operator Templates

Within the Operator *conf* directory are several configuration files and templates used by the Operator to determine the various resources that it deploys on your Kubernetes cluster, specifically the PostgreSQL clusters it deploys.

When you install the Operator you must make choices as to what kind of storage the Operator has to work with for example.  Storage varies with each installation.  As an installer, you would modify these configuration templates used by the Operator to customize its behavior.

**Note**:  when you want to make changes to these Operator templates and configuration files after your initial installation, you will need to re-deploy the Operator in order for it to pick up any future configuration changes.

Here are some common examples of configuration changes most installers would make:

### Storage
Inside `conf/postgres-operator/pgo.yaml` there are various storage configurations defined.  

    PrimaryStorage: gce
    XlogStorage: gce
    ArchiveStorage: gce
    BackupStorage: gce
    ReplicaStorage: gce
      gce:
        AccessMode:  ReadWriteOnce
        Size:  1G
        StorageType:  dynamic
        StorageClass:  standard
        Fsgroup:  26

Listed above are the *pgo.yaml* sections related to storage choices.  *PrimaryStorage* specifies the name of the storage configuration used for PostgreSQL primary database volumes to be provisioned.  In the example above, a NFS storage configuration is picked.  That same storage configuration is selected for the other volumes that the Operator will create.

This sort of configuration allows for a PostgreSQL primary and replica to use different storage if you want.  Other storage settings like *AccessMode*, *Size*, *StorageType*, *StorageClass*, and *Fsgroup* further define the storage configuration.  Currently, NFS, HostPath, and Storage Classes are supported in the configuration.

As part of the Operator installation, you will need to adjust these storage settings to suit your deployment requirements.  For users wanting to try
out the Operator on Google Kubernetes Engine you would make the 
following change to the storage configuration in pgo.yaml:



For NFS Storage, it is assumed that there are sufficient Persistent Volumes (PV) created for the Operator to use when it creates Persistent Volume Claims (PVC).  The creation of Persistent Volumes is something a Kubernetes cluster-admin user would typically provide before installing the Operator.  There is an example script which can be used to create NFS Persistent Volumes located here:

    ./pv/create-nfs-pv.sh

That script looks for the IP address of an NFS server using the 
environment variable PGO_NFS_IP you would set in your .bashrc environment.

A similar script is provided for HostPath persistent volume creation if
you wanted to use HostPath for testing:
```
./pv/create-pv.sh
```

Adjust the above PV creation scripts to suit your local requirements, the 
purpose of these scripts are solely to produce a test set of Volume to test the 
Operator.

Other settings in *pgo.yaml* are described in the [pgo.yaml Configuration](/configuration/pgo-yaml-configuration) section of the documentation.

## Operator Security

The Operator implements its own RBAC (Role Based Access Controls) for authenticating Operator users access to the Operator REST API.

There is a default set of Roles and Users defined respectively in the following files and can be copied into your home directory as such:

```
./conf/postgres-operator/pgouser
./conf/postgres-operator/pgorole

cp ./conf/postgres-operator/pgouser $HOME/.pgouser
cp ./conf/postgres-operator/pgorole $HOME/.pgorole
```

Or create a *.pgouser* file in your home directory with a credential known by the Operator (see your administrator for Operator credentials to use):
```
    username:password
or
    pgouser1:password
or
    pgouser2:password
or
    pgouser3:password
or
    readonlyuser:password
```

Each example above has different priviledges in the Operator.  You can create this file as follows:

    echo "pgouser3:password" > $HOME/.pgouser

Note, you can also store the pgouser file in alternate locations, see the
Security documentation for details.

Operator security is discussed in the Security section [Security](/security) of the documentation.

Adjust these settings to meet your local requirements.

## Default Installation - Create Kube RBAC Controls

The Operator installation requires Kubernetes administrators to create Resources required by the Operator.  These resources are only allowed to be created by a cluster-admin user.  To install on Google Cloud, you will need a user
account with cluster-admin priviledges.  If you own the GKE cluster you
are installing on, you can add cluster-admin role to your account as
follows:

    kubectl create clusterrolebinding cluster-admin-binding --clusterrole cluster-admin --user $(gcloud config get-value account)

Specifically, Custom Resource Definitions for the Operator, and Service Accounts used by the Operator are created which require cluster permissions.

Tor create the Kube RBAC used by the Operator, run the following as a cluster-admin Kube user:

    make installrbac

This set of Resources is created a single time unless a new Operator 
release requires these Resources to be recreated.  Note that when you 
run *make installrbac* the set of keys used by the Operator REST API and 
also the pgbackrest ssh keys are generated.  

Verify the Operator Custom Resource Definitions are created as follows:

    kubectl get crd

You should see the *pgclusters* CRD among the listed CRD resource types.

See the Security documentation for a description of the various RBAC
resources created and used by the Operator.

## Default Installation - Deploy the Operator
At this point, you as a normal Kubernetes user should be able to deploy the Operator.  To do this, run the following Makefile target:

    make deployoperator

This will cause any existing Operator to be removed first, then the configuration to be bundled into a ConfigMap, then the Operator Deployment to be created.

This will create a postgres-operator Deployment and a postgres-operator Service.Operator administrators needing to make changes to the Operator 
configuration would run this make target to pick up any changes to pgo.yaml,
pgo users/roles,  or the Operator templates.

## Default Installation - Completely Cleaning Up

You can completely remove all the namespaces you have previously 
created using the default installation by running the following:

    make cleannamespaces

This will permanently delete each namespace the Operator installation
created previously.


## pgo CLI Installation
Most users will work with the Operator using the *pgo* CLI tool.  That tool is downloaded from the GitHub Releases page for the Operator (https://github.com/crunchydata/postgres-operator/releases). Crunchy Enterprise Customer can download the pgo binaries from https://access.crunchydata.com/ on the downloads page. 

The *pgo* client is provided in Mac, Windows, and Linux binary formats, 
download the appropriate client to your local laptop or workstation to work 
with a remote Operator.

Prior to using *pgo*, users testing the Operator on a single host can specify the
*postgres-operator* URL as follows:

```
    $ kubectl get service postgres-operator -n pgo
    NAME                CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
    postgres-operator   10.104.47.110   <none>        8443/TCP   7m
    $ export PGO_APISERVER_URL=https://10.104.47.110:8443
    pgo version
```

That URL address needs to be reachable from your local *pgo* client host.  Your Kubernetes administrator will likely need to create a network route, ingress, or LoadBalancer service to expose the Operator REST API to applications outside of the Kubernetes cluster.  Your Kubernetes administrator might also allow you to run the Kubernetes port-forward command, contact your adminstrator for details.

Next, the *pgo* client needs to reference the keys used to secure the Operator REST API:

```
    export PGO_CA_CERT=$PGOROOT/conf/postgres-operator/server.crt
    export PGO_CLIENT_CERT=$PGOROOT/conf/postgres-operator/server.crt
    export PGO_CLIENT_KEY=$PGOROOT/conf/postgres-operator/server.key
```

You can also specify these keys on the command line as follows:

    pgo version --pgo-ca-cert=$PGOROOT/conf/postgres-operator/server.crt --pgo-client-cert=$PGOROOT/conf/postgres-operator/server.crt --pgo-client-key=$PGOROOT/conf/postgres-operator/server.key

{{% notice tip %}} if you are running the Operator on Google Cloud, you would open up another terminal and run *kubectl port-forward ...* to forward the Operator pod port 8443 to your localhost where you can access the Operator API from your local workstation.
{{% /notice %}}

At this point, you can test connectivity between your laptop or workstation and the Postgres Operator deployed on a Kubernetes cluster as follows:

    pgo version

You should get back a valid response showing the client and server version numbers.

## Verify the Installation

Now that you have deployed the Operator, you can verify that it is running correctly.

You should see a pod running that contains the Operator:

    kubectl get pod --selector=name=postgres-operator -n pgo
    NAME                                 READY     STATUS    RESTARTS   AGE
    postgres-operator-79bf94c658-zczf6   3/3       Running   0          47s


That pod should show 3 of 3 containers in *running* state and that the operator is installed into the *pgo* namespace.

The sample environment script, examples/env.sh, if used creates some bash functions that you can use to view the Operator logs.  This is useful in case you find one of the Operator containers not in a running status.

Using the pgo CLI, you can verify the versions of the client and server match as follows:

    pgo version

This also tests connectivity between your pgo client host and the Operator server.

