---
title: "Prerequisites"
date:
draft: false
weight: 10
---

# Prerequisites

The following is required prior to installing Crunchy PostgreSQL Operator using Ansible:

* [postgres-operator playbooks](https://github.com/CrunchyData/postgres-operator/) source code for the target version
* Ansible 2.4.6+

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

The installation of the Crunchy PostgreSQL Operator requires elevated privileges.  
It is required that the playbooks are run as a `cluster-admin` to ensure the playbooks 
can install:

* Custom Resource Definitions
* Cluster RBAC
* Create required namespaces

## Obtaining Operator Ansible Role

The Crunchy PostgreSQL Operator Roles are available here:

* Clone the [postgres-operator project](https://github.com/CrunchyData/postgres-operator)


### GitHub Installation

All necessary files (inventory, main playbook and roles) can be found in the `ansible` 
directory in the  [postgres-operator project](https://github.com/CrunchyData/postgres-operator).

## Configuring the Inventory File

The `inventory` file included with the PostgreSQL Operator Playbooks allows installers 
to configure how the operator will function when deployed into Kubernetes.  This file 
should contain all configurable variables the playbooks offer.

The following are the variables available for configuration:

| Name                              | Default     | Description                                                                                                                                                                      |
|-----------------------------------|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `archive_mode`                    | true        | Set to true enable archive logging on all newly created clusters.                                                                                                                |
| `archive_timeout`                 | 60          | Set to a value in seconds to configure the timeout threshold for archiving.                                                                                                      |
| `auto_failover_replace_replica`   | false       | Set to true to replace promoted replicas during failovers with a new replica on all newly created clusters.                                                                      |
| `auto_failover_sleep_secs`        | 9           | Set to a value in seconds to configure the sleep time before initiating a failover on all newly created clusters.                                                                |
| `auto_failover`                   | false       | Set to true enable auto failover capabilities on all newly created cluster requests.  This can be disabled by the client.                                                        |
| `backrest`                        | false       | Set to true enable pgBackRest capabilities on all newly created cluster request.  This can be disabled by the client.                                                            |
| `backrest_aws_s3_key`             |             | Set to configure the key used by pgBackRest to authenticate with Amazon Web Service S3 for backups and restoration in S3.                                                        |
| `backrest_aws_s3_secret`          |             | Set to configure the secret used by pgBackRest to authenticate with Amazon Web Service S3 for backups and restoration in S3.                                                     |
| `backrest_aws_s3_bucket`          |             | Set to configure the bucket used by pgBackRest with Amazon Web Service S3 for backups and restoration in S3.                                                                     |
| `backrest_aws_s3_endpoint`        |             | Set to configure the endpoint used by pgBackRest with Amazon Web Service S3 for backups and restoration in S3.                                                                   |
| `backrest_aws_s3_region`          |             | Set to configure the region used by pgBackRest with Amazon Web Service S3 for backups and restoration in S3.                                                                     |
| `backrest_storage`                | storage1    | Set to configure which storage definition to use when creating volumes used by pgBackRest on all newly created clusters.                                                         |
| `badger`                          | false       | Set to true enable pgBadger capabilities on all newly created clusters.  This can be disabled by the client.                                                                     |
| `ccp_image_prefix`                | crunchydata | Configures the image prefix used when creating containers from Crunchy Container Suite.                                                                                          |
| `ccp_image_tag`                   |             | Configures the image tag (version) used when creating containers from Crunchy Container Suite.                                                                                   |
| `cleanup`                         | false       | Set to configure the playbooks to delete all objects when deprovisioning Operator.  Note: this will delete all objects related to the Operator (including clusters provisioned). |
| `crunchy_debug`                   | false       | Set to configure Operator to use debugging mode.  Note: this can cause sensitive data such as passwords to appear in Operator logs.                                              |
| `db_name`                         | userdb      | Set to a value to configure the default database name on all newly created clusters.                                                                                             |
| `db_password_age_days`            | 60          | Set to a value in days to configure the expiration age on PostgreSQL role passwords on all newly created clusters.                                                               |
| `db_password_length`              | 20          | Set to configure the size of passwords generated by the operator on all newly created roles.                                                                                     |
| `db_port`                         | 5432        | Set to configure the default port used on all newly created clusters.                                                                                                            |
| `db_replicas`                     | 1           | Set to configure the amount of replicas provisioned on all newly created clusters.                                                                                               |
| `db_user`                         | testuser    | Set to configure the username of the dedicated user account on all newly created clusters.                                                                                       |
| `grafana_admin_username`          | admin       | Set to configure the login username for the Grafana administrator.														                                                         |
| `grafana_admin_password`          |             | Set to configure the login password for the Grafana administrator.														                                                         |
| `grafana_install`                 | true        | Set to true to install Crunchy Grafana to visualize metrics.                                                                                                                     |
| `grafana_storage_access_mode`     |             | Set to the access mode used by the configured storage class for Grafana persistent volumes.                                                                                      |
| `grafana_storage_class_name`      |             | Set to the name of the storage class used when creating Grafana persistent volumes.                                                                                              |
| `grafana_volume_size`             |             | Set to the size of persistent volume to create for Grafana.                                                                                                                      |
| `kubernetes_context`              |             | When deploying to Kubernetes, set to configure the context name of the kubeconfig to be used for authentication.                                                                 |
| `log_statement`                   | none        | Set to `none`, `ddl`, `mod`, or `all` to configure the statements that will be logged in PostgreSQL's logs on all newly created clusters.                                        |
| `metrics`                         | false       | Set to true enable performance metrics on all newly created clusters.  This can be disabled by the client.                                                                       |
| `metrics_namespace`               | metrics     | Configures the target namespace when deploying Grafana and/or Prometheus                                                                                                         |
| `namespace`                       |             | Set to a comma delimited string of all the namespaces Operator will manage.                                                                                                      |
| `openshift_host`                  |             | When deploying to OpenShift, set to configure the hostname of the OpenShift cluster to connect to.                                                                               |
| `openshift_password`              |             | When deploying to OpenShift, set to configure the password used for login.                                                                                                       |
| `openshift_skip_tls_verify`       |             | When deploying to Openshift, set to ignore the integrity of TLS certificates for the OpenShift cluster.                                                                          |
| `openshift_token`                 |             | When deploying to OpenShift, set to configure the token used for login (when not using username/password authentication).                                                        |
| `openshift_user`                  |             | When deploying to OpenShift, set to configure the username used for login.                                                                                                       |
| `pgo_admin_username`              | admin       | Configures the pgo administrator username.                                                                                                                                       |
| `pgo_admin_password`              |             | Configures the pgo administrator password.                                                                                                                                       |
| `pgo_client_install`              | true        | Configures the playbooks to install the `pgo` client if set to true.                                                                                                             |
| `pgo_client_version`              |             | Configures which version of `pgo` the playbooks should install.                                                                                                                  |
| `pgo_image_prefix`                | crunchydata | Configures the image prefix used when creating containers for the Crunchy PostgreSQL Operator (apiserver, operator, scheduler..etc).                                             |
| `pgo_image_tag`                   |             | Configures the image tag used when creating containers for the Crunchy PostgreSQL Operator (apiserver, operator, scheduler..etc)                                                 |
| `pgo_operator_namespace`          |             | Set to configure the namespace where Operator will be deployed.                                                                                                                  |
| `pgo_tls_no_verify`               |             | Set to configure Operator to verify TLS certificates.                                                                                                                            |
| `primary_storage`                 | storage2    | Set to configure which storage definition to use when creating volumes used by PostgreSQL primaries on all newly created clusters.                                               |
| `prometheus_install`              | true        | Set to true to install Crunchy Prometheus timeseries database.                                                                                                                   |
| `prometheus_storage_access_mode`  |             | Set to the access mode used by the configured storage class for Prometheus persistent volumes.                                                                                   |
| `prometheus_storage_class_name`   |             | Set to the name of the storage class used when creating Prometheus persistent volumes.                                                                                           |
| `replica_storage`                 | storage3    | Set to configure which storage definition to use when creating volumes used by PostgreSQL replicas on all newly created clusters.                                                |
| `scheduler_timeout`               | 3600        | Set to a value in seconds to configure the `pgo-scheduler` timeout threshold when waiting for schedules to complete.                                                             |
| `service_type`                    | ClusterIP   | Set to configure the type of Kubernetes service provisioned on all newly created clusters.                                                                                       |
| `storage<ID>_access_mode`         |             | Set to configure the access mode of the volumes created when using this storage definition.                                                                                      |
| `storage<ID>_class`               |             | Set to configure the storage class name used when creating dynamic volumes.                                                                                                      |
| `storage<ID>_fs_group`            |             | Set to configure any filesystem groups that should be added to security contexts on newly created clusters.                                                                      |
| `storage<ID>_size`                |             | Set to configure the size of the volumes created when using this storage definition.                                                                                             |
| `storage<ID>_supplemental_groups` |             | Set to configure any supplemental groups that should be added to security contexts on newly created clusters.                                                                    |
| `storage<ID>_type`                |             | Set to either `create` or `dynamic` to configure the operator to create persistent volumes or have them created dynamically by a storage class.                                  |

{{% notice tip %}}
To retrieve the `kubernetes_context` value for Kubernetes installs, run the following command:

```bash
kubectl config current-context
```
{{% /notice %}}

### Minimal Variable Requirements

The following variables should be configured at a minimum to deploy the Crunchy 
PostgreSQL Operator:

* `backrest_storage`
* `ccp_image_prefix`
* `ccp_image_tag`
* `kubernetes_context`
* `namespace`
* `openshift_host`
* `openshift_password`
* `openshift_skip_tls_verify`
* `openshift_token`
* `openshift_user`
* `pgo_admin_username`
* `pgo_admin_password`
* `pgo_client_install`
* `pgo_image_prefix`
* `pgo_image_tag`
* `pgo_operator_namespace`
* `pgo_tls_no_verify`
* `primary_storage`
* `replica_storage`
* `storage<ID>_access_mode`
* `storage<ID>_class`
* `storage<ID>_fs_group`
* `storage<ID>_size`
* `storage<ID>_supplemental_groups`
* `storage<ID>_type`

{{% notice tip %}}
Users should remove or comment out the `kubernetes` or `openshift` variables if they're not being used 
from the inventory file.  Both sets of variables cannot be used at the same time.
{{% /notice %}}

## Storage

Kubernetes and OpenShift offer support for a variety of storage types.  The Crunchy 
PostgreSQL Operator must be configured to utilize the storage options available by 
configuring the `storage` options included in the inventory file.

{{% notice tip %}}
At this time the Crunchy PostgreSQL Operator Playbooks only support storage classes.  
For more information on storage classes see the [official Kubernetes documentation](https://kubernetes.io/docs/concepts/storage/storage-classes/).
{{% /notice %}}

### Considerations for Multi-Zone Cloud Environments

When using the Operator in a Kubernetes cluster consisting of nodes that span 
multiple zones, special consideration must betaken to ensure all pods and the 
volumes they require are scheduled and provisioned within the same zone.  Specifically,
being that a pod is unable mount a volume that is located in another zone, any 
volumes that are dynamically provisioned must be provisioned in a topology-aware 
manner according to the specific scheduling requirements for the pod. For instance, 
this means ensuring that the volume containing the database files for the primary 
database in a new PostgreSQL cluster is provisioned in the same zone as the node 
containing the PostgreSQL primary pod that will be using it.

For instructions on setting up storage classes for multi-zone environments, see 
the [PostgreSQL Operator Documentation](/gettingstarted/design/designoverview/).

### Examples

The following are examples on configuring the storage variables for different types 
of storage classes.

#### Generic Storage Class

To setup storage1 to use the storage class `fast`

```ini
storage1_access_mode='ReadWriteOnce'
storage1_size='10G'
storage1_type='dynamic'
storage1_class='fast'
```

To assign this storage definition to all `primary` pods created by the Operator, we 
can configure the `primary_storage=storage1` variable in the inventory file.

#### GKE

The storage class provided by Google Kubernetes Environment (GKE) can be configured 
to be used by the Operator by setting the following variables in the `inventory` file:

```ini
storage1_access_mode='ReadWriteOnce'
storage1_size='10G'
storage1_type='dynamic'
storage1_class='standard'
storage1_fs_group=26
```

To assign this storage definition to all `primary` pods created by the Operator, we 
can configure the `primary_storage=storage1` variable in the inventory file.

{{% notice tip %}}
To utitlize mutli-zone deployments, see _Considerations for Multi-Zone Cloud Environments_ above.
{{% /notice %}}

## Understanding `pgo_operator_namespace` & `namespace`

The Crunchy PostgreSQL Operator can be configured to be deployed and manage a single 
namespace or manage several namespaces.  The following are examples of different types 
of deployment models configurable in the `inventory` file.

### Single Namespace

To deploy the Crunchy PostgreSQL Operator to work with a single namespace (in this example 
our namespace is named `pgo`), configure the following `inventory` settings:

```ini
pgo_operator_namespace='pgo'
namespace='pgo'
```

### Multiple Namespaces

To deploy the Crunchy PostgreSQL Operator to work with multiple namespaces (in this example
our namespaces are named `pgo`, `pgouser1` and `pgouser2`), configure the following `inventory` settings:

```ini
pgo_operator_namespace='pgo'
namespace='pgouser1,pgouser2'
```

## Deploying Multiple Operators

The 4.0 release of the Crunchy PostgreSQL Operator allows for multiple operator deployments in the same cluster.  
To install the Crunchy PostgreSQL Operator to multiple namespaces, it's recommended to have an `inventory` file 
for each deployment of the operator.

For each operator deployment the following inventory variables should be configured uniquely for each install.

For example, operator could be deployed twice by changing the `pgo_operator_namespace` and `namespace` for those 
deployments:

Inventory A would deploy operator to the `pgo` namespace and it would manage the `pgo` target namespace.

```init
# Inventory A
pgo_operator_namespace='pgo'
namespace='pgo'
...
```

Inventory B would deploy operator to the `pgo2` namespace and it would manage the `pgo2` and `pgo3` target namespaces.
```init
# Inventory B
pgo_operator_namespace='pgo2'
namespace='pgo2,pgo3'
...
```

Each install of the operator will create a corresponding directory in `$HOME/.pgo/<PGO NAMESPACE>` which will contain 
the TLS and `pgouser` client credentials.

## Deploying Grafana and Prometheus

PostgreSQL clusters created by the operator can be configured to create additional containers for collecting metrics.  
These metrics are very useful for understanding the overall health and performance of PostgreSQL database deployments 
over time.  The collectors included by the operator are:

* Node Exporter - Host metrics where the PostgreSQL containers are running
* PostgreSQL Exporter - PostgreSQL metrics

The operator, however, does not install the necessary timeseries database (Prometheus) for storing the collected 
metrics or the front end visualization (Grafana) of those metrics.

Included in these playbooks are roles for deploying Granfana and/or Prometheus.  See the `inventory` file 
for options to install the metrics stack.

{{% notice tip %}}
At this time the Crunchy PostgreSQL Operator Playbooks only support storage classes.
{{% /notice %}}
