---
title: "Installation"
date:
draft: false
weight: 1
---

A full installation of the Operator includes the following steps:

 - create a project structure
 - configure your environment variables
 - configure Operator templates
 - create security resources
 - deploy the operator
 - install pgo CLI (end user command tool)

Operator end-users are only required to install the pgo CLI client on their host and can skip the server-side installation steps.  pgo CLI clients are provided
on the Github Releases page for Linux, Mac, and Windows clients.

The Operator can also be deployed with a sample Helm chart and also a *quickstart* script.  Those installation methods don't provide the same level of customization that the installation provides but are alternatives.  Crunchy also provides an Ansible playbook for Crunchy customers.  

See below for details on the Helm and quickstart installation methods.

## Create Project Structure
The Operator follows a golang project structure, you can create a structure as follows on your local Linux host:

    mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
    cd $HOME/odev/src/github.com/crunchydata
    git clone https://github.com/CrunchyData/postgres-operator.git
    cd postgres-operator
    git checkout 3.5.0


This creates a directory structure under your HOME directory name *odev* and clones the current Operator version to that structure.  

## Configure Environment
Environment variables control aspects of the Operator installation.  You can copy a sample set of Operator environment variables and aliases to your *.bashrc* file to work with.

    cat $HOME/odev/src/github.com/crunchydata/postgres-operator/examples/envs.sh >> $HOME/.bashrc
    source $HOME/.bashrc

In this example set of environment variables, the CO_NAMESPACE environment variable is set to *demo* as an example namespace in which the Operator will be deployed.  See the Design section of documentation on the Operator namespace requirements.  

Adjust the namespace value to suit your needs.   There is a Makefile target you can run to create the *demo* namespace if you want:

    make setupnamespace

Note, that command sets your Kubernetes context to be *demo* as well, so use with caution if you are using your system's main kubeconfig file.

## Configure Operator Templates

Within the Operator *conf* directory are several configuration files and templates used by the Operator to determine the various resources that it deploys on your Kubernetes cluster, specifically the PostgreSQL clusters it deploys.

When you install the Operator you must make choices as to what kind of storage the Operator has to work with for example.  Storage varies with each installation.  As an installer, you would modify these configuration templates used by the Operator to customize its behavior.

**Note**:  when you want to make changes to these Operator templates and configuration files after your initial installation, you will need to re-deploy the Operator in order for it to pick up any future configuration changes.

Here are some common examples of configuration changes most installers would make:

### Storage
Inside `conf/postgresql-operator/pgo.yaml` there are various storage configurations defined.  

    PrimaryStorage: nfsstorage
    ArchiveStorage: nfsstorage
    BackupStorage: nfsstorage
    ReplicaStorage: nfsstorage
    Storage:
      hostpathstorage:
        AccessMode:  ReadWriteMany
        Size:  1G
        StorageType:  create
      nfsstorage:
        AccessMode:  ReadWriteMany
        Size:  1G
        StorageType:  create
        SupplementalGroups:  65534
      storageos:
        AccessMode:  ReadWriteOnce
        Size:  1G
        StorageType:  dynamic
        StorageClass:  fast
        Fsgroup:  26

Listed above are the *pgo.yaml* sections related to storage choices.  *PrimaryStorage* specifies the name of the storage configuration used for PostgreSQL primary database volumes to be provisioned.  In the example above, a NFS storage configuration is picked.  That same storage configuration is selected for the other volumes that the Operator will create.

This sort of configuration allows for a PostgreSQL primary and replica to use different storage if you want.  Other storage settings like *AccessMode*, *Size*, *StorageType*, *StorageClass*, and *Fsgroup* further define the storage configuration.  Currently, NFS, HostPath, and Storage Classes are supported in the configuration.

As part of the Operator installation, you will need to adjust these storage settings to suit your deployment requirements.

For NFS Storage, it is assumed that there are sufficient Persistent Volumes (PV) created for the Operator to use when it creates Persistent Volume Claims (PVC).  The creation of PV's is something a Kubernetes cluster-admin user would typically provide before installing the Operator.  There is an example script which can be used to create NFS Persistent Volumes located here:

    ./pv/create-nfs-pv.sh

A similar script is provided for HostPath persistent volume creation if
you wanted to use HostPath for testing:
```
./pv/create-pv.sh
```

Adjust the above PV creation scripts to suit your local requirements, the purpose
of these scripts are solely to produce a test set of Volume to test the 
Operator.

Other settings in *pgo.yaml* are described in the [pgo.yaml Configuration](/configuration/pgo-yaml-configuration) section of the documentation.

## Operator Security
 The Operator implements its own RBAC (Role Based Access Controls) for authenticating Operator users access to the Operator's REST API.

There is a default set of Roles and Users defined respectively in the following files:

```
./conf/postgres-operator/pgouser
./conf/postgres-operator/pgorole
```

Adjust these settings to meet your local requirements.

## Create Security Resources
The Operator installation requires Kubernetes administrators to create Resources required by the Operator.  These resources are only allowed to be created by a cluster-admin user.

Specifically, Custom Resource Definitions for the Operator, and Service Accounts used by the Operator are created which require cluster permissions.

As part of the installation, download the *expenv* utility from the Releases page, add that to
your path and as cluster admin, run the following Operator Makefile target:

    make installrbac

That target will create the RBAC Resources required by the Operator.   This set of Resources is created a single time unless a new Operator release requires these Resources to be recreated.  Note that when you run *make installrbac* the set of keys used by the Operator REST API and also the pgbackrest ssh keys are generated.  These keys are stored in the ConfigMap used by the Operator for securing connections.  

Verify the Operator Custom Resource Definitions are created as follows:

    kubectl get crd

You should see the *pgclusters* CRD among the listed CRD resource types.

## Deploy the Operator
At this point, you as a normal Kubernetes user should be able to deploy the Operator.  To do this, run the following Makefile target:

    make deployoperator

This will cause any existing Operator to be removed first, then the configuration to be bundled into a ConfigMap, then the Operator Deployment to be created.

This will create a postgres-operator Deployment along with a crunchy-scheduler Deployment, and a postgres-operator Service.  So, Operator administrators needing
to make changes to the Operator configuration would run this make target
to pick up any changes to pgo.yaml or the Operator templates.


## pgo CLI Installation
Most users will work with the Operator using the *pgo* CLI tool.  That tool is downloaded from the GitHub Releases page for the Operator (https://github.com/crunchydata/postgres-operator/releases).

The *pgo* client is provided in Mac, Windows, and Linux binary formats, download the appropriate client to your local laptop or workstation to work with a remote Operator.
Prior to using *pgo*, users testing the Operator on a single host can specify the
*postgres-operator* URL as follows:

```
    $ kubectl get service postgres-operator
    NAME                CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
    postgres-operator   10.104.47.110   <none>        8443/TCP   7m
    $ export CO_APISERVER_URL=https://10.104.47.110:8443
    pgo version
```

That URL address needs to be reachable from your local *pgo* client host.  Your Kubernetes administrator will likely need to create a network route, ingress, or LoadBalancer service to expose the Operator's REST API to applications outside of the Kubernetes cluster.  Your Kubernetes administrator might also allow you to run the Kubernetes port-forward command, contact your adminstrator for details.

Next, the *pgo* client needs to reference the keys used to secure the Operator REST API:

```
    export PGO_CA_CERT=$COROOT/conf/postgres-operator/server.crt
    export PGO_CLIENT_CERT=$COROOT/conf/postgres-operator/server.crt
    export PGO_CLIENT_KEY=$COROOT/conf/postgres-operator/server.key
```

You can also specify these keys on the command line as follows:

    pgo version --pgo-ca-cert=$COROOT/conf/postgres-operator/server.crt --pgo-client-cert=$COROOT/conf/postgres-operator/server.crt --pgo-client-key=$COROOT/conf/postgres-operator/server.key

Lastly, create a *.pgouser* file in your home directory with a credential known by the Operator (see your administrator for Operator credentials to use):

    username:password

You can create this file as follows:

    echo "username:password" > $HOME/.pgouser

Note, you can also store the pgouser file in alternate locations, see the
Security documentation for details.

At this point, you can test connectivity between your laptop or workstation and the Postgres Operator deployed on a Kubernetes cluster as follows:

    pgo version

You should get back a valid response showing the client and server version numbers.

## Verify the Installation
Now that you have deployed the Operator, you can verify that it is running correctly.

You should see a pod running that contains the Operator:

    kubectl get pod --selector=name=postgres-operator

That pod should show 2 of 2 containers in *running* state.

The sample environment script, env.sh, if used creates some bash alias commands that you can use to view the Operator logs.  This is useful in case you find one of the Operator containers not in a running status.

Using the pgo CLI, you can verify the versions of the client and server match as follows:

    pgo version

This also tests connectivity between your pgo client host and the Operator server.


## Helm Chart
The Operator Helm chart is located in the following location:
./postgres-operator/chart

Modify the Helm templates to suit your requirements.  The Operator templates in the *conf* directory are essentially the same as found in the Helm chart folder.  Adjust as mentioned above to customize the installation.

Also, a pre-installation step is currently required prior to installing the Operator Helm chart.  Specifically, the following script must be executed prior to installing the chart:

    ./postgres-operator/chart/gen-pgo-keys.sh

This script will generate any keys and certificates required to deploy the Operator, and will then place them in the proper directory within the Helm chart.

## Quickstart Script
There is a *quickstart* script found in the following GitHub repository location which seeks to automate a simple Operator deployment onto an existing Kubernetes installation:

    ./examples/quickstart.sh

This script is a bash script and is intended to run on Linux hosts.  The script will ask you questions related to your configuration and the proceed to execute commands to cause the Operator to be deployed.  The quickstart script is meant for very simple deployments and to test the Operator and would not be typically used to maintain an Operator deployment.

Get a copy of the script as follows:

    wget https://raw.githubusercontent.com/CrunchyData/postgres-operator/master/examples/quickstart.sh
    chmod +x ./quickstart.sh

There are some prerequisites for running this script:

 * a running Kubernetes system
 * access to a Kube user account that has cluster-admin priviledges, this is required to install the Operator RBAC rules
 * a namespace created to hold the Operator
 * a Storage Class used for dynamic storage provisioning
 * a Mac, Ubuntu, or Centos host to install from, this host and your terminal session should be configured to access your Kube cluster
