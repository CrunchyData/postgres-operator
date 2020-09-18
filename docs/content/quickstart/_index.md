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
  - Installation via [Google Cloud Marketplace]({{< relref "/installation/other/google-cloud-marketplace.md" >}})

Marketplaces can help you get more quickly started in your environment as they provide a mostly automated process, but there are a few steps you will need to take to ensure you can fully utilize your PostgreSQL Operator environment.

# PostgreSQL Operator Installer

Below will guide you through the steps for installing and using the PostgreSQL Operator using an installer that works with Ansible.

## The Very, VERY Quickstart

If your environment is set up to use hostpath storage (found in things like [minikube](https://kubernetes.io/docs/tasks/tools/install-minikube/) or [OpenShift Code Ready Containers](https://developers.redhat.com/products/codeready-containers/overview), the following command could work for you:

```
kubectl create namespace pgo
kubectl apply -f https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml
```

If not, please read onward: you can still get up and running fairly quickly with just a little bit of configuration.

## Step 1: Configuration

### Get the PostgreSQL Operator Installer Manifest

You will need to download the PostgreSQL Operator Installer manifest to your environment, which you can do with the following command:

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml > postgres-operator.yml
```

If you wish to download a specific version of the installer, you can substitute `master` with the version of the tag, i.e.

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml > postgres-operator.yml
```

### Configure the PostgreSQL Operator Installer

There are many [configuration parameters]({{< relref "/installation/configuration.md">}}) to help you fine tune your installation, but there are a few that you may want to change to get the PostgreSQL Operator to run in your environment. Open up the `postgres-operator.yml` file and edit a few variables.

Find the `pgo_admin_password` variable. This is the password you will use with the [`pgo` client]({{< relref "/installation/pgo-client" >}}) to manage your PostgreSQL clusters. The default is `password`, but you can change it to something like `hippo-elephant`.

You will also need to set the storage default storage classes that you would like the PostgreSQL Operator to use. These variables are called `primary_storage`, `replica_storage`, `backup_storage`, and `backrest_storage`. There are several storage configurations listed out in the configuration file under the heading `storage[1-9]_name`. Find the one that you want to use, and set it to that value.

For example, if your Kubernetes environment is using NFS storage, you would set these variables to the following:

```
backrest_storage: "nfsstorage"
backup_storage: "nfsstorage"
primary_storage: "nfsstorage"
replica_storage: "nfsstorage"
```

If you are using either Openshift or CodeReady Containers, you will need to set `disable_fsgroup` to 'true' in order to deploy the PostgreSQL Operator in OpenShift environments that have the typical restricted Security Context Constraints.

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
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/client-setup.sh > client-setup.sh
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
pgo client version {{< param operatorVersion >}}
pgo-apiserver version {{< param operatorVersion >}}
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

