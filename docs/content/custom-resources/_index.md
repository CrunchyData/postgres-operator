---
title: "Using Custom Resources"
date:
draft: false
weight: 55
---

![Operator Architecture with CRDs](/Operator-Architecture-wCRDs.png)

As discussed in the [architecture overview]({{< relref "/architecture/overview.md" >}}),
the heart of the [PostgreSQL Operator]({{< relref "_index.md" >}}), and any
[Kubernetes Operator]([PostgreSQL Operator]({{< relref "_index.md" >}})), is one
or more [Custom Resources Definitions](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/#customresourcedefinitions),
also known as "CRDs". CRDs provide extensions to the Kubernetes API, and, in the
case of the PostgreSQL Operator, allow you to perform actions such as:

- Creating a PostgreSQL Cluster
- Updating PostgreSQL Cluster resource allocations
- Add additional utilities to a PostgreSQL cluster, e.g. [pgBouncer]({{< relref "/pgo-client/reference/pgo_create_pgbouncer.md" >}})
for connection pooling and more.

The PostgreSQL Operator provides the [`pgo` client]({{< relref "/pgo-client/_index.md" >}})
as a convenience for interfacing with the CRDs, as manipulating the CRDs
directly can be a tedious process. For example, there are several Kubernetes
objects that need to be set up prior to creating a `pgcluster` [custom resource](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)
in order to successfully deploy a new PostgreSQL cluster.

The Kubernetes community trend has been to move towards supporting a
"custom resource only" workflow for using Operators, and this is something that
the PostgreSQL Operator aims to do as well. Certain workflows are fully driven
by Custom Resources (e.g. creating a PostgreSQL cluster), while others still
need to interface through the [`pgo` client]({{< relref "/pgo-client/_index.md" >}})
(e.g. adding a PostgreSQL user).

The following sections will describe the functionality that is available today
when manipulating the PostgreSQL Operator Custom Resources directly.

## PostgreSQL Operator Custom Resource Definitions

There are several PostgreSQL Operator Custom Resource Definitions (CRDs) that
are installed in order for the PostgreSQL Operator to successfully function:

- `pgclusters.crunchydata.com`: Stores information required to manage a
PostgreSQL cluster. This includes things like the cluster name, what storage and
resource classes to use, which version of PostgreSQL to run, information about
how to maintain a high-availability cluster, etc.
- `pgreplicas.crunchydata.com`: Stores information required to manage the
replicas within a PostgreSQL cluster. This includes things like the number of
replicas, what storage and resource classes to use, special affinity rules, etc.
- `pgtasks.crunchydata.com`: A general purpose CRD that accepts a type of task
that is needed to run against a cluster (e.g. take a backup) and tracks the
state of said task through its workflow.
- `pgpolicies.crunchydata.com`: Stores a reference to a SQL file that can be
executed against a PostgreSQL cluster. In the past, this was used to manage RLS
policies on PostgreSQL clusters.

Below takes an in depth look for what each attribute does in a Custom Resource
Definition, and how they can be used in the creation and update workflow.

### Glossary

- `create`: if an attribute is listed as `create`, it means it can affect what
happens when a new Custom Resource is created.
- `update`: if an attribute is listed as `update`, it means it can affect the
Custom Resource, and by extension the objects it manages, when the attribute is
updated.

### `pgclusters.crunchydata.com`

The `pgclusters.crunchydata.com` Custom Resource Definition is the fundamental
definition of a PostgreSQL cluster. Most attributes only affect the deployment
of a PostgreSQL cluster at the time the PostgreSQL cluster is created. Some
attributes can be modified during the lifetime of the PostgreSQL cluster and
make changes, as described below.

#### Specification (`Spec`)

| Attribute | Action | Description |
|-----------|--------|-------------|
| BackrestLimits | `create`, `update` | Specify the container resource limits that the pgBackRest repository should use. Follows the [Kubernetes definitions of resource limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-requests-and-limits-of-pod-and-container). |
| BackrestResources | `create`, `update` | Specify the container resource requests that the pgBackRest repository should use. Follows the [Kubernetes definitions of resource requests](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-requests-and-limits-of-pod-and-container). |
| BackrestS3Bucket | `create` | An optional parameter that specifies a S3 bucket that pgBackRest should use. |
| BackrestS3Endpoint | `create` | An optional parameter that specifies the S3 endpoint pgBackRest should use. |
| BackrestS3Region | `create` | An optional parameter that specifies a cloud region that pgBackRest should use. |
| BackrestS3URIStyle | `create` | An optional parameter that specifies if pgBackRest should use the `path` or `host` S3 URI style. |
| BackrestS3VerifyTLS | `create` | An optional parameter that specifies if pgBackRest should verify the TLS endpoint. |
| BackrestStorage | `create` | A specification that gives information about the storage attributes for the pgBackRest repository, which stores backups and archives, of the PostgreSQL cluster. For details, please see the `Storage Specification` section below. This is required. |
| CCPImage | `create` | The name of the PostgreSQL container image to use, e.g. `crunchy-postgres-ha` or `crunchy-postgres-ha-gis`. |
| CCPImagePrefix | `create` | If provided, the image prefix (or registry) of the PostgreSQL container image, e.g. `registry.developers.crunchydata.com/crunchydata`. The default is to use the image prefix set in the PostgreSQL Operator configuration. |
| CCPImageTag | `create` | The tag of the PostgreSQL container image to use, e.g. `{{< param centosBase >}}-{{< param postgresVersion >}}-{{< param operatorVersion >}}`. |
| CollectSecretName | `create` | An optional attribute unless `crunchy_collect` is specified in the `UserLabels`; contains the name of a Kubernetes Secret that contains the credentials for a PostgreSQL user that is used for metrics collection, and is created when the PostgreSQL cluster is first bootstrapped. For more information, please see `User Secret Specification`.|
| ClusterName | `create` | The name of the PostgreSQL cluster, e.g. `hippo`. This is used to group PostgreSQL instances (primary, replicas) together. |
| CustomConfig | `create` | If specified, references a custom ConfigMap to use when bootstrapping a PostgreSQL cluster. For the shape of this file, please see the section on [Custom Configuration]({{< relref "/advanced/custom-configuration.md" >}}) |
| Database | `create` | The name of a database that the PostgreSQL user can log into after the PostgreSQL cluster is created. |
| ExporterPort | `create` | If the `"crunchy_collect"` label is set in `UserLabels`, then this specifies the port that the metrics sidecar runs on (e.g. `9187`) |
| Limits | `create`, `update` | Specify the container resource limits that the PostgreSQL cluster should use. Follows the [Kubernetes definitions of resource limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-requests-and-limits-of-pod-and-container). |
| Name | `create` | The name of the PostgreSQL instance that is the primary. On creation, this should be set to be the same as `ClusterName`. |
| Namespace | `create` | The Kubernetes Namespace that the PostgreSQL cluster is deployed in. |
| PGBadgerPort | `create` | If the `"crunchy-pgbadger"` label is set in `UserLabels`, then this specifies the port that the pgBadger sidecar runs on (e.g. `10000`) |
| PGDataSource | `create` | Used to indicate if a PostgreSQL cluster should bootstrap its data from a pgBackRest repository. This uses the PostgreSQL Data Source Specification, described below. |
| PGOImagePrefix | `create` | If provided, the image prefix (or registry) of any PostgreSQL Operator images that are used for jobs, e.g. `registry.developers.crunchydata.com/crunchydata`. The default is to use the image prefix set in the PostgreSQL Operator configuration. |
| PgBouncer | `create`, `update` | If specified, defines the attributes to use for the pgBouncer connection pooling deployment that can be used in conjunction with this PostgreSQL cluster. Please see the specification defined below. |
| PodAntiAffinity | `create` | A required section. Sets the [pod anti-affinity rules]({{< relref "/architecture/high-availability/_index.md#how-the-crunchy-postgresql-operator-uses-pod-anti-affinity" >}}) for the PostgreSQL cluster and associated deployments. Please see the `Pod Anti-Affinity Specification` section below. |
| Policies | `create` | If provided, a comma-separated list referring to `pgpolicies.crunchydata.com.Spec.Name` that should be run once the PostgreSQL primary is first initialized. |
| Port | `create` | The port that PostgreSQL will run on, e.g. `5432`. |
| PrimaryStorage | `create` | A specification that gives information about the storage attributes for the primary instance in the PostgreSQL cluster. For details, please see the `Storage Specification` section below. This is required. |
| RootSecretName | `create` | The name of a Kubernetes Secret that contains the credentials for a PostgreSQL _replication user_ that is created when the PostgreSQL cluster is first bootstrapped. For more information, please see `User Secret Specification`.|
| ReplicaStorage | `create` | A specification that gives information about the storage attributes for any replicas in the PostgreSQL cluster. For details, please see the `Storage Specification` section below. This will likely be changed in the future based on the nature of the high-availability system, but presently it is still required that you set it. It is recommended you use similar settings to that of `PrimaryStorage`. |
| Replicas | `create` | The number of replicas to create after a PostgreSQL primary is first initialized. This only works on create; to scale a cluster after it is initialized, please use the [`pgo scale`]({{< relref "/pgo-client/reference/pgo_scale.md" >}}) command. |
| Resources | `create`, `update` | Specify the container resource requests that the PostgreSQL cluster should use. Follows the [Kubernetes definitions of resource requests](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-requests-and-limits-of-pod-and-container). |
| RootSecretName | `create` | The name of a Kubernetes Secret that contains the credentials for a PostgreSQL superuser that is created when the PostgreSQL cluster is first bootstrapped. For more information, please see `User Secret Specification`.|
| SyncReplication | `create` | If set to `true`, specifies the PostgreSQL cluster to use [synchronous replication]({{< relref "/architecture/high-availability/_index.md#how-the-crunchy-postgresql-operator-uses-pod-anti-affinity#synchronous-replication-guarding-against-transactions-loss" >}}).|
| User | `create` | The name of the PostgreSQL user that is created when the PostgreSQL cluster is first created. |
| UserLabels | `create` | A set of key-value string pairs that are used as a sort of "catch-all" for things that really should be modeled in the CRD. These values do get copied to the actually CR labels. If you want to set up metrics collection or pgBadger, you would specify `"crunchy_collect": "true"` and `"crunchy-pgbadger": "true"` here, respectively. However, this structure does need to be set, so just follow whatever is in the example. |
| UserSecretName | `create` | The name of a Kubernetes Secret that contains the credentials for a standard PostgreSQL user that is created when the PostgreSQL cluster is first bootstrapped. For more information, please see `User Secret Specification`.|
| TablespaceMounts | `create`,`update` | Lists any tablespaces that are attached to the PostgreSQL cluster. Tablespaces can be added at a later time by updating the `TablespaceMounts` entry, but they cannot be removed. Stores a map of information, with the key being the name of the tablespace, and the value being a Storage Specification, defined below. |
| TLS | `create` | Defines the attributes for enabling TLS for a PostgreSQL cluster. See TLS Specification below. |
| TLSOnly | `create` | If set to true, requires client connections to use only TLS to connect to the PostgreSQL database. |
| Standby | `create`, `update` | If set to true, indicates that the PostgreSQL cluster is a "standby" cluster, i.e. is in read-only mode entirely. Please see [Kubernetes Multi-Cluster Deployments]({{< relref "/architecture/high-availability/multi-cluster-kubernetes.md" >}}) for more information. |
| Shutdown | `create`, `update` | If set to true, indicates that a PostgreSQL cluster should shutdown. If set to false, indicates that a PostgreSQL cluster should be up and running. |

##### Storage Specification

The storage specification is a spec that defines attributes about the storage to
be used for a particular function of a PostgreSQL cluster (e.g. a primary
instance or for the pgBackRest backup repository). The below describes each
attribute and how it works.

| Attribute | Action | Description |
|-----------|--------|-------------|
| AccessMode| `create` | The name of the Kubernetes Persistent Volume [Access Mode](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#access-modes) to use. |
| MatchLabels | `create` | Only used with `StorageType` of `create`, used to match a particular subset of provisioned Persistent Volumes. |
| Name | `create` | Only needed for `PrimaryStorage` in `pgclusters.crunchydata.com`.Used to identify the name of the PostgreSQL cluster. Should match `ClusterName`. |
| Size | `create` | The size of the [Persistent Volume Claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#persistentvolumeclaims) (PVC). Must use a Kubernetes resource value, e.g. `20Gi`. |
| StorageClass | `create` | The name of the Kubernetes [StorageClass](https://kubernetes.io/docs/concepts/storage/storage-classes/) to use. |
| StorageType | `create` | Set to `create` if storage is provisioned (e.g. using `hostpath`). Set to `dynamic` if using a dynamic storage provisioner, e.g. via a `StorageClass`. |
| SupplementalGroups | `create` | If provided, a comma-separated list of group IDs to use in case it is needed to interface with a particular storage system. Typically used with NFS or hostpath storage. |

##### Pod Anti-Affinity Specification

Sets the [pod anti-affinity]({{< relref "/architecture/high-availability/_index.md#how-the-crunchy-postgresql-operator-uses-pod-anti-affinity" >}})
for the PostgreSQL cluster and associated deployments. Each attribute can
contain one of the following values:

- `required`
- `preferred` (which is also the recommended default)
- `disabled`

For a detailed explanation for how this works. Please see the [high-availability]({{< relref "/architecture/high-availability/_index.md#how-the-crunchy-postgresql-operator-uses-pod-anti-affinity" >}})
documentation.

| Attribute | Action | Description |
|-----------|--------|-------------|
| Default | `create` | The default pod anti-affinity to use for all Pods managed in a given PostgreSQL cluster. |
| PgBackRest | `create` | If set to a value that differs from `Default`, specifies the pod anti-affinity to use for just the pgBackRest repository. |
| PgBouncer | `create` | If set to a value that differs from `Default`, specifies the pod anti-affinity to use for just the pgBouncer Pods. |

##### PostgreSQL Data Source Specification

This specification is used when one wants to bootstrap the data in a PostgreSQL
cluster from a pgBackRest repository. This can be a pgBackRest repository that
is attached to an active PostgreSQL cluster or is kept around to be used for
spawning new PostgreSQL clusters.

| Attribute | Action | Description |
|-----------|--------|-------------|
| RestoreFrom | `create` | The name of a PostgreSQL cluster, active or former, that will be used for bootstrapping the data of a new PostgreSQL cluster. |
| RestoreOpts | `create` | Additional pgBackRest [restore options](https://pgbackrest.org/command.html#command-restore) that can be used as part of the bootstrapping operation, for example, point-in-time-recovery options. |

##### TLS Specification

The TLS specification makes a reference to the various secrets that are required
to enable TLS in a PostgreSQL cluster. For more information on how these secrets
should be structured, please see [Enabling TLS in a PostgreSQL Cluster]({{< relref "/pgo-client/common-tasks.md#enable-tls" >}}).

| Attribute | Action | Description |
|-----------|--------|-------------|
| CASecret | `create` | A reference to the name of a Kubernetes Secret that specifies a certificate authority for the PostgreSQL cluster to trust. |
| ReplicationTLSSecret | `create` | A reference to the name of a Kubernetes TLS Secret that contains a keypair for authenticating the replication user. Must be used with `CASecret` and `TLSSecret`. |
| TLSSecret | `create` | A reference to the name of a Kubernetes TLS Secret that contains a keypair that is used for the PostgreSQL instance to identify itself and perform TLS communications with PostgreSQL clients. Must be used with `CASecret`. |

##### pgBouncer Specification

The pgBouncer specification defines how a pgBouncer deployment can be deployed
alongside the PostgreSQL cluster. pgBouncer is a PostgreSQL connection pooler
that can also help manage connection state, and is helpful to deploy alongside
a PostgreSQL cluster to help with failover scenarios too.

| Attribute | Action | Description |
|-----------|--------|-------------|
| Limits | `create`, `update` | Specify the container resource limits that the pgBouncer Pods should use. Follows the [Kubernetes definitions of resource limits](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-requests-and-limits-of-pod-and-container). |
| Replicas | `create`, `update` | The number of pgBouncer instances to deploy. Must be set to at least `1` to deploy pgBouncer. Setting to `0` removes an existing pgBouncer deployment for the PostgreSQL cluster. |
| Resources | `create`, `update` | Specify the container resource requests that the pgBouncer Pods should use. Follows the [Kubernetes definitions of resource requests](https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/#resource-requests-and-limits-of-pod-and-container). |

### `pgreplicas.crunchydata.com`

The `pgreplicas.crunchydata.com` Custom Resource Definition contains information
pertaning to the structure of PostgreSQL replicas associated within a PostgreSQL
cluster. All of the attributes only affect the replica when it is created.

#### Specification (`Spec`)

| Attribute | Action | Description |
|-----------|--------|-------------|
| ClusterName | `create` | The name of the PostgreSQL cluster, e.g. `hippo`. This is used to group PostgreSQL instances (primary, replicas) together. |
| Name | `create` | The name of this PostgreSQL replica. It should be unique within a `ClusterName`. |
| Namespace | `create` | The Kubernetes Namespace that the PostgreSQL cluster is deployed in. |
| ReplicaStorage | `create` | A specification that gives information about the storage attributes for any replicas in the PostgreSQL cluster. For details, please see the `Storage Specification` section in the `pgclusters.crunchydata.com` description. This will likely be changed in the future based on the nature of the high-availability system, but presently it is still required that you set it. It is recommended you use similar settings to that of `PrimaryStorage`. |
| UserLabels | `create` | A set of key-value string pairs that are used as a sort of "catch-all" for things that really should be modeled in the CRD. These values do get copied to the actually CR labels. If you want to set up metrics collection, you would specify `"crunchy_collect": "true"` here. This also allows for node selector pinning using `NodeLabelKey` and `NodeLabelValue`. However, this structure does need to be set, so just follow whatever is in the example. |

## Custom Resource Workflows

### Create a PostgreSQL Cluster

The fundamental workflow for interfacing with a PostgreSQL Operator Custom
Resource Definition is for creating a PostgreSQL cluster. However, this is also
one of the most complicated workflows to go through, as there are several
Kubernetes objects that need to be created prior to using this method. These
include:

- Secrets
  - Information for setting up a pgBackRest repository
  - PostgreSQL superuser bootstrap credentials
  - PostgreSQL replication user bootstrap credentials
  - PostgresQL standard user bootstrap credentials

Additionally, if you want to add some of the other sidecars, you may need to
create additional secrets.

The following guide goes through how to create a PostgreSQL cluster called
`hippo` by creating a new custom resource.

#### Step 1: Create the pgBackRest Secret

pgBackRest is a fundamental part of a PostgreSQL deployment with the PostgreSQL
Operator: not only is it a backup and archive repository, but it also helps with
operations such as self-healing. A PostgreSQL instance a pgBackRest communicate
using ssh, and as such, we need to generate a unique ssh keypair for
communication for each PostgreSQL cluster we deploy.

In this example, we generate a ssh keypair using ED25519 keys, but if your
environment requires it, you can also use RSA keys.

In your working directory, run the following commands:

<pre style="overflow-x: auto; white-space: pre-wrap; white-space: -moz-pre-wrap; white-space: -pre-wrap; white-space: -o-pre-wrap; word-wrap: break-word;">
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo

# generate a SSH public/private keypair for use by pgBackRest
ssh-keygen -t ed25519 -N '' -f "${pgo_cluster_name}-key"

# base64 encoded the keys for the generation of the Kubernetes secret, and place
# them into variables temporarily
public_key_temp=$(cat "${pgo_cluster_name}-key.pub" | base64)
private_key_temp=$(cat "${pgo_cluster_name}-key" | base64)
export pgbackrest_public_key="${public_key_temp//[$'\n']}" pgbackrest_private_key="${private_key_temp//[$'\n']}"

# create the backrest-repo-config example file and substitute in the newly
# created keys
#
# (Note: that the "config" / "sshd_config" entries contain configuration to
# ensure that PostgreSQL instances are able to communicate with the pgBackRest
# repository, which houses backups and archives, and vice versa. Most of the
# settings follow the sshd defaults, with a few overrides. Edit at your own
# discretion.)
cat <<-EOF > "${pgo_cluster_name}-backrest-repo-config.yaml"
apiVersion: v1
kind: Secret
type: Opaque
metadata:
  labels:
    pg-cluster: ${pgo_cluster_name}
    pgo-backrest-repo: "true"
  name: ${pgo_cluster_name}-backrest-repo-config
  namespace: ${cluster_namespace}
data:
  authorized_keys: ${pgbackrest_public_key}
  id_ed25519: ${pgbackrest_private_key}
  ssh_host_ed25519_key: ${pgbackrest_private_key}
  config: SG9zdCAqClN0cmljdEhvc3RLZXlDaGVja2luZyBubwpJZGVudGl0eUZpbGUgL3RtcC9pZF9lZDI1NTE5ClBvcnQgMjAyMgpVc2VyIHBnYmFja3Jlc3QK
  sshd_config: IwkkT3BlbkJTRDogc3NoZF9jb25maWcsdiAxLjEwMCAyMDE2LzA4LzE1IDEyOjMyOjA0IG5hZGR5IEV4cCAkCgojIFRoaXMgaXMgdGhlIHNzaGQgc2VydmVyIHN5c3RlbS13aWRlIGNvbmZpZ3VyYXRpb24gZmlsZS4gIFNlZQojIHNzaGRfY29uZmlnKDUpIGZvciBtb3JlIGluZm9ybWF0aW9uLgoKIyBUaGlzIHNzaGQgd2FzIGNvbXBpbGVkIHdpdGggUEFUSD0vdXNyL2xvY2FsL2JpbjovdXNyL2JpbgoKIyBUaGUgc3RyYXRlZ3kgdXNlZCBmb3Igb3B0aW9ucyBpbiB0aGUgZGVmYXVsdCBzc2hkX2NvbmZpZyBzaGlwcGVkIHdpdGgKIyBPcGVuU1NIIGlzIHRvIHNwZWNpZnkgb3B0aW9ucyB3aXRoIHRoZWlyIGRlZmF1bHQgdmFsdWUgd2hlcmUKIyBwb3NzaWJsZSwgYnV0IGxlYXZlIHRoZW0gY29tbWVudGVkLiAgVW5jb21tZW50ZWQgb3B0aW9ucyBvdmVycmlkZSB0aGUKIyBkZWZhdWx0IHZhbHVlLgoKIyBJZiB5b3Ugd2FudCB0byBjaGFuZ2UgdGhlIHBvcnQgb24gYSBTRUxpbnV4IHN5c3RlbSwgeW91IGhhdmUgdG8gdGVsbAojIFNFTGludXggYWJvdXQgdGhpcyBjaGFuZ2UuCiMgc2VtYW5hZ2UgcG9ydCAtYSAtdCBzc2hfcG9ydF90IC1wIHRjcCAjUE9SVE5VTUJFUgojClBvcnQgMjAyMgojQWRkcmVzc0ZhbWlseSBhbnkKI0xpc3RlbkFkZHJlc3MgMC4wLjAuMAojTGlzdGVuQWRkcmVzcyA6OgoKSG9zdEtleSAvc3NoZC9zc2hfaG9zdF9lZDI1NTE5X2tleQoKIyBDaXBoZXJzIGFuZCBrZXlpbmcKI1Jla2V5TGltaXQgZGVmYXVsdCBub25lCgojIExvZ2dpbmcKI1N5c2xvZ0ZhY2lsaXR5IEFVVEgKU3lzbG9nRmFjaWxpdHkgQVVUSFBSSVYKI0xvZ0xldmVsIElORk8KCiMgQXV0aGVudGljYXRpb246CgojTG9naW5HcmFjZVRpbWUgMm0KUGVybWl0Um9vdExvZ2luIG5vClN0cmljdE1vZGVzIG5vCiNNYXhBdXRoVHJpZXMgNgojTWF4U2Vzc2lvbnMgMTAKClB1YmtleUF1dGhlbnRpY2F0aW9uIHllcwoKIyBUaGUgZGVmYXVsdCBpcyB0byBjaGVjayBib3RoIC5zc2gvYXV0aG9yaXplZF9rZXlzIGFuZCAuc3NoL2F1dGhvcml6ZWRfa2V5czIKIyBidXQgdGhpcyBpcyBvdmVycmlkZGVuIHNvIGluc3RhbGxhdGlvbnMgd2lsbCBvbmx5IGNoZWNrIC5zc2gvYXV0aG9yaXplZF9rZXlzCkF1dGhvcml6ZWRLZXlzRmlsZQkvc3NoZC9hdXRob3JpemVkX2tleXMKCiNBdXRob3JpemVkUHJpbmNpcGFsc0ZpbGUgbm9uZQoKI0F1dGhvcml6ZWRLZXlzQ29tbWFuZCBub25lCiNBdXRob3JpemVkS2V5c0NvbW1hbmRVc2VyIG5vYm9keQoKIyBGb3IgdGhpcyB0byB3b3JrIHlvdSB3aWxsIGFsc28gbmVlZCBob3N0IGtleXMgaW4gL2V0Yy9zc2gvc3NoX2tub3duX2hvc3RzCiNIb3N0YmFzZWRBdXRoZW50aWNhdGlvbiBubwojIENoYW5nZSB0byB5ZXMgaWYgeW91IGRvbid0IHRydXN0IH4vLnNzaC9rbm93bl9ob3N0cyBmb3IKIyBIb3N0YmFzZWRBdXRoZW50aWNhdGlvbgojSWdub3JlVXNlcktub3duSG9zdHMgbm8KIyBEb24ndCByZWFkIHRoZSB1c2VyJ3Mgfi8ucmhvc3RzIGFuZCB+Ly5zaG9zdHMgZmlsZXMKI0lnbm9yZVJob3N0cyB5ZXMKCiMgVG8gZGlzYWJsZSB0dW5uZWxlZCBjbGVhciB0ZXh0IHBhc3N3b3JkcywgY2hhbmdlIHRvIG5vIGhlcmUhCiNQYXNzd29yZEF1dGhlbnRpY2F0aW9uIHllcwojUGVybWl0RW1wdHlQYXNzd29yZHMgbm8KUGFzc3dvcmRBdXRoZW50aWNhdGlvbiBubwoKIyBDaGFuZ2UgdG8gbm8gdG8gZGlzYWJsZSBzL2tleSBwYXNzd29yZHMKQ2hhbGxlbmdlUmVzcG9uc2VBdXRoZW50aWNhdGlvbiB5ZXMKI0NoYWxsZW5nZVJlc3BvbnNlQXV0aGVudGljYXRpb24gbm8KCiMgS2VyYmVyb3Mgb3B0aW9ucwojS2VyYmVyb3NBdXRoZW50aWNhdGlvbiBubwojS2VyYmVyb3NPckxvY2FsUGFzc3dkIHllcwojS2VyYmVyb3NUaWNrZXRDbGVhbnVwIHllcwojS2VyYmVyb3NHZXRBRlNUb2tlbiBubwojS2VyYmVyb3NVc2VLdXNlcm9rIHllcwoKIyBHU1NBUEkgb3B0aW9ucwojR1NTQVBJQXV0aGVudGljYXRpb24geWVzCiNHU1NBUElDbGVhbnVwQ3JlZGVudGlhbHMgbm8KI0dTU0FQSVN0cmljdEFjY2VwdG9yQ2hlY2sgeWVzCiNHU1NBUElLZXlFeGNoYW5nZSBubwojR1NTQVBJRW5hYmxlazV1c2VycyBubwoKIyBTZXQgdGhpcyB0byAneWVzJyB0byBlbmFibGUgUEFNIGF1dGhlbnRpY2F0aW9uLCBhY2NvdW50IHByb2Nlc3NpbmcsCiMgYW5kIHNlc3Npb24gcHJvY2Vzc2luZy4gSWYgdGhpcyBpcyBlbmFibGVkLCBQQU0gYXV0aGVudGljYXRpb24gd2lsbAojIGJlIGFsbG93ZWQgdGhyb3VnaCB0aGUgQ2hhbGxlbmdlUmVzcG9uc2VBdXRoZW50aWNhdGlvbiBhbmQKIyBQYXNzd29yZEF1dGhlbnRpY2F0aW9uLiAgRGVwZW5kaW5nIG9uIHlvdXIgUEFNIGNvbmZpZ3VyYXRpb24sCiMgUEFNIGF1dGhlbnRpY2F0aW9uIHZpYSBDaGFsbGVuZ2VSZXNwb25zZUF1dGhlbnRpY2F0aW9uIG1heSBieXBhc3MKIyB0aGUgc2V0dGluZyBvZiAiUGVybWl0Um9vdExvZ2luIHdpdGhvdXQtcGFzc3dvcmQiLgojIElmIHlvdSBqdXN0IHdhbnQgdGhlIFBBTSBhY2NvdW50IGFuZCBzZXNzaW9uIGNoZWNrcyB0byBydW4gd2l0aG91dAojIFBBTSBhdXRoZW50aWNhdGlvbiwgdGhlbiBlbmFibGUgdGhpcyBidXQgc2V0IFBhc3N3b3JkQXV0aGVudGljYXRpb24KIyBhbmQgQ2hhbGxlbmdlUmVzcG9uc2VBdXRoZW50aWNhdGlvbiB0byAnbm8nLgojIFdBUk5JTkc6ICdVc2VQQU0gbm8nIGlzIG5vdCBzdXBwb3J0ZWQgaW4gUmVkIEhhdCBFbnRlcnByaXNlIExpbnV4IGFuZCBtYXkgY2F1c2Ugc2V2ZXJhbAojIHByb2JsZW1zLgpVc2VQQU0geWVzIAoKI0FsbG93QWdlbnRGb3J3YXJkaW5nIHllcwojQWxsb3dUY3BGb3J3YXJkaW5nIHllcwojR2F0ZXdheVBvcnRzIG5vClgxMUZvcndhcmRpbmcgeWVzCiNYMTFEaXNwbGF5T2Zmc2V0IDEwCiNYMTFVc2VMb2NhbGhvc3QgeWVzCiNQZXJtaXRUVFkgeWVzCiNQcmludE1vdGQgeWVzCiNQcmludExhc3RMb2cgeWVzCiNUQ1BLZWVwQWxpdmUgeWVzCiNVc2VMb2dpbiBubwpVc2VQcml2aWxlZ2VTZXBhcmF0aW9uIG5vCiNQZXJtaXRVc2VyRW52aXJvbm1lbnQgbm8KI0NvbXByZXNzaW9uIGRlbGF5ZWQKI0NsaWVudEFsaXZlSW50ZXJ2YWwgMAojQ2xpZW50QWxpdmVDb3VudE1heCAzCiNTaG93UGF0Y2hMZXZlbCBubwojVXNlRE5TIHllcwojUGlkRmlsZSAvdmFyL3J1bi9zc2hkLnBpZAojTWF4U3RhcnR1cHMgMTA6MzA6MTAwCiNQZXJtaXRUdW5uZWwgbm8KI0Nocm9vdERpcmVjdG9yeSBub25lCiNWZXJzaW9uQWRkZW5kdW0gbm9uZQoKIyBubyBkZWZhdWx0IGJhbm5lciBwYXRoCiNCYW5uZXIgbm9uZQoKIyBBY2NlcHQgbG9jYWxlLXJlbGF0ZWQgZW52aXJvbm1lbnQgdmFyaWFibGVzCkFjY2VwdEVudiBMQU5HIExDX0NUWVBFIExDX05VTUVSSUMgTENfVElNRSBMQ19DT0xMQVRFIExDX01PTkVUQVJZIExDX01FU1NBR0VTCkFjY2VwdEVudiBMQ19QQVBFUiBMQ19OQU1FIExDX0FERFJFU1MgTENfVEVMRVBIT05FIExDX01FQVNVUkVNRU5UCkFjY2VwdEVudiBMQ19JREVOVElGSUNBVElPTiBMQ19BTEwgTEFOR1VBR0UKQWNjZXB0RW52IFhNT0RJRklFUlMKCiMgb3ZlcnJpZGUgZGVmYXVsdCBvZiBubyBzdWJzeXN0ZW1zClN1YnN5c3RlbQlzZnRwCS91c3IvbGliZXhlYy9vcGVuc3NoL3NmdHAtc2VydmVyCgojIEV4YW1wbGUgb2Ygb3ZlcnJpZGluZyBzZXR0aW5ncyBvbiBhIHBlci11c2VyIGJhc2lzCiNNYXRjaCBVc2VyIGFub25jdnMKIwlYMTFGb3J3YXJkaW5nIG5vCiMJQWxsb3dUY3BGb3J3YXJkaW5nIG5vCiMJUGVybWl0VFRZIG5vCiMJRm9yY2VDb21tYW5kIGN2cyBzZXJ2ZXI=
EOF

# remove the pgBackRest ssh keypair from the shell session
unset pgbackrest_public_key pgbackrest_private_key

# create the pgBackRest secret
kubectl apply -f "${pgo_cluster_name}-backrest-repo-config.yaml"
</pre>

#### Step 2: Creating the PostgreSQL User Secrets

As mentioned above, there are a minimum of three PostgreSQL user accounts that
you must create in order to bootstrap a PostgreSQL cluster. These are:

- A PostgreSQL superuser
- A replication user
- A standard PostgreSQL user

The below code will help you set up these Secrets.

```
# this variable is the name of the cluster being created
pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
cluster_namespace=pgo

# this is the superuser secret
kubectl create secret generic -n "${cluster_namespace}" "${pgo_cluster_name}-postgres-secret" \
  --from-literal=username=postgres \
  --from-literal=password=Supersecurepassword*

# this is the replication user secret
kubectl create secret generic -n "${cluster_namespace}" "${pgo_cluster_name}-primaryuser-secret" \
  --from-literal=username=primaryuser \
  --from-literal=password=Anothersecurepassword*

# this is the standard user secret
kubectl create secret generic -n "${cluster_namespace}" "${pgo_cluster_name}-hippo-secret" \
  --from-literal=username=hippo \
  --from-literal=password=Moresecurepassword*


kubectl label secrets -n "${cluster_namespace}" "${pgo_cluster_name}-postgres-secret" "pg-cluster=${pgo_cluster_name}"
kubectl label secrets -n "${cluster_namespace}" "${pgo_cluster_name}-primaryuser-secret" "pg-cluster=${pgo_cluster_name}"
kubectl label secrets -n "${cluster_namespace}" "${pgo_cluster_name}-hippo-secret" "pg-cluster=${pgo_cluster_name}"
```

#### Step 3: Create the PostgreSQL Cluster

With the Secrets in place. It is now time to create the PostgreSQL cluster.

The below manifest references the Secrets created in the previous step to add a
custom resource to the `pgclusters.crunchydata.com` custom resource definition.

**NOTE**: You will need to modify the storage sections to match your storage
configuration.

```
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo

cat <<-EOF > "${pgo_cluster_name}-pgcluster.yaml"
apiVersion: crunchydata.com/v1
kind: Pgcluster
metadata:
  annotations:
    current-primary: ${pgo_cluster_name}
  labels:
    autofail: "true"
    crunchy-pgbadger: "false"
    crunchy-pgha-scope: ${pgo_cluster_name}
    crunchy_collect: "false"
    deployment-name: ${pgo_cluster_name}
    name: ${pgo_cluster_name}
    pg-cluster: ${pgo_cluster_name}
    pg-pod-anti-affinity: ""
    pgo-backrest: "true"
    pgo-version: {{< param operatorVersion >}}
    pgouser: admin
  name: ${pgo_cluster_name}
  namespace: ${cluster_namespace}
spec:
  BackrestStorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ""
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  PrimaryStorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ${pgo_cluster_name}
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  ReplicaStorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ""
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  backrestLimits: {}
  backrestRepoPath: ""
  backrestResources:
    memory: 48Mi
  backrestS3Bucket: ""
  backrestS3Endpoint: ""
  backrestS3Region: ""
  backrestS3URIStyle: ""
  backrestS3VerifyTLS: ""
  ccpimage: crunchy-postgres-ha
  ccpimageprefix: registry.developers.crunchydata.com/crunchydata
  ccpimagetag: {{< param centosBase >}}-{{< param postgresVersion >}}-{{< param operatorVersion >}}
  clustername: ${pgo_cluster_name}
  customconfig: ""
  database: ${pgo_cluster_name}
  exporterport: "9187"
  limits: {}
  name: ${pgo_cluster_name}
  namespace: ${cluster_namespace}
  pgBouncer:
    limits: {}
    replicas: 0
  pgDataSource:
    restoreFrom: ""
    restoreOpts: ""
  pgbadgerport: "10000"
  pgoimageprefix: registry.developers.crunchydata.com/crunchydata
  podAntiAffinity:
    default: preferred
    pgBackRest: preferred
    pgBouncer: preferred
  policies: ""
  port: "5432"
  primarysecretname: ${pgo_cluster_name}-primaryuser-secret
  replicas: "0"
  rootsecretname: ${pgo_cluster_name}-postgres-secret
  shutdown: false
  standby: false
  tablespaceMounts: {}
  tls:
    caSecret: ""
    replicationTLSSecret: ""
    tlsSecret: ""
  tlsOnly: false
  user: hippo
  userlabels:
    crunchy_collect: "false"
    pg-pod-anti-affinity: ""
    pgo-version: {{< param operatorVersion >}}
  usersecretname: ${pgo_cluster_name}-hippo-secret
EOF

kubectl apply -f "${pgo_cluster_name}-pgcluster.yaml"
```

### Modify a Cluster

There following modification operations are supported on the
`pgclusters.crunchydata.com` custom resource definition:

#### Modify Resource Requests & Limits

Modifying the `resources`, `limits`, `backrestResources`, `backRestLimits`,
`pgBouncer.resources`, or `pgbouncer.limits` will cause the PostgreSQL Operator
to apply the new values to the affected [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/).

For example, if we wanted to make a memory request of 512Mi for the `hippo`
cluster created in the previous example, we could do the following:

```
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo

kubectl edit pgclusters.crunchydata.com -n "${cluster_namespace}" "${pgo_cluster_name}"
```

This will open up your editor. Find the `resources` block, and have it read as
the following:

```
resources:
  memory: 256Mi
```

The PostgreSQL Operator will respond and modify the PostgreSQL instances to
request 256Mi of memory.

Be careful when editing these values for a variety of reasons, mainly that
modifying these values will cause the Pods to restart, which in turn will create
potential downtime events. It's best to modify the values for a deployment group
together and not mix and match, i.e.

- PostgreSQL instances: `resources`, `limits`
- pgBackRest: `backrestResources`, `backrestLimits`
- pgBouncer: `pgBouncer.resources`, `pgBouncer.limits`

### Scale

Once you have created a PostgreSQL cluster, you may want to add a replica to
create a high-availability environment. Replicas are added and removed using the
`pgreplicas.crunchydata.com` custom resource definition. Each replica must have
a unique name, e.g. `hippo-rpl1` could be one unique replica for a PostgreSQL
cluster.

Using the above example cluster, `hippo`, let's add a replica called
`hippo-rpl1` using the configuration below. Be sure to change the
`replicastorage` block to match the storage configuration for your environment:

```
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this helps to name the replica, in this case "rpl1"
export pgo_cluster_replica_suffix=rpl1
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo

cat <<-EOF > "${pgo_cluster_name}-${pgo_cluster_replica_suffix}-pgreplica.yaml"
apiVersion: crunchydata.com/v1
kind: Pgreplica
metadata:
  labels:
    name: ${pgo_cluster_name}-${pgo_cluster_replica_suffix}
    pg-cluster: ${pgo_cluster_name}
    pgouser: admin
  name: ${pgo_cluster_name}-${pgo_cluster_replica_suffix}
  namespace: ${cluster_namespace}
spec:
  clustername: ${pgo_cluster_name}
  name: ${pgo_cluster_name}-${pgo_cluster_replica_suffix}
  namespace: ${cluster_namespace}
  replicastorage:
    accessmode: ReadWriteMany
    matchLabels: ""
    name: ${pgo_cluster_name}-${pgo_cluster_replica_suffix}
    size: 1G
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
  userlabels:
    NodeLabelKey: ""
    NodeLabelValue: ""
    crunchy_collect: "false"
    pg-pod-anti-affinity: ""
    pgo-version: {{< param operatorVersion >}}
EOF

kubectl apply -f "${pgo_cluster_name}-${pgo_cluster_replica_suffix}-pgreplica.yaml"
```

Add this time, removing a replica must be handled through the [`pgo` client]({{< relref "/pgo-client/common-tasks.md#high-availability-scaling-up-down">}}).

### Add a Tablespace

Tablespaces can be added during the lifetime of a PostgreSQL cluster (tablespaces can be removed as well, but for a detailed explanation as to how, please see the [Tablespaces]({{< relref "/architecture/tablespaces.md">}}) section).

To add a tablespace, you need to add an entry to the `tablespaceMounts` section
of a custom entry, where the key is the name of the tablespace (unique to the
`pgclusters.crunchydata.com` custom resource entry) and the value is a storage
configuration as defined in the `pgclusters.crunchydata.com` section above.

For example, to add a tablespace named `lake` to our `hippo` cluster, we can
open up the editor with the following code:

```
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo

kubectl edit pgclusters.crunchydata.com -n "${cluster_namespace}" "${pgo_cluster_name}"
```

and add an entry to the `tablespaceMounts` block that looks similar to this,
with the addition of the correct storage configuration for your environment:

```
tablespaceMounts:
  lake:
    accessmode: ReadWriteMany
    matchLabels: ""
    size: 5Gi
    storageclass: ""
    storagetype: create
    supplementalgroups: ""
```

### pgBouncer

[pgBouncer](https://www.pgbouncer.org/) is a PostgreSQL connection pooler and
state manager that can be useful for high-availability setups as well as
managing overall performance of a PostgreSQL cluster. A pgBouncer deployment for
a PostgreSQL cluster can be fully managed from a `pgclusters.crunchydata.com`
custom resource.

For example, to add a pgBouncer deployment to our `hippo` cluster with two
instances and a memory limit of 36Mi, you can edit the custom resource:

```
# this variable is the name of the cluster being created
export pgo_cluster_name=hippo
# this variable is the namespace the cluster is being deployed into
export cluster_namespace=pgo

kubectl edit pgclusters.crunchydata.com -n "${cluster_namespace}" "${pgo_cluster_name}"
```

And modify the `pgBouncer` block to look like this:

```
pgBouncer:
  limits:
    memory: 36Mi
  replicas: 2
```

Likewise, to remove pgBouncer from a PostgreSQL cluster, you would set
`replicas` to `0`:

```
pgBouncer:
  replicas: 0
```

### Start / Stop a Cluster

A PostgreSQL cluster can be start and stopped by toggling the `shutdown`
parameter in a `pgclusters.crunchydata.com` custom resource. Setting `shutdown`
to `true` will stop a PostgreSQL cluster, whereas a value of `false` will make
a cluster available. This affects all of the associated instances of a
PostgreSQL cluster.
