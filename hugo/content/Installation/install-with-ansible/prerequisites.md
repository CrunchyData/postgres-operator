---
title: "Prerequisites"
date:
draft: false
weight: 10
---

# Prerequisites

The following is required prior to installing Crunchy PostgreSQL Operator using Ansible:

* [postgres-operator playbooks](https://github.com/CrunchyData/postgres-operator/) source code for the target version
* Ansible 2.7.0+

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

## Environment

Ensure the appropriate [environment variables]({{< relref "common-env.md" >}}) are set.

## Permissions

The installation of the Crunchy PostgreSQL Operator requires elevated privileges.  
It is required that the playbooks are run as a `cluster-admin` to ensure the playbooks
can install:

* Custom Resource Definitions
* Cluster RBAC
* Create required namespaces

{{% notice warning %}}In Kubernetes versions prior to 1.12 (including Openshift up through 3.11), there is a limitation that requires an extra step during installation for the operator to function properly with watched namespaces. This limitation does not exist when using Kubernetes 1.12+. When a list of namespaces are provided through the NAMESPACE environment variable, the setupnamespaces.sh script handles the limitation properly in both the bash and ansible installation.

However, if the user wishes to add a new watched namespace after installation, where the user would normally use pgo create namespace to add the new namespace, they should instead run the add-targeted-namespace.sh script or they may give themselves cluster-admin privileges instead of having to run setupnamespaces.sh script. Again, this is only required when running on a Kubernetes distribution whose version is below 1.12. In Kubernetes version 1.12+ the pgo create namespace command works as expected.

{{% /notice %}}

## Obtaining Operator Ansible Role

There are two ways to obtain the Crunchy PostgreSQL Operator Roles:

* Clone the [postgres-operator project](https://github.com/CrunchyData/postgres-operator)

* `postgres-operator-playbooks` RPM provided for Crunchy customers via the [Crunchy Access Portal](https://access.crunchydata.com/).

### GitHub Installation

All necessary files (inventory, main playbook and roles) can be found in the `ansible`
directory in the  [postgres-operator project](https://github.com/CrunchyData/postgres-operator).

### RPM Installation using Yum

Available to Crunchy customers is an RPM containing all the necessary Ansible roles
and files required for installation using Ansible.  The RPM can be found in Crunchy's
yum repository.  For information on setting up `yum` to use the Crunchy repoistory,
see the [Crunchy Access Portal](https://access.crunchydata.com/).

To install the Crunchy PostgreSQL Operator Ansible roles using `yum`, run the following
command on a RHEL or CentOS host:

```bash
sudo yum install postgres-operator-playbooks
```

* Ansible roles can be found in: `/usr/share/ansible/roles/crunchydata`
* Ansible playbooks/inventory files can be found in: `/usr/share/ansible/postgres-operator/playbooks`

Once installed users should take a copy of the `inventory` file included in the installation
using the following command:

```bash
cp /usr/share/ansible/postgres-operator/playbooks/inventory ${HOME?}
```

## Configuring the Inventory File

The `inventory` file included with the PostgreSQL Operator Playbooks allows installers
to configure how the operator will function when deployed into Kubernetes.  This file
should contain all configurable variables the playbooks offer.

### Requirements

The following configuration parameters must be set in order to deploy the
Crunchy PostgreSQL Operator.

Additionally, `storage` variables will need to be defined to provide the Crunchy
PostgreSQL Operator with any required storage configuration.  Guidance for
defining `storage` variables can be found further in this documentation.

{{% notice tip %}}
You should remove or comment out variables either either the  `kubernetes` or
`openshift` variables if you are not being using them for your environment. Both
sets of variables cannot be used at the same time.
{{% /notice %}}

* `archive_mode`
* `archive_timeout`
* `backup_storage`
* `backrest`
* `backrest_storage`
* `badger`
* `ccp_image_prefix`
* `ccp_image_tag`
* `create_rbac`
* `db_name`
* `db_password_age_days`
* `db_password_length`
* `db_port`
* `db_replicas`
* `db_user`
* `disable_auto_failover``
* `exporterport`
* `kubernetes_context` (Comment out if deploying to am OpenShift environment)
* `metrics`
* `openshift_host` (Comment out if deploying to a Kubernetes environment)
* `openshift_password` (Comment out if deploying to a Kubernetes environment)
* `openshift_skip_tls_verify` (Comment out if deploying to a Kubernetes environment)
* `openshift_token` (Comment out if deploying to a Kubernetes environment)
* `openshift_user` (Comment out if deploying to a Kubernetes environment)
* `pgbadgerport`
* `pgo_admin_password`
* `pgo_admin_perms`
* `pgo_admin_role_name`
* `pgo_admin_username`
* `pgo_client_version`
* `pgo_image_prefix`
* `pgo_image_tag`
* `pgo_installation_name`
* `pgo_operator_namespace`
* `primary_storage`
* `replica_storage`
* `scheduler_timeout`

### Configuration Parameters

| Name                              | Default     | Required |  Description                                                                                                                                                                      |
|-----------------------------------|-------------|----------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `archive_mode`                    | true        | **Required** | Set to true enable archive logging on all newly created clusters.                                                                                                                |
| `archive_timeout`                 | 60          | **Required** | Set to a value in seconds to configure the timeout threshold for archiving.                                                                                                      |
| `backrest`                        | false       | **Required** | Set to true enable pgBackRest capabilities on all newly created cluster request.  This can be disabled by the client.                                                            |
| `backrest_aws_s3_bucket`          |             |          | Set to configure the bucket used by pgBackRest with Amazon Web Service S3 for backups and restoration in S3.                                                                     |
| `backrest_aws_s3_endpoint`        |             |          | Set to configure the endpoint used by pgBackRest with Amazon Web Service S3 for backups and restoration in S3.                                                                   |
| `backrest_aws_s3_key`             |             |          | Set to configure the key used by pgBackRest to authenticate with Amazon Web Service S3 for backups and restoration in S3.                                                        |
| `backrest_aws_s3_region`          |             |          | Set to configure the region used by pgBackRest with Amazon Web Service S3 for backups and restoration in S3.                                                                     |
| `backrest_aws_s3_secret`          |             |          | Set to configure the secret used by pgBackRest to authenticate with Amazon Web Service S3 for backups and restoration in S3.                                                     |
| `backrest_storage`                | storageos   | **Required** | Set to configure which storage definition to use when creating volumes used by pgBackRest on all newly created clusters.                                                         |
| `backup_storage`                  | storageos   | **Required** | Set to configure which storage definition to use when creating volumes used for storing logical backups created by `pg_dump`. |
| `badger`                          | false       | **Required** | Set to true enable pgBadger capabilities on all newly created clusters.  This can be disabled by the client.                                                                     |
| `ccp_image_prefix`                | crunchydata | **Required** | Configures the image prefix used when creating containers from Crunchy Container Suite.                                                                                          |
| `ccp_image_tag`                   |             | **Required** | Configures the image tag (version) used when creating containers from Crunchy Container Suite.                                                                                   |
| `cleanup`                         | false       |          | Set to configure the playbooks to delete all objects when uninstalling the Operator.  Note: this will delete all objects related to the Operator (including clusters provisioned). |
| `create_rbac`                     | true        | **Required** | Set to `true` if the installer should create the RBAC resources required to run the PostgreSQL Operator. |
| `crunchy_debug`                   | false       |          | Set to configure Operator to use debugging mode.  Note: this can cause sensitive data such as passwords to appear in Operator logs.                                              |
| `delete_metrics_namespace`        | false       |          | Set to configure whether or not the metrics namespace (defined using variable `metrics_namespace`) is deleted when uninstalling the metrics infrastructure                     |
| `delete_operator_namespace`       | false       |          | Set to configure whether or not the PGO operator namespace (defined using variable `pgo_operator_namespace`) is deleted when uninstalling the PGO.                             |
| `delete_watched_namespaces`       | false       |          | Set to configure whether or not the PGO watched namespaces (defined using variable `namespace`) are deleted when uninstalling the PGO.                                         |
| `db_name`                         | userdb      | **Required** | Set to a value to configure the default database name on all newly created clusters.                                                                                             |
| `db_password_age_days`            | 60          | **Required** | Set to a value in days to configure the expiration age on PostgreSQL role passwords on all newly created clusters.                                                               |
| `db_password_length`              | 24          | **Required** | Set to configure the size of passwords generated by the operator on all newly created roles.                                                                                     |
| `db_port`                         | 5432        | **Required** | Set to configure the default port used on all newly created clusters.                                                                                                            |
| `db_replicas`                     | 1           | **Required** | Set to configure the amount of replicas provisioned on all newly created clusters.                                                                                               |
| `db_user`                         | testuser    | **Required** | Set to configure the username of the dedicated user account on all newly created clusters.                                                                                       |
| `disable_failover`                | false       | **Required** | Set to true disable auto failover capabilities on all newly created cluster requests. This cannot be overriden by the client on a cluster create, but on a cluster update. Setting this is not generally recommend, as it disable high-availability capabilities.|
| `exporterport`                    | 9187        | **Required** | Set to configure the default port used to connect to postgres exporter.  |
| `grafana_admin_password`          |             |          | Set to configure the login password for the Grafana administrator. |
| `grafana_admin_username`          | admin       |          | Set to configure the login username for the Grafana administrator. |
| `grafana_install`                 | true        |          | Set to true to install Crunchy Grafana to visualize metrics.                                                                                                                     |
| `grafana_storage_access_mode`     |             |          | Set to the access mode used by the configured storage class for Grafana persistent volumes.                                                                                      |
| `grafana_storage_class_name`      |             |          | Set to the name of the storage class used when creating Grafana persistent volumes.                                                                                              |
| `grafana_volume_size`             |             |          | Set to the size of persistent volume to create for Grafana.                                                                                                                      |
| `kubernetes_context`              |             | **Required**, if deploying to Kubernetes |When deploying to Kubernetes, set to configure the context name of the kubeconfig to be used for authentication.                                                                 |
| `log_statement`                   | none        |          | Set to `none`, `ddl`, `mod`, or `all` to configure the statements that will be logged in PostgreSQL's logs on all newly created clusters.                                        |
| `metrics`                         | false       | **Required** | Set to true enable performance metrics on all newly created clusters.  This can be disabled by the client.                                                                       |
| `metrics_namespace`               | metrics     |          | Configures the target namespace when deploying Grafana and/or Prometheus                                                                                                         |
| `namespace`                       |             |          | Set to a comma delimited string of all the namespaces Operator will manage.                                                                                                      |
| `openshift_host`                  |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the hostname of the OpenShift cluster to connect to.                                                                               |
| `openshift_password`              |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the password used for login.                                                                                                       |
| `openshift_skip_tls_verify`       |             | **Required**, if deploying to OpenShift | When deploying to Openshift, set to ignore the integrity of TLS certificates for the OpenShift cluster.                                                                          |
| `openshift_token`                 |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the token used for login (when not using username/password authentication).                                                        |
| `openshift_user`                  |             | **Required**, if deploying to OpenShift | When deploying to OpenShift, set to configure the username used for login.                                                                                                       |
| `pgbadgerport`                    | 10000       | **Required** | Set to configure the default port used to connect to pgbadger.  |
| `pgo_add_os_ca_store`             | false       |          | When true, includes system default certificate authorities          |
| `pgo_admin_username`              | admin       | **Required** | Configures the pgo administrator username.                                                                                                                                       |
| `pgo_admin_password`              |             | **Required** | Configures the pgo administrator password.                                                                                                                                       |
| `pgo_admin_perms`                 | `*` | **Required** | Sets the access control rules provided by the PostgreSQL Operator RBAC resources for the PostgreSQL Operator administrative account that is created by this installer. Defaults to allowing all of the permissions, which is represented with the `*` |
| `pgo_admin_role_name`             | pgoadmin    | **Required** | Sets the name of the PostgreSQL Operator role that is utilized for administrative operations performed by the PostgreSQL Operator. |
| `pgo_apiserver_port`              | 8443        |          | Set to configure the port used by the Crunchy PostgreSQL Operator apiserver.                                                                                                     |
| `pgo_client_install`              | true        |          | Configures the playbooks to install the `pgo` client if set to true.                                                                                                             |
| `pgo_client_version`              |             | **Required** | Configures which version of `pgo` the playbooks should install.                                                                                                                  |
| `pgo_disable_eventing`            | false       |          | Set to configure whether or not eventing should be enabled for the Crunchy PostgreSQL Operator installation.                                                                     |
| `pgo_disable_tls`                 | false       |          | Set to configure whether or not TLS should be enabled for the Crunchy PostgreSQL Operator apiserver.                                                                             |
| `pgo_image_prefix`                | crunchydata | **Required** | Configures the image prefix used when creating containers for the Crunchy PostgreSQL Operator (apiserver, operator, scheduler..etc).                                             |
| `pgo_image_tag`                   |             | **Required** | Configures the image tag used when creating containers for the Crunchy PostgreSQL Operator (apiserver, operator, scheduler..etc)                                                 |
| `pgo_installation_name`           |             | **Required** | The name of the PGO installation.                                                                                                                                                |
| `pgo_noauth_routes`               |             |          | Configures URL routes with mTLS and HTTP BasicAuth disabled.                                  |
| `pgo_operator_namespace`          |             | **Required** | Set to configure the namespace where Operator will be deployed.                                                                                                                  |
| `pgo_tls_ca_store`                |             |          | Set to add additional Certificate Authorities for Operator to trust (PEM-encoded file).                                                                                      |
| `pgo_tls_no_verify`               | false       |          | Set to configure Operator to verify TLS certificates.                                                                                                                            |
| `pgo_client_container_install` | false | | Installs the pgo-client deployment along with ansible isnstall |
| `pgo_apiserver_url` | `https://postgres-operator` | | Sets the `pgo_apiserver_url` in the pgo-client deployment |
| `pgo_client_cert_secret` | `pgo.tls` | | Sets the secret that the pgo-client will use when connecting to the operator. Secret should hold the TLS certs |
| `primary_storage`                 | storageos   | **Required** | Set to configure which storage definition to use when creating volumes used by PostgreSQL primaries on all newly created clusters.                                               |
| `prometheus_install`              | true        |          | Set to true to install Crunchy Prometheus timeseries database.                                                                                                                   |
| `prometheus_storage_access_mode`  |             |          | Set to the access mode used by the configured storage class for Prometheus persistent volumes.                                                                                   |
| `prometheus_storage_class_name`   |             |          | Set to the name of the storage class used when creating Prometheus persistent volumes.                                                                                           |
| `replica_storage`                 | storageos   | **Required** | Set to configure which storage definition to use when creating volumes used by PostgreSQL replicas on all newly created clusters.                                                |
| `scheduler_timeout`               | 3600        | **Required** | Set to a value in seconds to configure the `pgo-scheduler` timeout threshold when waiting for schedules to complete.                                                             |
| `service_type`                    | ClusterIP   |          | Set to configure the type of Kubernetes service provisioned on all newly created clusters.                                                                                       |
| `pgo_cluster_admin`               | false       |          | Determines whether or not the `cluster-admin` role is assigned to the PGO service account. Must be `true` to enable PGO namespace & role creation when installing in OpenShift.  |

{{% notice tip %}}
To retrieve the `kubernetes_context` value for Kubernetes installs, run the following command:

```bash
kubectl config current-context
```
{{% /notice %}}

## Storage

Kubernetes and OpenShift offer support for a wide variety of different storage types, and by default, the `inventory` is
pre-populated with storage configurations for some of these storage types.  However, the storage types defined
in the `inventory` can be modified or removed as needed, while additional storage configurations can also be
added to meet the specific storage requirements for your PG clusters.  

The following `storage` variables are utilized to add or modify operator storage configurations in the `inventory`:

| Name                              | Required    | Description                                                                                                                                                                      |
|-----------------------------------|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `storage<ID>_name` | Yes | Set to specify a name for the storage configuration. |
| `storage<ID>_access_mode` | Yes | Set to configure the access mode of the volumes created when using this storage definition. |
| `storage<ID>_size` | Yes | Set to configure the size of the volumes created when using this storage definition. |
| `storage<ID>_class` | Required when using the `dynamic` storage type | Set to configure the storage class name used when creating dynamic volumes. |
| `storage<ID>_fs_group` | Required when using a storage class | Set to configure any filesystem groups that should be added to security contexts on newly created clusters. |
| `storage<ID>_supplemental_groups` | Required when using NFS storage | Set to configure any supplemental groups that should be added to security contexts on newly created clusters. |
| `storage<ID>_type` | Yes  | Set to either `create` or `dynamic` to configure the operator to create persistent volumes or have them created dynamically by a storage class. |

The `ID` portion of `storage` prefix for each variable name above should be an integer that is used to group the various `storage` variables into a single storage configuration.  For instance, the following shows a single storage configuration for NFS storage:

```ini
storage3_name='nfsstorage'
storage3_access_mode='ReadWriteMany'
storage3_size='1G'
storage3_type='create'
storage3_supplemental_groups=65534
```

As this example storage configuration shows, integer `3` is used as the ID for each of the `storage` variables, which together form a single storage configuration called `nfsstorage`.  This approach allows different storage configurations to be created by defining the proper `storage` variables with a unique ID for each required storage configuration.

Additionally, once all storage configurations have been defined in the `inventory`, they can then be used to specify the default storage configuration that should be utilized for the various PG pods created by the operator.  This is done using the following variables, which are also defined in the `inventory`:

```ini
backrest_storage='nfsstorage'
backup_storage='nfsstorage'
primary_storage='nfsstorage'
replica_storage='nfsstorage'
```

With the configuration shown above, the `nfsstorage` storage configuration would be used by default for the various containers created for a PG cluster (i.e. containers for the primary DB, replica DB's, backups and/or `pgBackRest`).

### Examples

The following are additional examples of storage configurations for various storage types.

#### Generic Storage Class

The following example defines a storageTo setup storage1 to use the storage class `fast`

```ini
storage5_name='storageos'
storage5_access_mode='ReadWriteOnce'
storage5_size='300M'
storage5_type='dynamic'
storage5_class='fast'
storage5_fs_group=26
```

To assign this storage definition to all `primary` pods created by the Operator, we
can configure the `primary_storage=storageos` variable in the inventory file.

#### GKE

The storage class provided by Google Kubernetes Environment (GKE) can be configured
to be used by the Operator by setting the following variables in the `inventory` file:

```ini
storage8_name='gce'
storage8_access_mode='ReadWriteOnce'
storage8_size='300M'
storage8_type='dynamic'
storage8_class='standard'
storage8_fs_group=26
```

To assign this storage definition to all `primary` pods created by the Operator, we
can configure the `primary_storage=gce` variable in the inventory file.

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

## Resource Configuration

Kubernetes and OpenShift allow specific resource requirements to be specified for the various containers deployed inside of a pod.
This includes defining the required resources for each container, i.e. how much memory and CPU each container will need, while also
allowing resource limits to be defined, i.e. the maximum amount of memory and CPU a container will be allowed to consume.
In support of this capability, the Crunchy PGO allows any required resource configurations to be defined in the `inventory`, which
can the be utilized by the operator to set any desired resource requirements/limits for the various containers that will
be deployed by the Crunchy PGO when creating and managing PG clusters.

The following `resource` variables are utilized to add or modify operator resource configurations in the `inventory`:

| Name                              | Required    | Description                                                                                                                                                                      |
|-----------------------------------|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `resource<ID>_requests_memory` | Yes | The amount of memory required by the container. |
| `resource<ID>_requests_cpu` | Yes | The amount of CPU required by the container. |
| `resource<ID>_limits_memory` | Yes | The maximum amount of memory that can be consumed by the container. |
| `resource<ID>_limits_cpu` | Yes  | The maximum amount of CPU that can be consumed by the container. |

The `ID` portion of `resource` prefix for each variable name above should be an integer that is used to group the various `resource` variables into a single resource configuration.  For instance, the following shows a single resource configuration called `small`:

```ini
resource1_name='small'
resource1_requests_memory='512Mi'
resource1_requests_cpu=0.1
resource1_limits_memory='512Mi'
resource1_limits_cpu=0.1
```

As this example resource configuration shows, integer `1` is used as the ID for each of the `resource` variables, which together form a single resource configuration called `small`.  This approach allows different resource configurations to be created by defining the proper `resource` variables with a unique ID for each required resource configuration.

Additionally, once all resource configurations have been defined in the `inventory`, they can then be used to specify the default resource configurations that should be utilized for the various PG containers created by the operator.  This is done using the following variables, which are also defined in the  `inventory`:

```ini
default_container_resources='large'
default_load_resources='small'
default_rmdata_resources='small'
default_backup_resources='small'
default_pgbouncer_resources='small'
```

With the configuration shown above, the `large` resource configuration would be used by default for all database containers, while the `small` resource configuration would then be utilized by default for the various other containers created for a PG cluster.

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

* PostgreSQL Exporter - PostgreSQL metrics

The operator, however, does not install the necessary timeseries database (Prometheus) for storing the collected
metrics or the front end visualization (Grafana) of those metrics.

Included in these playbooks are roles for deploying Granfana and/or Prometheus.  See the `inventory` file
for options to install the metrics stack.

{{% notice tip %}}
At this time the Crunchy PostgreSQL Operator Playbooks only support storage classes.
{{% /notice %}}
