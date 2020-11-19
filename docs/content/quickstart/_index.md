---
title: "Quickstart"
date:
draft: false
weight: 10
---

# PostgreSQL Operator Quickstart

Can't wait to try out the PostgreSQL Operator? Let us show you the quickest possible path to getting up and running.

There are two paths to quickly get you up and running with the PostgreSQL Operator:

- [Installation via the PostgreSQL Operator Installer](#postgresql-operator-installer)
- Installation via a Marketplace
  - Installation via [Google Cloud Platform Marketplace](#google-cloud-platform-marketplace)

Marketplaces can help you get more quickly started in your environment as they provide a mostly automated process, but there are a few steps you will need to take to ensure you can fully utilize your PostgreSQL Operator environment.

# PostgreSQL Operator Installer

Below will guide you through the steps for installing and using the PostgreSQL Operator using an installer that works with Ansible.

## The Very, VERY Quickstart

If your environment is set up to use hostpath storage (found in things like [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [OpenShift Code Ready Containers](https://developers.redhat.com/products/codeready-containers/overview), the following command could work for you:

```
kubectl create namespace pgo
kubectl apply -f https://raw.githubusercontent.com/CrunchyData/postgres-operator/v4.3.4/installers/kubectl/postgres-operator.yml
```

If not, please read onward: you can still get up and running fairly quickly with just a little bit of configuration.

## Step 1: Configuration

### Get the PostgreSQL Operator Installer Manifest

You will need to download the PostgreSQL Operator Installer manifest to your environment, which you can do with the following command:

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v4.3.4/installers/kubectl/postgres-operator.yml > postgres-operator.yml
```

If you wish to download a specific version of the installer, you can substitute `master` with the version of the tag, i.e.

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v4.3.4/installers/kubectl/postgres-operator.yml > postgres-operator.yml
```

### Configure the PostgreSQL Operator Installer

There are many [configuration parameters]({{< relref "/installation/configuration.md">}}) to help you fine tune your installation, but there are a few that you may want to change to get the PostgreSQL Operator to run in your environment. Open up the `postgres-operator.yml` file and edit a few variables.

Find the `PGO_ADMIN_PASSWORD` variable. This is the password you will use with the [`pgo` client]({{< relref "/installation/pgo-client" >}}) to manage your PostgreSQL clusters. The default is `password`, but you can change it to something like `hippo-elephant`.

You will need also need to set the storage default storage classes that you would like the PostgreSQL Operator to use. These variables are called `PRIMARY_STORAGE`, `REPLICA_STORAGE`, `BACKUP_STORAGE`, and `BACKREST_STORAGE`. There are several storage configurations listed out in the configuration file under the heading `STORAGE[1-9]_TYPE`. Find the one that you want to use, and set it to that value.

For example, if your Kubernetes environment is using NFS storage, you would set these variables to the following:

```
- name: BACKREST_STORAGE
  value: "nfsstorage"
- name: BACKUP_STORAGE
  value: "nfsstorage"
- name: PRIMARY_STORAGE
  value: "nfsstorage"
- name: REPLICA_STORAGE
  value: "nfsstorage"
```

For a full list of available storage types that can be used with this installation method, please review the [configuration parameters]({{< relref "/installation/configuration.md">}}).

## Step 2: Installation

Installation is as easy as executing:

```
kubectl create namespace pgo
kubectl apply -f postgres-operator.yml
```

This will launch the `pgo-deployer` container that will run the various setup and installation jobs. This can take a few minutes to complete depending on your Kubernetes cluster.

While the installation is occurring, download the `pgo` client set up script. This will help set up your local environment for using the PostgreSQL Operator:

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v4.3.4/installers/kubectl/client-setup.sh > client-setup.sh
chmod +x client-setup.sh
```

When the PostgreSQL Operator is done installing, run the client setup script:

```
./client-setup.sh
```

This will download the `pgo` client and provide instructions for how to easily use it in your environment. It will prompt you to add some environmental variables for you to set up in your session, which you can do with the following commands:


```
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/pgo/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
export PGO_NAMESPACE=pgo
```

If you wish to permanently add these variables to your environment, you can run the following:

```
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/pgo/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
export PGO_NAMESPACE=pgo
EOF

source ~/.bashrc
```

**NOTE**: For macOS users, you must use `~/.bash_profile` instead of `~/.bashrc`

## Step 3: Verification

Below are a few steps to check if the PostgreSQL Operator is up and running.

By default, the PostgreSQL Operator installs into a namespace called `pgo`. First, see that the the Kubernetes Deployment of the Operator exists and is healthy:

```shell
kubectl -n pgo get deployments
```

If successful, you should see output similar to this:

```
NAME                READY   UP-TO-DATE   AVAILABLE   AGE
postgres-operator   1/1     1            1           16h
```

Next, see if the Pods that run the PostgreSQL Operator are up and running:

```shell
kubectl -n pgo get pods
```

If successful, you should see output similar to this:

```
NAME                                READY   STATUS    RESTARTS   AGE
postgres-operator-56d6ccb97-tmz7m   4/4     Running   0          2m
```

Finally, let's see if we can connect to the PostgreSQL Operator from the `pgo` command-line client. The Ansible installer installs the `pgo` command line client into your environment, along with the username/password file that allows you to access the PostgreSQL Operator. In order to communicate with the PostgreSQL Operator API server, you will first need to set up a [port forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) to your local environment.

In a new console window, run the following command to set up a port forward:

```shell
kubectl -n pgo port-forward svc/postgres-operator 8443:8443
```

Back to your original console window, you can verify that you can connect to the PostgreSQL Operator using the following command:

```shell
pgo version
```

If successful, you should see output similar to this:

```
pgo client version 4.3.4
pgo-apiserver version 4.3.4
```

## Step 4: Have Some Fun - Create a PostgreSQL Cluster

The quickstart installation method creates a namespace called `pgo` where the PostgreSQL Operator manages PostgreSQL clusters. Try creating a PostgreSQL cluster called `hippo`:

```shell
pgo create cluster -n pgo hippo
```

Alternatively, because we set the [`PGO_NAMESPACE`](/pgo-client/#general-notes-on-using-the-pgo-client) environmental variable in our `.bashrc` file, we could omit the `-n` flag from the [`pgo create cluster`](/pgo-client/reference/pgo_create_cluster/) command and just run this:

```shell
pgo create cluster hippo
```

Even with `PGO_NAMESPACE` set, you can always overwrite which namespace to use by setting the `-n` flag for the specific command. For explicitness, we will continue to use the `-n` flag in the remaining examples of this quickstart.

If your cluster creation command executed successfully, you should see output similar to this:

```
created Pgcluster hippo
workflow id 1cd0d225-7cd4-4044-b269-aa7bedae219b
```

This will create a PostgreSQL cluster named `hippo`. It may take a few moments for the cluster to be provisioned. You can see the status of this cluster using the `pgo test` command:

```shell
pgo test -n pgo hippo
```

When everything is up and running, you should see output similar to this:

```
cluster : hippo
	Services
		primary (10.97.140.113:5432): UP
	Instances
		primary (hippo-7b64747476-6dr4h): UP
```

The `pgo test` command provides you the basic information you need to connect to your PostgreSQL cluster from within your Kubernetes environment. For more detailed information, you can use `pgo show cluster -n pgo hippo`.

# Marketplaces

Below is the list of the marketplaces where you can find the Crunchy PostgreSQL Operator:

- Google Cloud Platform Marketplace: [Crunchy PostgreSQL for GKE](https://console.cloud.google.com/marketplace/details/crunchydata/crunchy-postgresql-operator)

Follow the instructions below for the marketplace that you want to use to deploy the Crunchy PostgreSQL Operator.

## Google Cloud Platform Marketplace

The PostgreSQL Operator is installed as part of the [Crunchy PostgreSQL for GKE](https://console.cloud.google.com/marketplace/details/crunchydata/crunchy-postgresql-operator) project that is available in the Google Cloud Platform Marketplace (GCP Marketplace). Please follow the steps deploy to get the PostgreSQL Operator deployed!

### Step 1: Prerequisites

#### Install `Kubectl` and `gcloud` SDK

- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) is required to execute kube commands with in GKE.
- [gcloudsdk](https://cloud.google.com/sdk/install) essential command line tools for google cloud

#### Verification

Below are a few steps to check if the PostgreSQL Operator is up and running.

For this example we are deploying the operator into a namespace called `pgo`. First, see that the the Kubernetes Deployment of the Operator exists and is healthy:

```shell
kubectl -n pgo get deployments
```

If successful, you should see output similar to this:

```
NAME                READY   UP-TO-DATE   AVAILABLE   AGE
postgres-operator   1/1     1            1           16h
```

Next, see if the Pods that run the PostgreSQL Operator are up and running:

```shell
kubectl -n pgo get pods
```

If successful, you should see output similar to this:

```
NAME                                READY   STATUS    RESTARTS   AGE
postgres-operator-56d6ccb97-tmz7m   4/4     Running   0          2m
```

### Step 2: Install the PostgreSQL Operator User Keys

After your operator is deployed via GCP Marketplace you will need to get keys used to secure the Operator REST API. For these instructions we will assume the operator is deployed in a namespace named "pgo" if this in not the case for your operator change the namespace to coencide with where your operator is deployed. Using the `gcloud` utility, ensure you are logged into the GKE cluster that you installed the PostgreSQL Operator into, run the following commands to retrieve the cert and key:

```shell
kubectl get secret pgo.tls -n pgo -o jsonpath='{.data.tls\.key}' | base64 --decode > /tmp/client.key
kubectl get secret pgo.tls -n pgo -o jsonpath='{.data.tls\.crt}' | base64 --decode > /tmp/client.crt
```

### Step 3: Setup PostgreSQL Operator User

The PostgreSQL Operator implements its own role-based access control (RBAC) system for authenticating and authorization PostgreSQL Operator users access to its REST API.  A default PostgreSQL Operator user (aka a "pgouser") is created as part of the marketplace installation (these credentials are set during the marketplace deployment workflow).

Create the pgouser file in `${HOME?}/.pgo/<operatornamespace>/pgouser` and insert the user and password you created on deployment of the PostgreSQL Operator via GCP Marketplace.  For example, if you set up a user with the username of `username` and a password of `hippo`:

```shell
username:hippo
```

### Step 4: Setup Environment variables

The PostgreSQL Operator Client uses several environmental variables to make it easier for interfacing with the PostgreSQL Operator.

Set the environmental variables to use the key / certificate pair that you pulled in Step 2 was deployed via the marketplace. Using the previous examples, You can set up environment variables with the following command:

```shell
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="/tmp/client.crt"
export PGO_CLIENT_CERT="/tmp/client.crt"
export PGO_CLIENT_KEY="/tmp/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
export PGO_NAMESPACE=pgo
```

If you wish to permanently add these variables to your environment, you can run the following command:

```shell
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="/tmp/client.crt"
export PGO_CLIENT_CERT="/tmp/client.crt"
export PGO_CLIENT_KEY="/tmp/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
export PGO_NAMESPACE=pgo
EOF

source ~/.bashrc
```

**NOTE**: For macOS users, you must use `~/.bash_profile` instead of `~/.bashrc`

### Step 5:  Install the PostgreSQL Operator Client `pgo`

The [`pgo` client](/pgo-client/) provides a helpful command-line interface to perform key operations on a PostgreSQL Operator, such as creating a PostgreSQL cluster.

The `pgo` client can be downloaded from GitHub [Releases](https://github.com/crunchydata/postgres-operator/releases) (subscribers can download it from the [Crunchy Data Customer Portal](https://access.crunchydata.com)).

Note that the `pgo` client's version must match the version of the PostgreSQL Operator that you have deployed. For example, if you have deployed version 4.3.4 of the PostgreSQL Operator, you must use the `pgo` for 4.3.4.

Once you have download the `pgo` client, change the permissions on the file to be executable if need be as shown below:

```shell
chmod +x pgo
```

### Step 6:  Connect to the PostgreSQL Operator

Finally, let's see if we can connect to the PostgreSQL Operator from the `pgo` client. In order to communicate with the PostgreSQL Operator API server, you will first need to set up a [port forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) to your local environment.

In a new console window, run the following command to set up a port forward:

```shell
kubectl -n pgo port-forward svc/postgres-operator 8443:8443
```

Back to your original console window, you can verify that you can connect to the PostgreSQL Operator using the following command:

```shell
pgo version
```

If successful, you should see output similar to this:

```
pgo client version 4.3.4
pgo-apiserver version 4.3.4
```

### Step 7: Create a Namespace

We are almost there!  You can optionally add a namespace that can be managed by the PostgreSQL Operator to watch and to deploy a PostgreSQL cluster into.

```shell
pgo create namespace wateringhole
```

verify the operator has access to the newly added namespace

```shell
pgo show namespace --all
```

you should see out put similar to this:

```shell
pgo username: admin
namespace                useraccess          installaccess       
application-system       accessible          no access                   
default                  accessible          no access                  
kube-public              accessible          no access           
kube-system              accessible          no access           
pgo                      accessible          no access
wateringhole             accessible          accessible  
```

### Step 8: Have Some Fun - Create a PostgreSQL Cluster

You are now ready to create a new cluster in the `wateringhole` namespace, try the command below:

```shell
pgo create cluster -n wateringhole hippo
```

If successful, you should see output similar to this:

```
created Pgcluster hippo
workflow id 1cd0d225-7cd4-4044-b269-aa7bedae219b
```

This will create a PostgreSQL cluster named `hippo`. It may take a few moments for the cluster to be provisioned. You can see the status of this cluster using the `pgo test` command:

```shell
pgo test -n wateringhole hippo
```

When everything is up and running, you should see output similar to this:

```
cluster : hippo
	Services
		primary (10.97.140.113:5432): UP
	Instances
		primary (hippo-7b64747476-6dr4h): UP
```

The `pgo test` command provides you the basic information you need to connect to your PostgreSQL cluster from within your Kubernetes environment. For more detailed information, you can use `pgo show cluster -n wateringhole hippo`.
