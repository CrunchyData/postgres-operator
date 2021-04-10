---
title: Install PGO the Postgres Operator
date:
draft: false
weight: 20
---

# PGO: Postgres Operator Installer

## Quickstart

If you believe that all the default settings in the installation manifest work
for you, you can take a chance by running the manifest directly from the
repository:

```
kubectl create namespace pgo
kubectl apply -f https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml
```

However, we still advise that you read onward to see how to properly configure
the PostgreSQL Operator.

## Overview

PGO comes with a container called `pgo-deployer` which
handles a variety of lifecycle actions for the Postgres Operator, including:

- Installation
- Upgrading
- Uninstallation

After configuring the Job template, the installer can be run using
[`kubectl apply`](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#apply)
and takes care of setting up all of the objects required to run the PostgreSQL
Operator.

The installation manifest, called [`postgres-operator.yaml`](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml), is available in the [`installers/kubectl/postgres-operator.yml`](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml)
path in the PostgreSQL Operator repository.


## Requirements

### RBAC

The `pgo-deployer` requires a [ServiceAccount](https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/)
and [ClusterRoleBinding](https://kubernetes.io/docs/reference/access-authn-authz/rbac/#role-and-clusterrole)
to run the installation job. Both of these resources are already defined
in the `postgres-operator.yml`, but can be updated based on your specific
environmental requirements.

By default, the `pgo-deployer` uses a ServiceAccount called `pgo-deployer-sa`
that has a ClusterRoleBinding (`pgo-deployer-crb`) with several ClusterRole
permissions. This is required to create the [Custom Resource Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
that power PGO. While the Postgres Operator itself can be
scoped to a specific namespace, you will need to have `cluster-admin` for the
initial deployment, or privileges that allow you to install Custom Resource
Definitions. The required list of privileges are available in the [postgres-operator.yml](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml) file:

[https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml)

If you have already configured the ServiceAccount and ClusterRoleBinding for the
installation process (e.g. from a previous installation), then you can remove
these objects from the `postgres-operator.yml` manifest.

### Config Map

The `pgo-deployer` uses a [Kubernetes ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)
to pass configuration options into the installer. The ConfigMap is defined in
the `postgres-operator.yaml` file and can be updated based on your configuration
preferences.

### Namespaces

By default, the installer will run in the `pgo` Namespace. This can be
updated in the `postgres-operator.yml` file. **Please ensure that this namespace
exists before the job is run**.

For example, to create the `pgo` namespace:

```
kubectl create namespace pgo
```

The Postgres Operator has the ability to manage PostgreSQL clusters across
multiple Kubernetes [Namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/),
including the ability to add and remove Namespaces that it watches. Doing so
does require the PostgreSQL Operator to have elevated privileges, and as such,
the PostgreSQL Operator comes with three "namespace modes" to select what level
of privileges to provide:

- `dynamic`: The default is the default mode. This enables full dynamic Namespace
management capabilities, in which the PostgreSQL Operator can create, delete and
update any Namespaces within the Kubernetes cluster, while then also having the
ability to create the Roles, RoleBindings andService Accounts within those
Namespaces for normal operations. The PostgreSQL Operator can also listen for
Namespace events and create or remove controllers for various Namespaces as
changes are made to Namespaces from Kubernetes and the PostgreSQL Operator's
management.

- `readonly`: In this mode, the PostgreSQL Operator is able to listen for
namespace events within the Kubernetes cluster, and then manage controllers
as Namespaces are added, updated or deleted. While this still requires a
ClusterRole, the permissions mirror those of a "read-only" environment, and as
such the PostgreSQL Operator is unable to create, delete or update Namespaces
itself nor create RBAC that it requires in any of those Namespaces. Therefore,
while in readonly, mode namespaces must be preconfigured with the proper RBAC
as the PostgreSQL Operator cannot create the RBAC itself.

- `disabled`: Use this mode if you do not want to deploy the PostgreSQL Operator
with any ClusterRole privileges, especially if you are only deploying the
PostgreSQL Operator to a single namespace. This disables any Namespace
management capabilities within the PostgreSQL Operator and will simply attempt
to work with the target Namespaces specified during installation. If no target
Namespaces are specified, then the Operator will be configured to work within
the namespace in which it is deployed. As with the readonly mode, while in
this mode, Namespaces must be preconfigured with the proper RBAC, since the
PostgreSQL Operator cannot create the RBAC itself.

## Configuration - `postgres-operator.yml`

The `postgres-operator.yml` file contains all of the configuration parameters
for deploying PGO. The [example file](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/kubectl/postgres-operator.yml)
contains defaults that should work in most Kubernetes environments, but it may
require some customization.

For a detailed description of each configuration parameter, please read the
[PostgreSQL Operator Installer Configuration Reference](<{{< relref "/installation/configuration.md">}}>)

#### Configuring to Update and Uninstall

The deploy job can be used to perform different deployment actions for the
PostgreSQL Operator. When you run the job it will install the operator by
default but you can change the deployment action to uninstall or update. The
`DEPLOY_ACTION` environment variable in the `postgres-operator.yml` file can be
set to `install`, `update`, and `uninstall`.


### Image Pull Secrets

If you are pulling PGO images from a private registry, you
will need to setup an
[imagePullSecret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
with access to the registry. The image pull secret will need to be added to the
installer service account to have access. The secret will need to be created in
each namespace that the PostgreSQL Operator will be using.

After you have configured your image pull secret in the Namespace the installer
runs in (by default, this is `pgo`),
add the name of the secret to the job yaml that you are using. You can update
the existing section like this:

```
apiVersion: v1
kind: ServiceAccount
metadata:
    name: pgo-deployer-sa
    namespace: pgo
imagePullSecrets:
  - name: <image_pull_secret_name>
```

If the service account is configured without using the job yaml file, you
can link the secret to an existing service account with the `kubectl` or `oc`
clients.

```
# kubectl
kubectl patch serviceaccount <deployer-sa> -p '{"imagePullSecrets": [{"name": "myregistrykey"}]}' -n <install-namespace>

# oc
oc secrets link <registry-secret> <deployer-sa> --for=pull --namespace=<install-namespace>
```

## Installation

Once you have configured the PGO Installer to your
specification, you can install the PostgreSQL Operator with the following
command:

```shell
kubectl apply -f /path/to/postgres-operator.yml
```

### Install the [`pgo` Client]({{< relref "/installation/pgo-client" >}})

To use the [`pgo` Client]({{< relref "/installation/pgo-client" >}}),
there are a few additional steps to take in order to get it to work with you
PostgreSQL Operator installation. For convenience, you can download and run the
[`client-setup.sh`](https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/client-setup.sh)
script in your local environment:

```shell
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/client-setup.sh > client-setup.sh
chmod +x client-setup.sh
./client-setup.sh
```

{{% notice tip %}}
Running this script can cause existing `pgo` client binary, `pgouser`,
`client.crt`, and `client.key` files to be overwritten.
{{% /notice %}}

The `client-setup.sh` script performs the following tasks:

- Sets `$PGO_OPERATOR_NAMESPACE` to `pgo` if it is unset. This is the default
namespace that the PostgreSQL Operator is deployed to
- Checks for valid Operating Systems and determines which `pgo` binary to
download
- Creates a directory in `$HOME/.pgo/$PGO_OPERATOR_NAMESPACE` (e.g. `/home/hippo/.pgo/pgo`)
- Downloads the `pgo` binary, saves it to in `$HOME/.pgo/$PGO_OPERATOR_NAMESPACE`,
and sets it to be executable
- Pulls the TLS keypair from the PostgreSQL Operator `pgo.tls` Secret so that
the `pgo` client can communicate with the PostgreSQL Operator. These are saved
as `client.crt` and `client.key` in the `$HOME/.pgo/$PGO_OPERATOR_NAMESPACE`
path.
- Pulls the `pgouser` credentials from the `pgouser-admin` secret and saves them
in the format `username:password` in a file called `pgouser`
- `client.crt`, `client.key`, and `pgouser` are all set to be read/write by the
file owner. All other permissions are removed.
- Sets the following environmental variables with the following values:

```shell
export PGOUSER=$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/pgouser
export PGO_CA_CERT=$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.crt
export PGO_CLIENT_CERT=$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.crt
export PGO_CLIENT_KEY=$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.key
```

For convenience, after the script has finished, you can permanently at these
environmental variables to your environment:


```shell
cat <<EOF >> ~/.bashrc
export PATH="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE:$PATH"
export PGOUSER="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/pgouser"
export PGO_CA_CERT="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.crt"
export PGO_CLIENT_CERT="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.crt"
export PGO_CLIENT_KEY="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.key"
EOF
```

By default, the `client-setup.sh` script targets the user that is stored in the
`pgouser-admin` secret in the `pgo` (`$PGO_OPERATOR_NAMESPACE`) Namespace. If
you wish to use a different Secret, you can set the `PGO_USER_ADMIN`
environmental variable.

For more detailed information about [installing the `pgo` client]({{< relref "/installation/pgo-client" >}}),
please see [Installing the `pgo` client]({{< relref "/installation/pgo-client" >}}).

### Verify the Installation

One way to verify the installation was successful is to execute the
[`pgo version`]({{< relref "/pgo-client/reference/pgo_version.md" >}}) command.

In a new console window, run the following command to set up a port forward:

```shell
kubectl -n pgo port-forward svc/postgres-operator 8443:8443
```

Next, in another console window, set the following environment variable to configure the API server address:

```bash
cat <<EOF >> ${HOME?}/.bashrc
export PGO_APISERVER_URL="https://127.0.0.1:8443"
EOF
```

Apply those changes to the current session by running:

```bash
source ${HOME?}/.bashrc
```

Now run the `pgo version` command:

```shell
pgo version
```

If successful, you should see output similar to this:

```
pgo client version {{< param operatorVersion >}}
pgo-apiserver version {{< param operatorVersion >}}
```

## Post-Installation

To clean up the installer artifacts, you can simply run:

```shell
kubectl delete -f /path/to/postgres-operator.yml
```

Note that if you still have the ServiceAccount and ClusterRoleBinding in there,
you will need to have elevated privileges.

## Installing the PGO Monitoring Infrastructure

Please see the [PostgreSQL Operator Monitoring installation section]({{< relref "/installation/metrics" >}})
for instructions on how to install the PostgreSQL Operator Monitoring infrastructure.
