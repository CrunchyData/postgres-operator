---
title: "Google Cloud Marketplace"
date:
draft: false
weight: 200
---

PGO: the PostgreSQL Operator from Crunchy Data is installed as part of [Crunchy PostgreSQL for GKE][gcm-listing]
that is available in the Google Cloud Marketplace.

[gcm-listing]: https://console.cloud.google.com/marketplace/details/crunchydata/crunchy-postgresql-operator


## Step 1: Install

Install [Crunchy PostgreSQL for GKE][gcm-listing] to a Google Kubernetes Engine cluster using
Google Cloud Marketplace.

## Step 2: Verify Installation

Install `kubectl` using the `gcloud components` command of the [Google Cloud SDK][sdk-install] or
by following the [Kubernetes documentation][kubectl-install].

[kubectl-install]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[sdk-install]: https://cloud.google.com/sdk/docs/install

Using the `gcloud` utility, ensure you are logged into the GKE cluster in which you installed PGO, the
PostgreSQL Operator, and see that it is running in the namespace in which you installed it.
For example, in the `pgo` namespace:

```shell
kubectl -n pgo get deployments,pods
```

If successful, you should see output similar to this:

```
NAME                                READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/postgres-operator   1/1     1            1           16h

NAME                                    READY   STATUS    RESTARTS   AGE
pod/postgres-operator-56d6ccb97-tmz7m   4/4     Running   0          2m
```


## Step 3: Install the PGO User Keys

You will need to get TLS keys used to secure the Operator REST API. Again, in the `pgo` namespace:

```shell
kubectl -n pgo get secret pgo.tls -o 'go-template={{ index .data "tls.crt" | base64decode }}' > /tmp/client.crt
kubectl -n pgo get secret pgo.tls -o 'go-template={{ index .data "tls.key" | base64decode }}' > /tmp/client.key
```


## Step 4: Setup PGO User

PGO implements its own role-based access control (RBAC) system for authenticating and authorization PostgreSQL Operator users access to its REST API.  A default PostgreSQL Operator user (aka a "pgouser") is created as part of the marketplace installation (these credentials are set during the marketplace deployment workflow).

Create the pgouser file in `${HOME?}/.pgo/<operatornamespace>/pgouser` and insert the user and password you created on deployment of the PostgreSQL Operator via GCP Marketplace.  For example, if you set up a user with the username of `username` and a password of `hippo`:

```shell
username:hippo
```


## Step 5: Setup Environment variables

The `pgo` Client uses several environmental variables to make it easier for interfacing with the PGO, the Postgres Operator.

Set the environmental variables to use the key / certificate pair that you pulled in Step 3 was deployed via the marketplace. Using the previous examples, You can set up environment variables with the following command:

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


## Step 6: Install the `pgo` Client

The [`pgo` client](/pgo-client/) provides a helpful command-line interface to perform key operations on a PGO Deployment, such as creating a PostgreSQL cluster.

The `pgo` client can be downloaded from GitHub [Releases](https://github.com/crunchydata/postgres-operator/releases) (subscribers can download it from the [Crunchy Data Customer Portal](https://access.crunchydata.com)).

Note that the `pgo` client's version must match the deployed version of PGO. For example, if you have deployed version {{< param operatorVersion >}} of the PostgreSQL Operator, you must use the `pgo` for {{< param operatorVersion >}}.

Once you have download the `pgo` client, change the permissions on the file to be executable if need be as shown below:

```shell
chmod +x pgo
```

## Step 7: Connect to PGO

Finally, let's see if we can connect to the Postgres Operator from the `pgo` client. In order to communicate with the PGO API server, you will first need to set up a [port forward](https://kubernetes.io/docs/tasks/access-application-cluster/port-forward-access-application-cluster/) to your local environment.

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

## Step 8: Create a Namespace

We are almost there!  You can optionally add a namespace that can be managed by PGO to watch and to deploy a PostgreSQL cluster into.

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

## Step 9: Have Some Fun - Create a PostgreSQL Cluster

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
