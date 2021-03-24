---
title: "Metrics Prerequisites"
date:
draft: false
weight: 10
---

# Prerequisites

The following is required prior to installing the Crunchy PostgreSQL Operator Monitoring infrastructure using Ansible:

* [postgres-operator playbooks](https://github.com/CrunchyData/postgres-operator/) source code for the target version
* Ansible 2.9.0+

## Kubernetes Installs

* Kubernetes v1.11+
* Cluster admin privileges in Kubernetes
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/) configured to communicate with Kubernetes

## OpenShift Installs

* OpenShift v3.11+
* Cluster admin privileges in OpenShift
* [oc](https://docs.openshift.com/container-platform/3.11/cli_reference/get_started_cli.html) configured to communicate with OpenShift

## Installing from a Windows Host

If the Crunchy PostgreSQL Operator is being installed from a Windows host the following
are required:

* [Windows Subsystem for Linux (WSL)](https://docs.microsoft.com/en-us/windows/wsl/install-win10)
* [Ubuntu for Windows](https://www.microsoft.com/en-us/p/ubuntu/9nblggh4msv6)

## Permissions

The installation of the Crunchy PostgreSQL Operator Monitoring infrastructure requires elevated
privileges, as the following objects need to be created:

* RBAC for use by Prometheus and/or Grafana
* The metrics namespace

## Obtaining Operator Ansible Role

* Clone the [postgres-operator project](https://github.com/CrunchyData/postgres-operator)

### GitHub Installation

All necessary files (inventory.yaml, values.yaml, main playbook and roles) can be found in the
[`installers/metrics/ansible`](https://github.com/CrunchyData/postgres-operator/tree/master/installers/metrics/ansible)
directory in the [source code](https://github.com/CrunchyData/postgres-operator).

## Configuring the Inventory File

The `inventory.yaml` file included with the PostgreSQL Operator Monitoring Playbooks allows installers
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
for deploying the PostgreSQL Operator Monitoring infrastructure.
The [example file](https://github.com/CrunchyData/postgres-operator/blob/v{{< param operatorVersion >}}/installers/metrics/ansible/values.yaml)
contains defaults that should work in most Kubernetes environments, but it may
require some customization.

Note that in OpenShift and CodeReady Containers you will need to set the
`disable_fsgroup` to `true` attribute to `true` if you are using the
`restricted` Security Context Constraint (SCC). If you are using the `anyuid`
SCC, you will need to set `disable_fsgroup` to `false`.

For a detailed description of each configuration parameter, please read the
[PostgreSQL Operator Installer Metrics Configuration Reference](<{{< relref "/installation/metrics/metrics-configuration.md">}}>)
