---
title: "Prerequisites"
date:
draft: false
weight: 10
---

# Prerequisites

The following is required prior to installing Crunchy PostgreSQL Operator using Ansible:

* [postgres-operator playbooks](https://github.com/CrunchyData/postgres-operator/) source code for the target version
* Ansible 2.9.0+

## Kubernetes Installs

* Kubernetes v1.11+
* Cluster admin privileges in Kubernetes
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) configured to communicate with Kubernetes

## OpenShift Installs

* OpenShift v3.09+
* Cluster admin privileges in OpenShift
* [oc](https://docs.openshift.com/container-platform/3.11/cli_reference/get_started_cli.html) configured to communicate with OpenShift

## Installing from a Windows Host

If the Crunchy PostgreSQL Operator is being installed from a Windows host the following
are required:

* [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install-win10)
* [Ubuntu for Windows](https://www.microsoft.com/en-us/p/ubuntu/9nblggh4msv6)

## Permissions

The installation of the Crunchy PostgreSQL Operator requires elevated
privileges, as the following objects need to be created:

* Custom Resource Definitions
* Cluster RBAC for using one of the multi-namespace modes
* Create required namespaces

{{% notice warning %}}In Kubernetes versions prior to 1.12 (including Openshift up through 3.11), there is a limitation that requires an extra step during installation for the operator to function properly with watched namespaces. This limitation does not exist when using Kubernetes 1.12+. When a list of namespaces are provided through the NAMESPACE environment variable, the setupnamespaces.sh script handles the limitation properly in both the bash and ansible installation.

However, if the user wishes to add a new watched namespace after installation, where the user would normally use pgo create namespace to add the new namespace, they should instead run the add-targeted-namespace.sh script or they may give themselves cluster-admin privileges instead of having to run setupnamespaces.sh script. Again, this is only required when running on a Kubernetes distribution whose version is below 1.12. In Kubernetes version 1.12+ the pgo create namespace command works as expected.

{{% /notice %}}

## Obtaining Operator Ansible Role

* Clone the [postgres-operator project](https://github.com/CrunchyData/postgres-operator)

### GitHub Installation

All necessary files (inventory.yaml, values.yaml, main playbook and roles) can be found in the
[`installers/ansible`](https://github.com/CrunchyData/postgres-operator/tree/master/installers/ansible) directory
in the [source code](https://github.com/CrunchyData/postgres-operator).

## Configuring the Inventory File

The `inventory.yaml` file included with the PostgreSQL Operator Playbooks allows installers
to configure how Ansible will connect to your Kubernetes cluster.  This file
should contain the following connection variables:

{{% notice tip %}}
You will have to uncomment out either the `kubernetes` or `openshift` variables
if you are being using them for your environment. Both sets of variables cannot
be used at the same time. The unused variables should be left commented out or removed.
{{% /notice %}}


| Name                              | Default     | Required |  Description                                                                                                                                                                      |
|-----------------------------------|-------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `kubernetes_context`              |             | **Required**, if deploying to Kubernetes |When deploying to Kubernetes, set to configure the context name of the kubeconfig to be used for authentication.                                                                 |
| `openshift_host`                  |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the hostname of the OpenShift cluster to connect to.                                                                               |
| `openshift_password`              |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the password used for login.                                                                                                       |
| `openshift_skip_tls_verify`       |             | **Required**, if deploying to OpenShift | When deploying to Openshift, set to ignore the integrity of TLS certificates for the OpenShift cluster.                                                                          |
| `openshift_token`                 |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the token used for login (when not using username/password authentication).                                                        |
| `openshift_user`                  |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the username used for login.                                                                                                       |

{{% notice tip %}}
To retrieve the `kubernetes_context` value for Kubernetes installs, run the following command:

```bash
kubectl config current-context
```
{{% /notice %}}

## Configuring - `values.yaml`

The `values.yaml` file contains all of the configuration parameters
for deploying the PostgreSQL Operator. The [example file](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/ansible/values.yaml)
contains defaults that should work in most Kubernetes environments, but it may
require some customization.

For a detailed description of each configuration parameter, please read the
[PostgreSQL Operator Installer Configuration Reference](<{{< relref "/installation/configuration.md">}}>)
