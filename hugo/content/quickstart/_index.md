---
title: "Quickstart"
date:
draft: false
weight: 10
---

# PostgreSQL Operator Quickstart

Can't wait to try out the PostgreSQL Operator? Let us show you the quickest possible path to getting up and running.

There are two paths to quickly get you up and running with the PostgreSQL Operator:

- Installation via a Marketplace
- Installation via the Ansible Installer

The marketplaces that include the Crunchy PostgreSQL Operator include:

- [Crunchy PostgreSQL for GKE](https://console.cloud.google.com/marketplace/details/crunchydata/crunchy-postgresql-operator) - Google Cloud Marketplace

The marketplaces will get your environment set up quickly, and you can use "Step 5" below to validate your installation.

Below will guide you through the steps for installing and using the PostgreSQL Operator using the Ansible Installer.

## Step 1: Prerequisites

### Kubernetes / OpenShift

- A Kubernetes or OpenShift environment where you have enough privileges to install an application, i.e. you can add a [ClusterRole](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole). If you're a Cluster Admin, you're all set.
  - Your Kubernetes version should be 1.13+. **NOTE**: For v4.2.0, while we have updated the PostgreSQL Operator for compatibility with 1.16+, we have not fully tested it.
  - For OpenShift, the PostgreSQL Operator will work in 3.11+
- [PersistentVolume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/)s that are available

### Your Environment

- [`kubectl`](https://kubernetes.io/docs/tasks/tools/install-kubectl/) or [`oc`](https://www.okd.io/download.html). Ensure you can access your Kubernetes or OpenShift cluster (this is outside the scope of this document)
- [`ansible`](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html) 2.5+. Learn how to [download ansible](https://docs.ansible.com/ansible/latest/installation_guide/intro_installation.html)
- `git`
- If you are installing to Google Kubernetes Engine, you will need the [`gcloud`](https://cloud.google.com/sdk/install) utility

## Step 2: Configuration

### Get the PostgreSQL Operator Ansible Installation Playbook

You can download the playbook by cloning the [PostgreSQL Operator git repository](https://github.com/CrunchyData/postgres-operator) and running the following commands:

```shell
git clone https://github.com/CrunchyData/postgres-operator.git
git checkout v4.2.0 # you can substitute this for the version that you want to install
cd ansible
```

### Configure your Installation

Within the `ansible` folder, there exists a file called `inventory`. When you open up this file, you can see several options that are used to install the PostgreSQL Operator. Most of these contain some sensible defaults for getting up and running quickly, but some you will need to fill out yourself.

Lines that start with a `#` are commented out. To activate that configuration setting, you will have to delete the `#`.

Set up your `inventory` file based on one of the environments that you are deploying to:

#### Kubernetes

You will have to uncomment and set the `kubernetes_context` variable. This can be determined based on the output of the `kubectl config current-context` e.g.:

```shell
kubectl config current-context
kubernetes-admin@kubernetes
```

Note that the output will vary based on the Kubernetes cluster you are using.

Using the above example, set the value of `kubernetes_context` to the output of the `kubectl config current-context` command, e.g.

```python
kubernetes_context="kubernetes-admin@kubernetes"
```

Find the location of the `pgo_admin_password` configuration variable. Set this to a password of your choosing, e.g.

```python
pgo_admin_password="hippo-elephant"
```

Finally, you will need to set the storage default storage classes that you would like the Operator to use. For example, if your Kubernetes environment is using NFS storage, you would set this variables to the following:

```python
backrest_storage='nfsstorage'
backup_storage='nfsstorage'
primary_storage='nfsstorage'
replica_storage='nfsstorage'
```

For a full list of available storage types that can be used with this installation method, see: $URL

#### OpenShift

For an OpenShfit deployment, you will at a minimum have to to uncomment and set the `openshift_host` variable. This is the location of where your OpenShift environment is, and can be obtained from your administrator. For example:

```python
openshift_host="https://openshift.example.com:6443"
```

Based on how your OpenShift environment is configured, you may need to set the following variables:

- `openshift_user`
- `openshift_password`
- `openshift_token`

An optional `openshift_skip_tls_verify=true` variable is available if your OpenShift environment allows you to skip TLS verification.

Next, find the location of the `pgo_admin_password` configuration variable. Set this to a password of your choosing, e.g.

```python
pgo_admin_password="hippo-elephant"
```

Finally, you will need to set the storage default storage classes that you would like the Operator to use. For example, if your OpenShift environment is using Rook storage, you would set this variables to the following:

```python
backrest_storage='rook'
backup_storage='rook'
primary_storage='rook'
replica_storage='rook'
```

For a full list of available storage types that can be used with this installation method, see: $URL

#### Google Kubernetes Engine (GKE)

For deploying the PostgreSQL Operator to GKE, you will need to set up your cluster similar to the Kubernetes set up. First, you will need to get the value for the `kubernetes_context` variable. Using the `gcloud` utility, ensure you are logged into the GCP Project that you are installing the PostgreSQL Operator into:

```shell
gcloud config set project [PROJECT_ID]
```

You can read about how you can [get the value of `[PROJECT_ID]`](https://cloud.google.com/resource-manager/docs/creating-managing-projects?visit_id=637125463737632776-3096453244&rd=1#identifying_projects)

From here, you can get the value that needs to be set into the `kubernetes_context`.

You will have to uncomment and set the `kubernetes_context` variable. This can be determined based on the output of the `kubectl config current-context` e.g.:

```shell
kubectl config current-context
gke_some-name_some-zone-some_project
```

Note that the output will vary based on your GKE project.

Using the above example, set the value of `kubernetes_context` to the output of the `kubectl config current-context` command, e.g.

```python
kubernetes_context="gke_some-name_some-zone-some_project"
```

Next, find the location of the `pgo_admin_password` configuration variable. Set this to a password of your choosing, e.g.

```python
pgo_admin_password="hippo-elephant"
```

Finally, you will need to set the storage default storage classes that you would like the Operator to use. For deploying to GKE it is recommended to use the `gce` storag class:

```python
backrest_storage='gce'
backup_storage='gce'
primary_storage='gce'
replica_storage='gce'
```
## Step 3: Installation

Ensure you are still in the `ansible` directory and run the following command to install the PostgreSQL Operator:

```shell
ansible-playbook -i inventory --tags=install main.yml
```

This can take a few minutes to complete depending on your Kubernetes cluster.

While the PostgreSQL Operator is installing, for ease of using the `pgo` command line interface, you will need to set up some environmental variables. You can do so with the following command:

```shell
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/pgo/client.pem"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
```

If you wish to permanently add these variables to your environment, you can run the following:

```shell
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/pgo/client.pem"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
EOF

source ~/.bashrc
```

**NOTE**: For macOS users, you must use `~/.bash_profile` instead of `~/.bashrc`

## Step 4: Verification

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
pgo client version 4.2.0
pgo-apiserver version 4.2.0
```

## Step 5: Have Some Fun - Create a PostgreSQL Cluster

The quickstart installation method creates two namespaces that you can deploy your PostgreSQL clusters into called `pgouser1` and `pgouser2`. Let's create a new PostgreSQL cluster in `pgouser1`:

```shell
pgo create cluster -n pgouser1 hippo
```

If successful, you should see output similar to this:

```
created Pgcluster hippo
workflow id 1cd0d225-7cd4-4044-b269-aa7bedae219b
```

This will create a PostgreSQL cluster named `hippo`. It may take a few moments for the cluster to be provisioned. You can see the status of this cluster using the `pgo test` command:

```shell
pgo test -n pgouser1 hippo
```

When everything is up and running, you should seet output similar to this:

```
cluster : hippo
	Services
		primary (10.97.140.113:5432): UP
	Instances
		primary (hippo-7b64747476-6dr4h): UP
```

The `pgo test` command provides you the basic information you need to connect to your PostgreSQL cluster from within your Kubernetes environment. For more detailed information, you can use `pgo show cluster -n pgouser1 hippo`.
