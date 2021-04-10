---
title: "Helm Chart"
date:
draft: false
weight: 100
---

# PGO: The Postgres Operator Helm Chart

## Overview

PGO, the Postgres Operator from Crunchy Data, comes with a
container called `pgo-deployer` which handles a variety of
lifecycle actions for the PostgreSQL Operator, including:

- Installation
- Upgrading
- Uninstallation

After configuring the `values.yaml` file with you configuration options, the
installer will be run using the `helm` command line tool and takes care of
setting up all of the objects required to run the PostgreSQL Operator.

The `postgres-operator` Helm chart is available in the [Helm](https://github.com/CrunchyData/postgres-operator/tree/master/installers/helm)
directory in the PostgreSQL Operator repository.

## Requirements

### RBAC

The Helm chart will create the ServiceAccount, ClusterRole, and ClusterRoleBinding
that are required to run the `pgo-deployer`. If you have already configured the
ServiceAccount and ClusterRoleBinding for the installation process (e.g. from a
previous installation), you can disable their creation using the `rbac.create`
and `serviceAccount.create` variables in the `values.yaml` file. If these options
are disabled, you must provide the name of your preconfigured ServiceAccount using
`serviceAccount.name`.

### Namespace

In order to install the PostgreSQL Operator using the Helm chart you will need
to first create the namespace in which the `pgo-deployer` will be run. By default,
it will run in the namespace that is provided to `helm` at the command line.

```
kubectl create namespace <namespace>
helm install postgres-operator -n <namespace> /path/to/chart_dir
```

The PostgreSQL Operator has the ability to manage PostgreSQL clusters across
multiple Kubernetes [Namespaces](https://kubernetes.io/docs/concepts/overview/working-with-objects/namespaces/),
including the ability to add and remove Namespaces that it watches. Doing so
does require the PostgreSQL Operator to have elevated privileges, and as such,
the PostgreSQL Operator comes with three "namespace modes" to select what level
of privileges to provide. Detailed information about these "namespace modes"
can be found in the [Namespace](<{{< relref "/installation/postgres-operator.md" >}}>)
section here.

### Config Map

The `pgo-deployer` uses a [Kubernetes ConfigMap](https://kubernetes.io/docs/concepts/configuration/configmap/)
to pass configuration options into the installer. The values in your `values.yaml`
file will be used to populate the configuation options in the ConfigMap.

### Configuration - `values.yaml`

The `values.yaml` file contains all of the configuration parametes for deploying
the PostgreSQL Operator. The [values.yaml file](https://github.com/CrunchyData/postgres-operator/blob/master/installers/helm/values.yaml) contains the defaults that
should work in most Kubernetes environments, but it may require some customization.

For a detailed description of each configuration parameter, please read the
[PostgreSQL Operator Installer Configuration Reference](<{{< relref "/installation/configuration.md">}}>)

## Installation

Once you have configured the PostgreSQL Operator Installer to your
specification, you can install the PostgreSQL Operator with the following
command:

```shell
helm install <name> -n <namespace> /path/to/chart_dir
```

{{% notice tip %}}
Take note of the `name` used when installing, this `name` will be used to
upgrade and uninstall the PostgreSQL Operator.
{{% /notice %}}

### Install the [`pgo` Client]({{< relref "/installation/pgo-client" >}})

To use the [`pgo` Client]({{< relref "/installation/pgo-client" >}}),
there are a few additional steps to take in order to get it to work with your
PostgreSQL Operator installation. For convenience, you can download and run the
[`client-setup.sh`](https://raw.githubusercontent.com/CrunchyData/postgres-operator/master/installers/kubectl/client-setup.sh)
script in your local environment:

```shell
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/master/installers/kubectl/client-setup.sh > client-setup.sh
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

For convenience, after the script has finished, you can permanently add these
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

In another console window, run the `pgo version` command:

```shell
pgo version
```

If successful, you should see output similar to this:

```
pgo client version {{< param operatorVersion >}}
pgo-apiserver version {{< param operatorVersion >}}
```

## Upgrade and Uninstall

Once install has be completed using Helm, it will also be used to upgrade and
uninstall your PostgreSQL Operator.

{{% notice tip %}}
The `name` and `namespace` in the following sections should match the options
provided at install.
{{% /notice %}}

### Upgrade

To make changes to your deployment of the PostgreSQL Operator you will use the
`helm upgrade` command. Once the configuration changes have been made to you
`values.yaml` file, you can run the following command to implement them in the
deployment:

```shell
helm upgrade <name> -n <namespace> /path/to/updated_chart
```

### Uninstall

To uninstall the PostgreSQL Operator you will use the `helm uninstall` command.
This will uninstall the operator and clean up resources used by the `pgo-deployer`.

```shell
helm uninstall <name> -n <namespace>
```

## Debugging

When the `pgo-deployer` job does not complete successfully, the resources that
are created and normally cleaned up by Helm will be left in your
Kubernetes cluster. This will allow you to use the failed job and its logs to
debug the issue. The following command will show the logs for the `pgo-deployer`
job:

```shell
kubectl logs -n <namespace> job.batch/pgo-deploy
```

{{% notice tip %}}
You can also view the logs as the job is running by using the `kubectl -f`
follow flag:
```shell
kubectl logs -n <namespace> job.batch/pgo-deploy -f
```
{{% /notice %}}


These logs will provide feedback if there are any misconfigurations in your
install. Once you have finished debugging the failed job and fixed any configuration
issues, you can take steps to re-run your install, upgrade, or uninstall. By
running another command the resources from the failed install will be cleaned up
so that a successfull install can run.
