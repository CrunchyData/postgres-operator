---
title: CRD Reference
draft: false
weight: 100
---

Packages:

- [postgres-operator.crunchydata.com/v1beta1](#postgres-operatorcrunchydatacomv1beta1)

<h1 id="postgres-operatorcrunchydatacomv1beta1">postgres-operator.crunchydata.com/v1beta1</h1>

Resource Types:

- [PostgresCluster](#postgrescluster)




<h2 id="postgrescluster">PostgresCluster</h2>






PostgresCluster is the Schema for the postgresclusters API

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
      <td><b>apiVersion</b></td>
      <td>string</td>
      <td>postgres-operator.crunchydata.com/v1beta1</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b>kind</b></td>
      <td>string</td>
      <td>PostgresCluster</td>
      <td>true</td>
      </tr>
      <tr>
      <td><b><a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.20/#objectmeta-v1-meta">metadata</a></b></td>
      <td>object</td>
      <td>Refer to the Kubernetes API documentation for the fields of the `metadata` field.</td>
      <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspec">spec</a></b></td>
        <td>object</td>
        <td>PostgresClusterSpec defines the desired state of PostgresCluster</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatus">status</a></b></td>
        <td>object</td>
        <td>PostgresClusterStatus defines the observed state of PostgresCluster</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspec">
  PostgresCluster.spec
  <sup><sup><a href="#postgrescluster">↩ Parent</a></sup></sup>
</h3>



PostgresClusterSpec defines the desired state of PostgresCluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackups">backups</a></b></td>
        <td>object</td>
        <td>PostgreSQL backup configuration</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindex">instances</a></b></td>
        <td>[]object</td>
        <td>Specifies one or more sets of PostgreSQL pods that replicate data for this cluster.</td>
        <td>true</td>
      </tr><tr>
        <td><b>postgresVersion</b></td>
        <td>integer</td>
        <td>The major version of PostgreSQL installed in the PostgreSQL image</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecconfig">config</a></b></td>
        <td>object</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspeccustomreplicationtlssecret">customReplicationTLSSecret</a></b></td>
        <td>object</td>
        <td>The secret containing the replication client certificates and keys for secure connections to the PostgreSQL server. It will need to contain the client TLS certificate, TLS key and the Certificate Authority certificate with the data keys set to tls.crt, tls.key and ca.crt, respectively. NOTE: If CustomReplicationClientTLSSecret is provided, CustomTLSSecret MUST be provided and the ca.crt provided must be the same.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspeccustomtlssecret">customTLSSecret</a></b></td>
        <td>object</td>
        <td>The secret containing the Certificates and Keys to encrypt PostgreSQL traffic will need to contain the server TLS certificate, TLS key and the Certificate Authority certificate with the data keys set to tls.crt, tls.key and ca.crt, respectively. It will then be mounted as a volume projection to the '/pgconf/tls' directory. For more information on Kubernetes secret projections, please see https://k8s.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths NOTE: If CustomTLSSecret is provided, CustomReplicationClientTLSSecret MUST be provided and the ca.crt provided must be the same.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>Specifies a data source for bootstrapping the PostgreSQL cluster.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatabaseinitsql">databaseInitSQL</a></b></td>
        <td>object</td>
        <td>DatabaseInitSQL defines a ConfigMap containing custom SQL that will be run after the cluster is initialized. This ConfigMap must be in the same namespace as the cluster.</td>
        <td>false</td>
      </tr><tr>
        <td><b>disableDefaultPodScheduling</b></td>
        <td>boolean</td>
        <td>Whether or not the PostgreSQL cluster should use the defined default scheduling constraints. If the field is unset or false, the default scheduling constraints will be used in addition to any custom constraints provided.</td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>The image name to use for PostgreSQL containers. When omitted, the value comes from an operator environment variable. For standard PostgreSQL images, the format is RELATED_IMAGE_POSTGRES_{postgresVersion}, e.g. RELATED_IMAGE_POSTGRES_13. For PostGIS enabled PostgreSQL images, the format is RELATED_IMAGE_POSTGRES_{postgresVersion}_GIS_{postGISVersion}, e.g. RELATED_IMAGE_POSTGRES_13_GIS_3.1.</td>
        <td>false</td>
      </tr><tr>
        <td><b>imagePullPolicy</b></td>
        <td>enum</td>
        <td>ImagePullPolicy is used to determine when Kubernetes will attempt to pull (download) container images. More info: https://kubernetes.io/docs/concepts/containers/images/#image-pull-policy</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecimagepullsecretsindex">imagePullSecrets</a></b></td>
        <td>[]object</td>
        <td>The image pull secrets used to pull from a private registry Changing this value causes all running pods to restart. https://k8s.io/docs/tasks/configure-pod-container/pull-image-private-registry/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmetadata">metadata</a></b></td>
        <td>object</td>
        <td>Metadata contains metadata for PostgresCluster resources</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoring">monitoring</a></b></td>
        <td>object</td>
        <td>The specification of monitoring tools that connect to PostgreSQL</td>
        <td>false</td>
      </tr><tr>
        <td><b>openshift</b></td>
        <td>boolean</td>
        <td>Whether or not the PostgreSQL cluster is being deployed to an OpenShift environment. If the field is unset, the operator will automatically detect the environment.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecpatroni">patroni</a></b></td>
        <td>object</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>The port on which PostgreSQL should listen.</td>
        <td>false</td>
      </tr><tr>
        <td><b>postGISVersion</b></td>
        <td>string</td>
        <td>The PostGIS extension version installed in the PostgreSQL image. When image is not set, indicates a PostGIS enabled image will be used.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxy">proxy</a></b></td>
        <td>object</td>
        <td>The specification of a proxy that connects to PostgreSQL.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecservice">service</a></b></td>
        <td>object</td>
        <td>Specification of the service that exposes the PostgreSQL primary instance.</td>
        <td>false</td>
      </tr><tr>
        <td><b>shutdown</b></td>
        <td>boolean</td>
        <td>Whether or not the PostgreSQL cluster should be stopped. When this is true, workloads are scaled to zero and CronJobs are suspended. Other resources, such as Services and Volumes, remain in place.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecstandby">standby</a></b></td>
        <td>object</td>
        <td>Run this cluster as a read-only copy of an existing cluster or archive.</td>
        <td>false</td>
      </tr><tr>
        <td><b>supplementalGroups</b></td>
        <td>[]integer</td>
        <td>A list of group IDs applied to the process of a container. These can be useful when accessing shared file systems with constrained permissions. More info: https://kubernetes.io/docs/reference/kubernetes-api/workload-resources/pod-v1/#security-context</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterface">userInterface</a></b></td>
        <td>object</td>
        <td>The specification of a user interface that connects to PostgreSQL.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecusersindex">users</a></b></td>
        <td>[]object</td>
        <td>Users to create inside PostgreSQL and the databases they should access. The default creates one user that can access one database matching the PostgresCluster name. An empty list creates no users. Removing a user from this list does NOT drop the user nor revoke their access.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackups">
  PostgresCluster.spec.backups
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



PostgreSQL backup configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrest">pgbackrest</a></b></td>
        <td>object</td>
        <td>pgBackRest archive configuration</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrest">
  PostgresCluster.spec.backups.pgbackrest
  <sup><sup><a href="#postgresclusterspecbackups">↩ Parent</a></sup></sup>
</h3>



pgBackRest archive configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindex">repos</a></b></td>
        <td>[]object</td>
        <td>Defines a pgBackRest repository</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindex">configuration</a></b></td>
        <td>[]object</td>
        <td>Projected volumes containing custom pgBackRest configuration.  These files are mounted under "/etc/pgbackrest/conf.d" alongside any pgBackRest configuration generated by the PostgreSQL Operator: https://pgbackrest.org/configuration.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>global</b></td>
        <td>map[string]string</td>
        <td>Global pgBackRest configuration settings.  These settings are included in the "global" section of the pgBackRest configuration generated by the PostgreSQL Operator, and then mounted under "/etc/pgbackrest/conf.d": https://pgbackrest.org/configuration.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>The image name to use for pgBackRest containers.  Utilized to run pgBackRest repository hosts and backups. The image may also be set using the RELATED_IMAGE_PGBACKREST environment variable</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestjobs">jobs</a></b></td>
        <td>object</td>
        <td>Jobs field allows configuration for all backup jobs</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestmanual">manual</a></b></td>
        <td>object</td>
        <td>Defines details for manual pgBackRest backup Jobs</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestmetadata">metadata</a></b></td>
        <td>object</td>
        <td>Metadata contains metadata for PostgresCluster resources</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohost">repoHost</a></b></td>
        <td>object</td>
        <td>Defines configuration for a pgBackRest dedicated repository host.  This section is only applicable if at least one "volume" (i.e. PVC-based) repository is defined in the "repos" section, therefore enabling a dedicated repository host Deployment.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestore">restore</a></b></td>
        <td>object</td>
        <td>Defines details for performing an in-place restore using pgBackRest</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestsidecars">sidecars</a></b></td>
        <td>object</td>
        <td>Configuration for pgBackRest sidecar containers</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindex">
  PostgresCluster.spec.backups.pgbackrest.repos[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



PGBackRestRepo represents a pgBackRest repository.  Only one of its members may be specified.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>The name of the the repository</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexazure">azure</a></b></td>
        <td>object</td>
        <td>Represents a pgBackRest repository that is created using Azure storage</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexgcs">gcs</a></b></td>
        <td>object</td>
        <td>Represents a pgBackRest repository that is created using Google Cloud Storage</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexs3">s3</a></b></td>
        <td>object</td>
        <td>RepoS3 represents a pgBackRest repository that is created using AWS S3 (or S3-compatible) storage</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexschedules">schedules</a></b></td>
        <td>object</td>
        <td>Defines the schedules for the pgBackRest backups Full, Differential and Incremental backup types are supported: https://pgbackrest.org/user-guide.html#concept/backup</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexvolume">volume</a></b></td>
        <td>object</td>
        <td>Represents a pgBackRest repository that is created using a PersistentVolumeClaim</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexazure">
  PostgresCluster.spec.backups.pgbackrest.repos[index].azure
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindex">↩ Parent</a></sup></sup>
</h3>



Represents a pgBackRest repository that is created using Azure storage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>container</b></td>
        <td>string</td>
        <td>The Azure container utilized for the repository</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexgcs">
  PostgresCluster.spec.backups.pgbackrest.repos[index].gcs
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindex">↩ Parent</a></sup></sup>
</h3>



Represents a pgBackRest repository that is created using Google Cloud Storage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>bucket</b></td>
        <td>string</td>
        <td>The GCS bucket utilized for the repository</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexs3">
  PostgresCluster.spec.backups.pgbackrest.repos[index].s3
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindex">↩ Parent</a></sup></sup>
</h3>



RepoS3 represents a pgBackRest repository that is created using AWS S3 (or S3-compatible) storage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>bucket</b></td>
        <td>string</td>
        <td>The S3 bucket utilized for the repository</td>
        <td>true</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>A valid endpoint corresponding to the specified region</td>
        <td>true</td>
      </tr><tr>
        <td><b>region</b></td>
        <td>string</td>
        <td>The region corresponding to the S3 bucket</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexschedules">
  PostgresCluster.spec.backups.pgbackrest.repos[index].schedules
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindex">↩ Parent</a></sup></sup>
</h3>



Defines the schedules for the pgBackRest backups Full, Differential and Incremental backup types are supported: https://pgbackrest.org/user-guide.html#concept/backup

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>differential</b></td>
        <td>string</td>
        <td>Defines the Cron schedule for a differential pgBackRest backup. Follows the standard Cron schedule syntax: https://k8s.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax</td>
        <td>false</td>
      </tr><tr>
        <td><b>full</b></td>
        <td>string</td>
        <td>Defines the Cron schedule for a full pgBackRest backup. Follows the standard Cron schedule syntax: https://k8s.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax</td>
        <td>false</td>
      </tr><tr>
        <td><b>incremental</b></td>
        <td>string</td>
        <td>Defines the Cron schedule for an incremental pgBackRest backup. Follows the standard Cron schedule syntax: https://k8s.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexvolume">
  PostgresCluster.spec.backups.pgbackrest.repos[index].volume
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindex">↩ Parent</a></sup></sup>
</h3>



Represents a pgBackRest repository that is created using a PersistentVolumeClaim

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspec">volumeClaimSpec</a></b></td>
        <td>object</td>
        <td>Defines a PersistentVolumeClaim spec used to create and/or bind a volume</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspec">
  PostgresCluster.spec.backups.pgbackrest.repos[index].volume.volumeClaimSpec
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindexvolume">↩ Parent</a></sup></sup>
</h3>



Defines a PersistentVolumeClaim spec used to create and/or bind a volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecresources">resources</a></b></td>
        <td>object</td>
        <td>Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecselector">selector</a></b></td>
        <td>object</td>
        <td>A label query over volumes to consider for binding.</td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>VolumeName is the binding reference to the PersistentVolume backing this claim.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecresources">
  PostgresCluster.spec.backups.pgbackrest.repos[index].volume.volumeClaimSpec.resources
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>true</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecdatasource">
  PostgresCluster.spec.backups.pgbackrest.repos[index].volume.volumeClaimSpec.dataSource
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>Kind is the type of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name is the name of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecselector">
  PostgresCluster.spec.backups.pgbackrest.repos[index].volume.volumeClaimSpec.selector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



A label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repos[index].volume.volumeClaimSpec.selector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestreposindexvolumevolumeclaimspecselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindex">
  PostgresCluster.spec.backups.pgbackrest.configuration[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>information about the configMap data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>information about the downwardAPI data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexsecret">secret</a></b></td>
        <td>object</td>
        <td>information about the secret data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>information about the serviceAccountToken data to project</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexconfigmap">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].configMap
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexconfigmapitemsindex">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].configMap.items[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindexconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexdownwardapi">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].downwardAPI
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>Items is a list of DownwardAPIVolume file</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindex">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].downwardAPI.items[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapi">↩ Parent</a></sup></sup>
</h3>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.</td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindexfieldref">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].downwardAPI.items[index].fieldRef
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>Path of the field to select in the specified API version.</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>Version of the schema the FieldPath is written in terms of, defaults to "v1".</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindexresourcefieldref">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].downwardAPI.items[index].resourceFieldRef
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>Required: resource to select</td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>Container name: required for volumes, optional for env vars</td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>Specifies the output format of the exposed resources, defaults to "1"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexsecret">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].secret
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestconfigurationindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexsecretitemsindex">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].secret.items[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindexsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestconfigurationindexserviceaccounttoken">
  PostgresCluster.spec.backups.pgbackrest.configuration[index].serviceAccountToken
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Path is the path relative to the mount point of the file to project the token into.</td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.</td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestjobs">
  PostgresCluster.spec.backups.pgbackrest.jobs
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Jobs field allows configuration for all backup jobs

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgBackRest backup Job pods. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestjobsresources">resources</a></b></td>
        <td>object</td>
        <td>Resource limits for backup jobs. Includes manual, scheduled and replica create backups</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestjobsresources">
  PostgresCluster.spec.backups.pgbackrest.jobs.resources
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestjobs">↩ Parent</a></sup></sup>
</h3>



Resource limits for backup jobs. Includes manual, scheduled and replica create backups

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestmanual">
  PostgresCluster.spec.backups.pgbackrest.manual
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Defines details for manual pgBackRest backup Jobs

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>repoName</b></td>
        <td>string</td>
        <td>The name of the pgBackRest repo to run the backup command against.</td>
        <td>true</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>[]string</td>
        <td>Command line options to include when running the pgBackRest backup command. https://pgbackrest.org/command.html#command-backup</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestmetadata">
  PostgresCluster.spec.backups.pgbackrest.metadata
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Metadata contains metadata for PostgresCluster resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohost">
  PostgresCluster.spec.backups.pgbackrest.repoHost
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Defines configuration for a pgBackRest dedicated repository host.  This section is only applicable if at least one "volume" (i.e. PVC-based) repository is defined in the "repos" section, therefore enabling a dedicated repository host Deployment.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of the Dedicated repo host pod. Changing this value causes repo host to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgBackRest repo host pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for a pgBackRest repository host</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostsshconfigmap">sshConfigMap</a></b></td>
        <td>object</td>
        <td>ConfigMap containing custom SSH configuration. Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostsshsecret">sshSecret</a></b></td>
        <td>object</td>
        <td>Secret containing custom SSH keys. Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohosttolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of a PgBackRest repo host pod. Changing this value causes a restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>Topology spread constraints of a Dedicated repo host pod. Changing this value causes the repo host to restart. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinity">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohost">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of the Dedicated repo host pod. Changing this value causes repo host to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinity">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinity">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinity">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostresources">
  PostgresCluster.spec.backups.pgbackrest.repoHost.resources
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohost">↩ Parent</a></sup></sup>
</h3>



Resource requirements for a pgBackRest repository host

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostsshconfigmap">
  PostgresCluster.spec.backups.pgbackrest.repoHost.sshConfigMap
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohost">↩ Parent</a></sup></sup>
</h3>



ConfigMap containing custom SSH configuration. Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostsshconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostsshconfigmapitemsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.sshConfigMap.items[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostsshconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostsshsecret">
  PostgresCluster.spec.backups.pgbackrest.repoHost.sshSecret
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohost">↩ Parent</a></sup></sup>
</h3>



Secret containing custom SSH keys. Deprecated: Repository hosts use mTLS for encryption, authentication, and authorization.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohostsshsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohostsshsecretitemsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.sshSecret.items[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohostsshsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohosttolerationsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.tolerations[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohost">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.topologySpreadConstraints[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohost">↩ Parent</a></sup></sup>
</h3>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>MaxSkew describes the degree to which pods may be unevenly distributed. When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 1/1/0: | zone1 | zone2 | zone3 | |   P   |   P   |       | - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 1/1/1; scheduling it onto zone1(zone2) would make the ActualSkew(2-0) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence to topologies that satisfy it. It's a required field. Default value is 1 and 0 is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each <key, value> as a "bucket", and try to put balanced number of pods into each bucket. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location, but giving higher precedence to topologies that would help reduce the skew. A constraint is considered "Unsatisfiable" for an incoming pod if and only if every possible node assigment for that pod would violate "MaxSkew" on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: | zone1 | zone2 | zone3 | | P P P |   P   |   P   | If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won't make it *more* imbalanced. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindexlabelselector">
  PostgresCluster.spec.backups.pgbackrest.repoHost.topologySpreadConstraints[index].labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindex">↩ Parent</a></sup></sup>
</h3>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.repoHost.topologySpreadConstraints[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrepohosttopologyspreadconstraintsindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestore">
  PostgresCluster.spec.backups.pgbackrest.restore
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Defines details for performing an in-place restore using pgBackRest

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>Whether or not in-place pgBackRest restores are enabled for this PostgresCluster.</td>
        <td>true</td>
      </tr><tr>
        <td><b>repoName</b></td>
        <td>string</td>
        <td>The name of the pgBackRest repo within the source PostgresCluster that contains the backups that should be utilized to perform a pgBackRest restore when initializing the data source for the new PostgresCluster.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterName</b></td>
        <td>string</td>
        <td>The name of an existing PostgresCluster to use as the data source for the new PostgresCluster. Defaults to the name of the PostgresCluster being created if not provided.</td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterNamespace</b></td>
        <td>string</td>
        <td>The namespace of the cluster specified as the data source using the clusterName field. Defaults to the namespace of the PostgresCluster being created if not provided.</td>
        <td>false</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>[]string</td>
        <td>Command line options to include when running the pgBackRest restore command. https://pgbackrest.org/command.html#command-restore</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgBackRest restore Job pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for the pgBackRest restore Job.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoretolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinity">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestore">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinity">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinity">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinity">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestoreaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoreresources">
  PostgresCluster.spec.backups.pgbackrest.restore.resources
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestore">↩ Parent</a></sup></sup>
</h3>



Resource requirements for the pgBackRest restore Job.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestrestoretolerationsindex">
  PostgresCluster.spec.backups.pgbackrest.restore.tolerations[index]
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestrestore">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestsidecars">
  PostgresCluster.spec.backups.pgbackrest.sidecars
  <sup><sup><a href="#postgresclusterspecbackupspgbackrest">↩ Parent</a></sup></sup>
</h3>



Configuration for pgBackRest sidecar containers

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestsidecarspgbackrest">pgbackrest</a></b></td>
        <td>object</td>
        <td>Defines the configuration for the pgBackRest sidecar container</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestsidecarspgbackrestconfig">pgbackrestConfig</a></b></td>
        <td>object</td>
        <td>Defines the configuration for the pgBackRest config sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestsidecarspgbackrest">
  PostgresCluster.spec.backups.pgbackrest.sidecars.pgbackrest
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestsidecars">↩ Parent</a></sup></sup>
</h3>



Defines the configuration for the pgBackRest sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestsidecarspgbackrestresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for a sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestsidecarspgbackrestresources">
  PostgresCluster.spec.backups.pgbackrest.sidecars.pgbackrest.resources
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestsidecarspgbackrest">↩ Parent</a></sup></sup>
</h3>



Resource requirements for a sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestsidecarspgbackrestconfig">
  PostgresCluster.spec.backups.pgbackrest.sidecars.pgbackrestConfig
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestsidecars">↩ Parent</a></sup></sup>
</h3>



Defines the configuration for the pgBackRest config sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecbackupspgbackrestsidecarspgbackrestconfigresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for a sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecbackupspgbackrestsidecarspgbackrestconfigresources">
  PostgresCluster.spec.backups.pgbackrest.sidecars.pgbackrestConfig.resources
  <sup><sup><a href="#postgresclusterspecbackupspgbackrestsidecarspgbackrestconfig">↩ Parent</a></sup></sup>
</h3>



Resource requirements for a sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindex">
  PostgresCluster.spec.instances[index]
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexdatavolumeclaimspec">dataVolumeClaimSpec</a></b></td>
        <td>object</td>
        <td>Defines a PersistentVolumeClaim for PostgreSQL data. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of a PostgreSQL pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexmetadata">metadata</a></b></td>
        <td>object</td>
        <td>Metadata contains metadata for PostgresCluster resources</td>
        <td>false</td>
      </tr><tr>
        <td><b>minAvailable</b></td>
        <td>int or string</td>
        <td>Minimum number of pods that should be available at a time. Defaults to one when the replicas field is greater than one.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name that associates this set of PostgreSQL pods. This field is optional when only one instance set is defined. Each instance set in a cluster must have a unique name. The combined length of this and the cluster name must be 46 characters or less.</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the PostgreSQL pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>Number of desired PostgreSQL pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexresources">resources</a></b></td>
        <td>object</td>
        <td>Compute resources of a PostgreSQL container.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexsidecars">sidecars</a></b></td>
        <td>object</td>
        <td>Configuration for instance sidecar containers</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindextolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of a PostgreSQL pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindextopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>Topology spread constraints of a PostgreSQL pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexwalvolumeclaimspec">walVolumeClaimSpec</a></b></td>
        <td>object</td>
        <td>Defines a separate PersistentVolumeClaim for PostgreSQL's write-ahead log. More info: https://www.postgresql.org/docs/current/wal.html</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexdatavolumeclaimspec">
  PostgresCluster.spec.instances[index].dataVolumeClaimSpec
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



Defines a PersistentVolumeClaim for PostgreSQL data. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexdatavolumeclaimspecresources">resources</a></b></td>
        <td>object</td>
        <td>Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexdatavolumeclaimspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexdatavolumeclaimspecselector">selector</a></b></td>
        <td>object</td>
        <td>A label query over volumes to consider for binding.</td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>VolumeName is the binding reference to the PersistentVolume backing this claim.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexdatavolumeclaimspecresources">
  PostgresCluster.spec.instances[index].dataVolumeClaimSpec.resources
  <sup><sup><a href="#postgresclusterspecinstancesindexdatavolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>true</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexdatavolumeclaimspecdatasource">
  PostgresCluster.spec.instances[index].dataVolumeClaimSpec.dataSource
  <sup><sup><a href="#postgresclusterspecinstancesindexdatavolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>Kind is the type of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name is the name of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexdatavolumeclaimspecselector">
  PostgresCluster.spec.instances[index].dataVolumeClaimSpec.selector
  <sup><sup><a href="#postgresclusterspecinstancesindexdatavolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



A label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexdatavolumeclaimspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexdatavolumeclaimspecselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].dataVolumeClaimSpec.selector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexdatavolumeclaimspecselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinity">
  PostgresCluster.spec.instances[index].affinity
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of a PostgreSQL pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinity">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.instances[index].affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinity">
  PostgresCluster.spec.instances[index].affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.instances[index].affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.instances[index].affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.instances[index].affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.instances[index].affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.instances[index].affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinity">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexmetadata">
  PostgresCluster.spec.instances[index].metadata
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



Metadata contains metadata for PostgresCluster resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexresources">
  PostgresCluster.spec.instances[index].resources
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



Compute resources of a PostgreSQL container.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexsidecars">
  PostgresCluster.spec.instances[index].sidecars
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



Configuration for instance sidecar containers

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexsidecarsreplicacertcopy">replicaCertCopy</a></b></td>
        <td>object</td>
        <td>Defines the configuration for the replica cert copy sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexsidecarsreplicacertcopy">
  PostgresCluster.spec.instances[index].sidecars.replicaCertCopy
  <sup><sup><a href="#postgresclusterspecinstancesindexsidecars">↩ Parent</a></sup></sup>
</h3>



Defines the configuration for the replica cert copy sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexsidecarsreplicacertcopyresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for a sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexsidecarsreplicacertcopyresources">
  PostgresCluster.spec.instances[index].sidecars.replicaCertCopy.resources
  <sup><sup><a href="#postgresclusterspecinstancesindexsidecarsreplicacertcopy">↩ Parent</a></sup></sup>
</h3>



Resource requirements for a sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindextolerationsindex">
  PostgresCluster.spec.instances[index].tolerations[index]
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindextopologyspreadconstraintsindex">
  PostgresCluster.spec.instances[index].topologySpreadConstraints[index]
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>MaxSkew describes the degree to which pods may be unevenly distributed. When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 1/1/0: | zone1 | zone2 | zone3 | |   P   |   P   |       | - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 1/1/1; scheduling it onto zone1(zone2) would make the ActualSkew(2-0) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence to topologies that satisfy it. It's a required field. Default value is 1 and 0 is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each <key, value> as a "bucket", and try to put balanced number of pods into each bucket. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location, but giving higher precedence to topologies that would help reduce the skew. A constraint is considered "Unsatisfiable" for an incoming pod if and only if every possible node assigment for that pod would violate "MaxSkew" on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: | zone1 | zone2 | zone3 | | P P P |   P   |   P   | If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won't make it *more* imbalanced. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindextopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindextopologyspreadconstraintsindexlabelselector">
  PostgresCluster.spec.instances[index].topologySpreadConstraints[index].labelSelector
  <sup><sup><a href="#postgresclusterspecinstancesindextopologyspreadconstraintsindex">↩ Parent</a></sup></sup>
</h3>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindextopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindextopologyspreadconstraintsindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].topologySpreadConstraints[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindextopologyspreadconstraintsindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexwalvolumeclaimspec">
  PostgresCluster.spec.instances[index].walVolumeClaimSpec
  <sup><sup><a href="#postgresclusterspecinstancesindex">↩ Parent</a></sup></sup>
</h3>



Defines a separate PersistentVolumeClaim for PostgreSQL's write-ahead log. More info: https://www.postgresql.org/docs/current/wal.html

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexwalvolumeclaimspecresources">resources</a></b></td>
        <td>object</td>
        <td>Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexwalvolumeclaimspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecinstancesindexwalvolumeclaimspecselector">selector</a></b></td>
        <td>object</td>
        <td>A label query over volumes to consider for binding.</td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>VolumeName is the binding reference to the PersistentVolume backing this claim.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexwalvolumeclaimspecresources">
  PostgresCluster.spec.instances[index].walVolumeClaimSpec.resources
  <sup><sup><a href="#postgresclusterspecinstancesindexwalvolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>true</td>
      </tr><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexwalvolumeclaimspecdatasource">
  PostgresCluster.spec.instances[index].walVolumeClaimSpec.dataSource
  <sup><sup><a href="#postgresclusterspecinstancesindexwalvolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>Kind is the type of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name is the name of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexwalvolumeclaimspecselector">
  PostgresCluster.spec.instances[index].walVolumeClaimSpec.selector
  <sup><sup><a href="#postgresclusterspecinstancesindexwalvolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



A label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecinstancesindexwalvolumeclaimspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecinstancesindexwalvolumeclaimspecselectormatchexpressionsindex">
  PostgresCluster.spec.instances[index].walVolumeClaimSpec.selector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecinstancesindexwalvolumeclaimspecselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfig">
  PostgresCluster.spec.config
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindex">files</a></b></td>
        <td>[]object</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindex">
  PostgresCluster.spec.config.files[index]
  <sup><sup><a href="#postgresclusterspecconfig">↩ Parent</a></sup></sup>
</h3>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>information about the configMap data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>information about the downwardAPI data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>information about the secret data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>information about the serviceAccountToken data to project</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexconfigmap">
  PostgresCluster.spec.config.files[index].configMap
  <sup><sup><a href="#postgresclusterspecconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexconfigmapitemsindex">
  PostgresCluster.spec.config.files[index].configMap.items[index]
  <sup><sup><a href="#postgresclusterspecconfigfilesindexconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexdownwardapi">
  PostgresCluster.spec.config.files[index].downwardAPI
  <sup><sup><a href="#postgresclusterspecconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>Items is a list of DownwardAPIVolume file</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexdownwardapiitemsindex">
  PostgresCluster.spec.config.files[index].downwardAPI.items[index]
  <sup><sup><a href="#postgresclusterspecconfigfilesindexdownwardapi">↩ Parent</a></sup></sup>
</h3>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.</td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexdownwardapiitemsindexfieldref">
  PostgresCluster.spec.config.files[index].downwardAPI.items[index].fieldRef
  <sup><sup><a href="#postgresclusterspecconfigfilesindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>Path of the field to select in the specified API version.</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>Version of the schema the FieldPath is written in terms of, defaults to "v1".</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexdownwardapiitemsindexresourcefieldref">
  PostgresCluster.spec.config.files[index].downwardAPI.items[index].resourceFieldRef
  <sup><sup><a href="#postgresclusterspecconfigfilesindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>Required: resource to select</td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>Container name: required for volumes, optional for env vars</td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>Specifies the output format of the exposed resources, defaults to "1"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexsecret">
  PostgresCluster.spec.config.files[index].secret
  <sup><sup><a href="#postgresclusterspecconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecconfigfilesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexsecretitemsindex">
  PostgresCluster.spec.config.files[index].secret.items[index]
  <sup><sup><a href="#postgresclusterspecconfigfilesindexsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecconfigfilesindexserviceaccounttoken">
  PostgresCluster.spec.config.files[index].serviceAccountToken
  <sup><sup><a href="#postgresclusterspecconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Path is the path relative to the mount point of the file to project the token into.</td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.</td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspeccustomreplicationtlssecret">
  PostgresCluster.spec.customReplicationTLSSecret
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



The secret containing the replication client certificates and keys for secure connections to the PostgreSQL server. It will need to contain the client TLS certificate, TLS key and the Certificate Authority certificate with the data keys set to tls.crt, tls.key and ca.crt, respectively. NOTE: If CustomReplicationClientTLSSecret is provided, CustomTLSSecret MUST be provided and the ca.crt provided must be the same.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspeccustomreplicationtlssecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspeccustomreplicationtlssecretitemsindex">
  PostgresCluster.spec.customReplicationTLSSecret.items[index]
  <sup><sup><a href="#postgresclusterspeccustomreplicationtlssecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspeccustomtlssecret">
  PostgresCluster.spec.customTLSSecret
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



The secret containing the Certificates and Keys to encrypt PostgreSQL traffic will need to contain the server TLS certificate, TLS key and the Certificate Authority certificate with the data keys set to tls.crt, tls.key and ca.crt, respectively. It will then be mounted as a volume projection to the '/pgconf/tls' directory. For more information on Kubernetes secret projections, please see https://k8s.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths NOTE: If CustomTLSSecret is provided, CustomReplicationClientTLSSecret MUST be provided and the ca.crt provided must be the same.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspeccustomtlssecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspeccustomtlssecretitemsindex">
  PostgresCluster.spec.customTLSSecret.items[index]
  <sup><sup><a href="#postgresclusterspeccustomtlssecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasource">
  PostgresCluster.spec.dataSource
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



Specifies a data source for bootstrapping the PostgreSQL cluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrest">pgbackrest</a></b></td>
        <td>object</td>
        <td>Defines a pgBackRest cloud-based data source that can be used to pre-populate the the PostgreSQL data directory for a new PostgreSQL cluster using a pgBackRest restore. The PGBackRest field is incompatible with the PostgresCluster field: only one data source can be used for pre-populating a new PostgreSQL cluster</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgrescluster">postgresCluster</a></b></td>
        <td>object</td>
        <td>Defines a pgBackRest data source that can be used to pre-populate the PostgreSQL data directory for a new PostgreSQL cluster using a pgBackRest restore. The PGBackRest field is incompatible with the PostgresCluster field: only one data source can be used for pre-populating a new PostgreSQL cluster</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcevolumes">volumes</a></b></td>
        <td>object</td>
        <td>Defines any existing volumes to reuse for this PostgresCluster.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrest">
  PostgresCluster.spec.dataSource.pgbackrest
  <sup><sup><a href="#postgresclusterspecdatasource">↩ Parent</a></sup></sup>
</h3>



Defines a pgBackRest cloud-based data source that can be used to pre-populate the the PostgreSQL data directory for a new PostgreSQL cluster using a pgBackRest restore. The PGBackRest field is incompatible with the PostgresCluster field: only one data source can be used for pre-populating a new PostgreSQL cluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepo">repo</a></b></td>
        <td>object</td>
        <td>Defines a pgBackRest repository</td>
        <td>true</td>
      </tr><tr>
        <td><b>stanza</b></td>
        <td>string</td>
        <td>The name of an existing pgBackRest stanza to use as the data source for the new PostgresCluster. Defaults to `db` if not provided.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindex">configuration</a></b></td>
        <td>[]object</td>
        <td>Projected volumes containing custom pgBackRest configuration.  These files are mounted under "/etc/pgbackrest/conf.d" alongside any pgBackRest configuration generated by the PostgreSQL Operator: https://pgbackrest.org/configuration.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>global</b></td>
        <td>map[string]string</td>
        <td>Global pgBackRest configuration settings.  These settings are included in the "global" section of the pgBackRest configuration generated by the PostgreSQL Operator, and then mounted under "/etc/pgbackrest/conf.d": https://pgbackrest.org/configuration.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>[]string</td>
        <td>Command line options to include when running the pgBackRest restore command. https://pgbackrest.org/command.html#command-restore</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgBackRest restore Job pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for the pgBackRest restore Job.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackresttolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepo">
  PostgresCluster.spec.dataSource.pgbackrest.repo
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrest">↩ Parent</a></sup></sup>
</h3>



Defines a pgBackRest repository

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>The name of the the repository</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepoazure">azure</a></b></td>
        <td>object</td>
        <td>Represents a pgBackRest repository that is created using Azure storage</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepogcs">gcs</a></b></td>
        <td>object</td>
        <td>Represents a pgBackRest repository that is created using Google Cloud Storage</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepos3">s3</a></b></td>
        <td>object</td>
        <td>RepoS3 represents a pgBackRest repository that is created using AWS S3 (or S3-compatible) storage</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestreposchedules">schedules</a></b></td>
        <td>object</td>
        <td>Defines the schedules for the pgBackRest backups Full, Differential and Incremental backup types are supported: https://pgbackrest.org/user-guide.html#concept/backup</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepovolume">volume</a></b></td>
        <td>object</td>
        <td>Represents a pgBackRest repository that is created using a PersistentVolumeClaim</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepoazure">
  PostgresCluster.spec.dataSource.pgbackrest.repo.azure
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepo">↩ Parent</a></sup></sup>
</h3>



Represents a pgBackRest repository that is created using Azure storage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>container</b></td>
        <td>string</td>
        <td>The Azure container utilized for the repository</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepogcs">
  PostgresCluster.spec.dataSource.pgbackrest.repo.gcs
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepo">↩ Parent</a></sup></sup>
</h3>



Represents a pgBackRest repository that is created using Google Cloud Storage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>bucket</b></td>
        <td>string</td>
        <td>The GCS bucket utilized for the repository</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepos3">
  PostgresCluster.spec.dataSource.pgbackrest.repo.s3
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepo">↩ Parent</a></sup></sup>
</h3>



RepoS3 represents a pgBackRest repository that is created using AWS S3 (or S3-compatible) storage

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>bucket</b></td>
        <td>string</td>
        <td>The S3 bucket utilized for the repository</td>
        <td>true</td>
      </tr><tr>
        <td><b>endpoint</b></td>
        <td>string</td>
        <td>A valid endpoint corresponding to the specified region</td>
        <td>true</td>
      </tr><tr>
        <td><b>region</b></td>
        <td>string</td>
        <td>The region corresponding to the S3 bucket</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestreposchedules">
  PostgresCluster.spec.dataSource.pgbackrest.repo.schedules
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepo">↩ Parent</a></sup></sup>
</h3>



Defines the schedules for the pgBackRest backups Full, Differential and Incremental backup types are supported: https://pgbackrest.org/user-guide.html#concept/backup

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>differential</b></td>
        <td>string</td>
        <td>Defines the Cron schedule for a differential pgBackRest backup. Follows the standard Cron schedule syntax: https://k8s.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax</td>
        <td>false</td>
      </tr><tr>
        <td><b>full</b></td>
        <td>string</td>
        <td>Defines the Cron schedule for a full pgBackRest backup. Follows the standard Cron schedule syntax: https://k8s.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax</td>
        <td>false</td>
      </tr><tr>
        <td><b>incremental</b></td>
        <td>string</td>
        <td>Defines the Cron schedule for an incremental pgBackRest backup. Follows the standard Cron schedule syntax: https://k8s.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepovolume">
  PostgresCluster.spec.dataSource.pgbackrest.repo.volume
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepo">↩ Parent</a></sup></sup>
</h3>



Represents a pgBackRest repository that is created using a PersistentVolumeClaim

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspec">volumeClaimSpec</a></b></td>
        <td>object</td>
        <td>Defines a PersistentVolumeClaim spec used to create and/or bind a volume</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspec">
  PostgresCluster.spec.dataSource.pgbackrest.repo.volume.volumeClaimSpec
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepovolume">↩ Parent</a></sup></sup>
</h3>



Defines a PersistentVolumeClaim spec used to create and/or bind a volume

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecresources">resources</a></b></td>
        <td>object</td>
        <td>Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecselector">selector</a></b></td>
        <td>object</td>
        <td>A label query over volumes to consider for binding.</td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>VolumeName is the binding reference to the PersistentVolume backing this claim.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecdatasource">
  PostgresCluster.spec.dataSource.pgbackrest.repo.volume.volumeClaimSpec.dataSource
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>Kind is the type of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name is the name of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecresources">
  PostgresCluster.spec.dataSource.pgbackrest.repo.volume.volumeClaimSpec.resources
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecselector">
  PostgresCluster.spec.dataSource.pgbackrest.repo.volume.volumeClaimSpec.selector
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



A label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.repo.volume.volumeClaimSpec.selector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestrepovolumevolumeclaimspecselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinity">
  PostgresCluster.spec.dataSource.pgbackrest.affinity
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrest">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinity">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinity">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinity">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.pgbackrest.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindex">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrest">↩ Parent</a></sup></sup>
</h3>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>information about the configMap data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>information about the downwardAPI data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexsecret">secret</a></b></td>
        <td>object</td>
        <td>information about the secret data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>information about the serviceAccountToken data to project</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexconfigmap">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].configMap
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexconfigmapitemsindex">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].configMap.items[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapi">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].downwardAPI
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>Items is a list of DownwardAPIVolume file</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindex">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].downwardAPI.items[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapi">↩ Parent</a></sup></sup>
</h3>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.</td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindexfieldref">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].downwardAPI.items[index].fieldRef
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>Path of the field to select in the specified API version.</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>Version of the schema the FieldPath is written in terms of, defaults to "v1".</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindexresourcefieldref">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].downwardAPI.items[index].resourceFieldRef
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>Required: resource to select</td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>Container name: required for volumes, optional for env vars</td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>Specifies the output format of the exposed resources, defaults to "1"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexsecret">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].secret
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexsecretitemsindex">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].secret.items[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindexsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestconfigurationindexserviceaccounttoken">
  PostgresCluster.spec.dataSource.pgbackrest.configuration[index].serviceAccountToken
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrestconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Path is the path relative to the mount point of the file to project the token into.</td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.</td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackrestresources">
  PostgresCluster.spec.dataSource.pgbackrest.resources
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrest">↩ Parent</a></sup></sup>
</h3>



Resource requirements for the pgBackRest restore Job.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepgbackresttolerationsindex">
  PostgresCluster.spec.dataSource.pgbackrest.tolerations[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepgbackrest">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgrescluster">
  PostgresCluster.spec.dataSource.postgresCluster
  <sup><sup><a href="#postgresclusterspecdatasource">↩ Parent</a></sup></sup>
</h3>



Defines a pgBackRest data source that can be used to pre-populate the PostgreSQL data directory for a new PostgreSQL cluster using a pgBackRest restore. The PGBackRest field is incompatible with the PostgresCluster field: only one data source can be used for pre-populating a new PostgreSQL cluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>repoName</b></td>
        <td>string</td>
        <td>The name of the pgBackRest repo within the source PostgresCluster that contains the backups that should be utilized to perform a pgBackRest restore when initializing the data source for the new PostgresCluster.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterName</b></td>
        <td>string</td>
        <td>The name of an existing PostgresCluster to use as the data source for the new PostgresCluster. Defaults to the name of the PostgresCluster being created if not provided.</td>
        <td>false</td>
      </tr><tr>
        <td><b>clusterNamespace</b></td>
        <td>string</td>
        <td>The namespace of the cluster specified as the data source using the clusterName field. Defaults to the namespace of the PostgresCluster being created if not provided.</td>
        <td>false</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>[]string</td>
        <td>Command line options to include when running the pgBackRest restore command. https://pgbackrest.org/command.html#command-restore</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgBackRest restore Job pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusterresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for the pgBackRest restore Job.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclustertolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinity">
  PostgresCluster.spec.dataSource.postgresCluster.affinity
  <sup><sup><a href="#postgresclusterspecdatasourcepostgrescluster">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of the pgBackRest restore Job. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinity">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinity">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinity">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.dataSource.postgresCluster.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgresclusteraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclusterresources">
  PostgresCluster.spec.dataSource.postgresCluster.resources
  <sup><sup><a href="#postgresclusterspecdatasourcepostgrescluster">↩ Parent</a></sup></sup>
</h3>



Resource requirements for the pgBackRest restore Job.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcepostgresclustertolerationsindex">
  PostgresCluster.spec.dataSource.postgresCluster.tolerations[index]
  <sup><sup><a href="#postgresclusterspecdatasourcepostgrescluster">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcevolumes">
  PostgresCluster.spec.dataSource.volumes
  <sup><sup><a href="#postgresclusterspecdatasource">↩ Parent</a></sup></sup>
</h3>



Defines any existing volumes to reuse for this PostgresCluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecdatasourcevolumespgbackrestvolume">pgBackRestVolume</a></b></td>
        <td>object</td>
        <td>Defines the existing pgBackRest repo volume and directory to use in the current PostgresCluster.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcevolumespgdatavolume">pgDataVolume</a></b></td>
        <td>object</td>
        <td>Defines the existing pgData volume and directory to use in the current PostgresCluster.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecdatasourcevolumespgwalvolume">pgWALVolume</a></b></td>
        <td>object</td>
        <td>Defines the existing pg_wal volume and directory to use in the current PostgresCluster. Note that a defined pg_wal volume MUST be accompanied by a pgData volume.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcevolumespgbackrestvolume">
  PostgresCluster.spec.dataSource.volumes.pgBackRestVolume
  <sup><sup><a href="#postgresclusterspecdatasourcevolumes">↩ Parent</a></sup></sup>
</h3>



Defines the existing pgBackRest repo volume and directory to use in the current PostgresCluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pvcName</b></td>
        <td>string</td>
        <td>The existing PVC name.</td>
        <td>true</td>
      </tr><tr>
        <td><b>directory</b></td>
        <td>string</td>
        <td>The existing directory. When not set, a move Job is not created for the associated volume.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcevolumespgdatavolume">
  PostgresCluster.spec.dataSource.volumes.pgDataVolume
  <sup><sup><a href="#postgresclusterspecdatasourcevolumes">↩ Parent</a></sup></sup>
</h3>



Defines the existing pgData volume and directory to use in the current PostgresCluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pvcName</b></td>
        <td>string</td>
        <td>The existing PVC name.</td>
        <td>true</td>
      </tr><tr>
        <td><b>directory</b></td>
        <td>string</td>
        <td>The existing directory. When not set, a move Job is not created for the associated volume.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatasourcevolumespgwalvolume">
  PostgresCluster.spec.dataSource.volumes.pgWALVolume
  <sup><sup><a href="#postgresclusterspecdatasourcevolumes">↩ Parent</a></sup></sup>
</h3>



Defines the existing pg_wal volume and directory to use in the current PostgresCluster. Note that a defined pg_wal volume MUST be accompanied by a pgData volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>pvcName</b></td>
        <td>string</td>
        <td>The existing PVC name.</td>
        <td>true</td>
      </tr><tr>
        <td><b>directory</b></td>
        <td>string</td>
        <td>The existing directory. When not set, a move Job is not created for the associated volume.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecdatabaseinitsql">
  PostgresCluster.spec.databaseInitSQL
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



DatabaseInitSQL defines a ConfigMap containing custom SQL that will be run after the cluster is initialized. This ConfigMap must be in the same namespace as the cluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the ConfigMap data key that points to a SQL string</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name is the name of a ConfigMap</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecimagepullsecretsindex">
  PostgresCluster.spec.imagePullSecrets[index]
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



LocalObjectReference contains enough information to let you locate the referenced object inside the same namespace.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmetadata">
  PostgresCluster.spec.metadata
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



Metadata contains metadata for PostgresCluster resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoring">
  PostgresCluster.spec.monitoring
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



The specification of monitoring tools that connect to PostgreSQL

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitor">pgmonitor</a></b></td>
        <td>object</td>
        <td>PGMonitorSpec defines the desired state of the pgMonitor tool suite</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitor">
  PostgresCluster.spec.monitoring.pgmonitor
  <sup><sup><a href="#postgresclusterspecmonitoring">↩ Parent</a></sup></sup>
</h3>



PGMonitorSpec defines the desired state of the pgMonitor tool suite

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporter">exporter</a></b></td>
        <td>object</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporter">
  PostgresCluster.spec.monitoring.pgmonitor.exporter
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitor">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindex">configuration</a></b></td>
        <td>[]object</td>
        <td>Projected volumes containing custom PostgreSQL Exporter configuration.  Currently supports the customization of PostgreSQL Exporter queries. If a "queries.yaml" file is detected in any volume projected using this field, it will be loaded using the "extend.query-path" flag: https://github.com/prometheus-community/postgres_exporter#flags Changing the values of field causes PostgreSQL and the exporter to restart.</td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>The image name to use for crunchy-postgres-exporter containers. The image may also be set using the RELATED_IMAGE_PGEXPORTER environment variable.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterresources">resources</a></b></td>
        <td>object</td>
        <td>Changing this value causes PostgreSQL and the exporter to restart. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindex">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index]
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporter">↩ Parent</a></sup></sup>
</h3>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>information about the configMap data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>information about the downwardAPI data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexsecret">secret</a></b></td>
        <td>object</td>
        <td>information about the secret data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>information about the serviceAccountToken data to project</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexconfigmap">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].configMap
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexconfigmapitemsindex">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].configMap.items[index]
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapi">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].downwardAPI
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>Items is a list of DownwardAPIVolume file</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindex">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].downwardAPI.items[index]
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapi">↩ Parent</a></sup></sup>
</h3>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.</td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindexfieldref">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].downwardAPI.items[index].fieldRef
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>Path of the field to select in the specified API version.</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>Version of the schema the FieldPath is written in terms of, defaults to "v1".</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindexresourcefieldref">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].downwardAPI.items[index].resourceFieldRef
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>Required: resource to select</td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>Container name: required for volumes, optional for env vars</td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>Specifies the output format of the exposed resources, defaults to "1"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexsecret">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].secret
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexsecretitemsindex">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].secret.items[index]
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindexsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterconfigurationindexserviceaccounttoken">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.configuration[index].serviceAccountToken
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporterconfigurationindex">↩ Parent</a></sup></sup>
</h3>



information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Path is the path relative to the mount point of the file to project the token into.</td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.</td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecmonitoringpgmonitorexporterresources">
  PostgresCluster.spec.monitoring.pgmonitor.exporter.resources
  <sup><sup><a href="#postgresclusterspecmonitoringpgmonitorexporter">↩ Parent</a></sup></sup>
</h3>



Changing this value causes PostgreSQL and the exporter to restart. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecpatroni">
  PostgresCluster.spec.patroni
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>dynamicConfiguration</b></td>
        <td>object</td>
        <td>Patroni dynamic configuration settings. Changes to this value will be automatically reloaded without validation. Changes to certain PostgreSQL parameters cause PostgreSQL to restart. More info: https://patroni.readthedocs.io/en/latest/SETTINGS.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>leaderLeaseDurationSeconds</b></td>
        <td>integer</td>
        <td>TTL of the cluster leader lock. "Think of it as the length of time before initiation of the automatic failover process." Changing this value causes PostgreSQL to restart.</td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>The port on which Patroni should listen. Changing this value causes PostgreSQL to restart.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecpatroniswitchover">switchover</a></b></td>
        <td>object</td>
        <td>Switchover gives options to perform ad hoc switchovers in a PostgresCluster.</td>
        <td>false</td>
      </tr><tr>
        <td><b>syncPeriodSeconds</b></td>
        <td>integer</td>
        <td>The interval for refreshing the leader lock and applying dynamicConfiguration. Must be less than leaderLeaseDurationSeconds. Changing this value causes PostgreSQL to restart.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecpatroniswitchover">
  PostgresCluster.spec.patroni.switchover
  <sup><sup><a href="#postgresclusterspecpatroni">↩ Parent</a></sup></sup>
</h3>



Switchover gives options to perform ad hoc switchovers in a PostgresCluster.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>Whether or not the operator should allow switchovers in a PostgresCluster</td>
        <td>true</td>
      </tr><tr>
        <td><b>targetInstance</b></td>
        <td>string</td>
        <td>The instance that should become primary during a switchover. This field is optional when Type is "Switchover" and required when Type is "Failover". When it is not specified, a healthy replica is automatically selected.</td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>Type of switchover to perform. Valid options are Switchover and Failover. "Switchover" changes the primary instance of a healthy PostgresCluster. "Failover" forces a particular instance to be primary, regardless of other factors. A TargetInstance must be specified to failover. NOTE: The Failover type is reserved as the "last resort" case.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxy">
  PostgresCluster.spec.proxy
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



The specification of a proxy that connects to PostgreSQL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncer">pgBouncer</a></b></td>
        <td>object</td>
        <td>Defines a PgBouncer proxy and connection pooler.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncer">
  PostgresCluster.spec.proxy.pgBouncer
  <sup><sup><a href="#postgresclusterspecproxy">↩ Parent</a></sup></sup>
</h3>



Defines a PgBouncer proxy and connection pooler.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of a PgBouncer pod. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfig">config</a></b></td>
        <td>object</td>
        <td>Configuration settings for the PgBouncer process. Changes to any of these values will be automatically reloaded without validation. Be careful, as you may put PgBouncer into an unusable state. More info: https://www.pgbouncer.org/usage.html#reload</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncercustomtlssecret">customTLSSecret</a></b></td>
        <td>object</td>
        <td>A secret projection containing a certificate and key with which to encrypt connections to PgBouncer. The "tls.crt", "tls.key", and "ca.crt" paths must be PEM-encoded certificates and keys. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths</td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>Name of a container image that can run PgBouncer 1.15 or newer. Changing this value causes PgBouncer to restart. The image may also be set using the RELATED_IMAGE_PGBOUNCER environment variable. More info: https://kubernetes.io/docs/concepts/containers/images</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncermetadata">metadata</a></b></td>
        <td>object</td>
        <td>Metadata contains metadata for PostgresCluster resources</td>
        <td>false</td>
      </tr><tr>
        <td><b>minAvailable</b></td>
        <td>int or string</td>
        <td>Minimum number of pods that should be available at a time. Defaults to one when the replicas field is greater than one.</td>
        <td>false</td>
      </tr><tr>
        <td><b>port</b></td>
        <td>integer</td>
        <td>Port on which PgBouncer should listen for client connections. Changing this value causes PgBouncer to restart.</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgBouncer pod. Changing this value causes PostgreSQL to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>Number of desired PgBouncer pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerresources">resources</a></b></td>
        <td>object</td>
        <td>Compute resources of a PgBouncer container. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerservice">service</a></b></td>
        <td>object</td>
        <td>Specification of the service that exposes PgBouncer.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncersidecars">sidecars</a></b></td>
        <td>object</td>
        <td>Configuration for pgBouncer sidecar containers</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncertolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of a PgBouncer pod. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncertopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>Topology spread constraints of a PgBouncer pod. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinity">
  PostgresCluster.spec.proxy.pgBouncer.affinity
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of a PgBouncer pod. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinity">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinity">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinity">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbounceraffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfig">
  PostgresCluster.spec.proxy.pgBouncer.config
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



Configuration settings for the PgBouncer process. Changes to any of these values will be automatically reloaded without validation. Be careful, as you may put PgBouncer into an unusable state. More info: https://www.pgbouncer.org/usage.html#reload

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>databases</b></td>
        <td>map[string]string</td>
        <td>PgBouncer database definitions. The key is the database requested by a client while the value is a libpq-styled connection string. The special key "*" acts as a fallback. When this field is empty, PgBouncer is configured with a single "*" entry that connects to the primary PostgreSQL instance. More info: https://www.pgbouncer.org/config.html#section-databases</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindex">files</a></b></td>
        <td>[]object</td>
        <td>Files to mount under "/etc/pgbouncer". When specified, settings in the "pgbouncer.ini" file are loaded before all others. From there, other files may be included by absolute path. Changing these references causes PgBouncer to restart, but changes to the file contents are automatically reloaded. More info: https://www.pgbouncer.org/config.html#include-directive</td>
        <td>false</td>
      </tr><tr>
        <td><b>global</b></td>
        <td>map[string]string</td>
        <td>Settings that apply to the entire PgBouncer process. More info: https://www.pgbouncer.org/config.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>users</b></td>
        <td>map[string]string</td>
        <td>Connection settings specific to particular users. More info: https://www.pgbouncer.org/config.html#section-users</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindex">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfig">↩ Parent</a></sup></sup>
</h3>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>information about the configMap data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>information about the downwardAPI data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>information about the secret data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>information about the serviceAccountToken data to project</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexconfigmap">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].configMap
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexconfigmapitemsindex">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].configMap.items[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindexconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexdownwardapi">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].downwardAPI
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>Items is a list of DownwardAPIVolume file</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindex">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].downwardAPI.items[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapi">↩ Parent</a></sup></sup>
</h3>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.</td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindexfieldref">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].downwardAPI.items[index].fieldRef
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>Path of the field to select in the specified API version.</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>Version of the schema the FieldPath is written in terms of, defaults to "v1".</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindexresourcefieldref">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].downwardAPI.items[index].resourceFieldRef
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>Required: resource to select</td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>Container name: required for volumes, optional for env vars</td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>Specifies the output format of the exposed resources, defaults to "1"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexsecret">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].secret
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncerconfigfilesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexsecretitemsindex">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].secret.items[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindexsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerconfigfilesindexserviceaccounttoken">
  PostgresCluster.spec.proxy.pgBouncer.config.files[index].serviceAccountToken
  <sup><sup><a href="#postgresclusterspecproxypgbouncerconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Path is the path relative to the mount point of the file to project the token into.</td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.</td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncercustomtlssecret">
  PostgresCluster.spec.proxy.pgBouncer.customTLSSecret
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



A secret projection containing a certificate and key with which to encrypt connections to PgBouncer. The "tls.crt", "tls.key", and "ca.crt" paths must be PEM-encoded certificates and keys. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncercustomtlssecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncercustomtlssecretitemsindex">
  PostgresCluster.spec.proxy.pgBouncer.customTLSSecret.items[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncercustomtlssecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncermetadata">
  PostgresCluster.spec.proxy.pgBouncer.metadata
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



Metadata contains metadata for PostgresCluster resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerresources">
  PostgresCluster.spec.proxy.pgBouncer.resources
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



Compute resources of a PgBouncer container. Changing this value causes PgBouncer to restart. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncerservice">
  PostgresCluster.spec.proxy.pgBouncer.service
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



Specification of the service that exposes PgBouncer.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncersidecars">
  PostgresCluster.spec.proxy.pgBouncer.sidecars
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



Configuration for pgBouncer sidecar containers

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncersidecarspgbouncerconfig">pgbouncerConfig</a></b></td>
        <td>object</td>
        <td>Defines the configuration for the pgBouncer config sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncersidecarspgbouncerconfig">
  PostgresCluster.spec.proxy.pgBouncer.sidecars.pgbouncerConfig
  <sup><sup><a href="#postgresclusterspecproxypgbouncersidecars">↩ Parent</a></sup></sup>
</h3>



Defines the configuration for the pgBouncer config sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncersidecarspgbouncerconfigresources">resources</a></b></td>
        <td>object</td>
        <td>Resource requirements for a sidecar container</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncersidecarspgbouncerconfigresources">
  PostgresCluster.spec.proxy.pgBouncer.sidecars.pgbouncerConfig.resources
  <sup><sup><a href="#postgresclusterspecproxypgbouncersidecarspgbouncerconfig">↩ Parent</a></sup></sup>
</h3>



Resource requirements for a sidecar container

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncertolerationsindex">
  PostgresCluster.spec.proxy.pgBouncer.tolerations[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncertopologyspreadconstraintsindex">
  PostgresCluster.spec.proxy.pgBouncer.topologySpreadConstraints[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncer">↩ Parent</a></sup></sup>
</h3>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>MaxSkew describes the degree to which pods may be unevenly distributed. When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 1/1/0: | zone1 | zone2 | zone3 | |   P   |   P   |       | - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 1/1/1; scheduling it onto zone1(zone2) would make the ActualSkew(2-0) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence to topologies that satisfy it. It's a required field. Default value is 1 and 0 is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each <key, value> as a "bucket", and try to put balanced number of pods into each bucket. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location, but giving higher precedence to topologies that would help reduce the skew. A constraint is considered "Unsatisfiable" for an incoming pod if and only if every possible node assigment for that pod would violate "MaxSkew" on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: | zone1 | zone2 | zone3 | | P P P |   P   |   P   | If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won't make it *more* imbalanced. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncertopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncertopologyspreadconstraintsindexlabelselector">
  PostgresCluster.spec.proxy.pgBouncer.topologySpreadConstraints[index].labelSelector
  <sup><sup><a href="#postgresclusterspecproxypgbouncertopologyspreadconstraintsindex">↩ Parent</a></sup></sup>
</h3>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecproxypgbouncertopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecproxypgbouncertopologyspreadconstraintsindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.proxy.pgBouncer.topologySpreadConstraints[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecproxypgbouncertopologyspreadconstraintsindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecservice">
  PostgresCluster.spec.service
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



Specification of the service that exposes the PostgreSQL primary instance.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecstandby">
  PostgresCluster.spec.standby
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



Run this cluster as a read-only copy of an existing cluster or archive.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>repoName</b></td>
        <td>string</td>
        <td>The name of the pgBackRest repository to follow for WAL files.</td>
        <td>true</td>
      </tr><tr>
        <td><b>enabled</b></td>
        <td>boolean</td>
        <td>Whether or not the PostgreSQL cluster should be read-only. When this is true, WAL files are applied from the pgBackRest repository.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterface">
  PostgresCluster.spec.userInterface
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>



The specification of a user interface that connects to PostgreSQL.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmin">pgAdmin</a></b></td>
        <td>object</td>
        <td>Defines a pgAdmin user interface.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmin">
  PostgresCluster.spec.userInterface.pgAdmin
  <sup><sup><a href="#postgresclusterspecuserinterface">↩ Parent</a></sup></sup>
</h3>



Defines a pgAdmin user interface.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspec">dataVolumeClaimSpec</a></b></td>
        <td>object</td>
        <td>Defines a PersistentVolumeClaim for pgAdmin data. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinity">affinity</a></b></td>
        <td>object</td>
        <td>Scheduling constraints of a pgAdmin pod. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfig">config</a></b></td>
        <td>object</td>
        <td>Configuration settings for the pgAdmin process. Changes to any of these values will be loaded without validation. Be careful, as you may put pgAdmin into an unusable state.</td>
        <td>false</td>
      </tr><tr>
        <td><b>image</b></td>
        <td>string</td>
        <td>Name of a container image that can run pgAdmin 4. Changing this value causes pgAdmin to restart. The image may also be set using the RELATED_IMAGE_PGADMIN environment variable. More info: https://kubernetes.io/docs/concepts/containers/images</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminmetadata">metadata</a></b></td>
        <td>object</td>
        <td>Metadata contains metadata for PostgresCluster resources</td>
        <td>false</td>
      </tr><tr>
        <td><b>priorityClassName</b></td>
        <td>string</td>
        <td>Priority class name for the pgAdmin pod. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/pod-priority-preemption/</td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>Number of desired pgAdmin pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminresources">resources</a></b></td>
        <td>object</td>
        <td>Compute resources of a pgAdmin container. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminservice">service</a></b></td>
        <td>object</td>
        <td>Specification of the service that exposes pgAdmin.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmintolerationsindex">tolerations</a></b></td>
        <td>[]object</td>
        <td>Tolerations of a pgAdmin pod. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindex">topologySpreadConstraints</a></b></td>
        <td>[]object</td>
        <td>Topology spread constraints of a pgAdmin pod. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/workloads/pods/pod-topology-spread-constraints/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmindatavolumeclaimspec">
  PostgresCluster.spec.userInterface.pgAdmin.dataVolumeClaimSpec
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



Defines a PersistentVolumeClaim for pgAdmin data. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>accessModes</b></td>
        <td>[]string</td>
        <td>AccessModes contains the desired access modes the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#access-modes-1</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspecdatasource">dataSource</a></b></td>
        <td>object</td>
        <td>This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspecresources">resources</a></b></td>
        <td>object</td>
        <td>Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspecselector">selector</a></b></td>
        <td>object</td>
        <td>A label query over volumes to consider for binding.</td>
        <td>false</td>
      </tr><tr>
        <td><b>storageClassName</b></td>
        <td>string</td>
        <td>Name of the StorageClass required by the claim. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#class-1</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeMode</b></td>
        <td>string</td>
        <td>volumeMode defines what type of volume is required by the claim. Value of Filesystem is implied when not included in claim spec.</td>
        <td>false</td>
      </tr><tr>
        <td><b>volumeName</b></td>
        <td>string</td>
        <td>VolumeName is the binding reference to the PersistentVolume backing this claim.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmindatavolumeclaimspecdatasource">
  PostgresCluster.spec.userInterface.pgAdmin.dataVolumeClaimSpec.dataSource
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



This field can be used to specify either: * An existing VolumeSnapshot object (snapshot.storage.k8s.io/VolumeSnapshot) * An existing PVC (PersistentVolumeClaim) * An existing custom resource that implements data population (Alpha) In order to use custom resource types that implement data population, the AnyVolumeDataSource feature gate must be enabled. If the provisioner or an external controller can support the specified data source, it will create a new volume based on the contents of the specified data source.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>Kind is the type of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name is the name of resource being referenced</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiGroup</b></td>
        <td>string</td>
        <td>APIGroup is the group for the resource being referenced. If APIGroup is not specified, the specified Kind must be in the core API group. For any other third-party types, APIGroup is required.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmindatavolumeclaimspecresources">
  PostgresCluster.spec.userInterface.pgAdmin.dataVolumeClaimSpec.resources
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



Resources represents the minimum resources the volume should have. More info: https://kubernetes.io/docs/concepts/storage/persistent-volumes#resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmindatavolumeclaimspecselector">
  PostgresCluster.spec.userInterface.pgAdmin.dataVolumeClaimSpec.selector
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspec">↩ Parent</a></sup></sup>
</h3>



A label query over volumes to consider for binding.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspecselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmindatavolumeclaimspecselectormatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.dataVolumeClaimSpec.selector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmindatavolumeclaimspecselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinity">
  PostgresCluster.spec.userInterface.pgAdmin.affinity
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



Scheduling constraints of a pgAdmin pod. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinity">nodeAffinity</a></b></td>
        <td>object</td>
        <td>Describes node affinity scheduling rules for the pod.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinity">podAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinity">podAntiAffinity</a></b></td>
        <td>object</td>
        <td>Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinity">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinity">↩ Parent</a></sup></sup>
</h3>



Describes node affinity scheduling rules for the pod.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node matches the corresponding matchExpressions; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



An empty preferred scheduling term matches all objects with implicit weight 0 (i.e. it's a no-op). A null preferred scheduling term matches no objects (i.e. is also a no-op).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">preference</a></b></td>
        <td>object</td>
        <td>A node selector term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>Weight associated with matching the corresponding nodeSelectorTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A node selector term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreferencematchfieldsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].preference.matchFields[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinitypreferredduringschedulingignoredduringexecutionindexpreference">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinity">↩ Parent</a></sup></sup>
</h3>



If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to an update), the system may or may not try to eventually evict the pod from its node.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">nodeSelectorTerms</a></b></td>
        <td>[]object</td>
        <td>Required. A list of node selector terms. The terms are ORed.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecution">↩ Parent</a></sup></sup>
</h3>



A null or empty node selector term matches no objects. The requirements of them are ANDed. The TopologySelectorTerm type implements a subset of the NodeSelectorTerm.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's labels.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">matchFields</a></b></td>
        <td>[]object</td>
        <td>A list of node selector requirements by node's fields.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindexmatchfieldsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.nodeAffinity.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms[index].matchFields[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitynodeaffinityrequiredduringschedulingignoredduringexecutionnodeselectortermsindex">↩ Parent</a></sup></sup>
</h3>



A node selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists, DoesNotExist. Gt, and Lt.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>An array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. If the operator is Gt or Lt, the values array must have a single element, which will be interpreted as an integer. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinity">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod affinity scheduling rules (e.g. co-locate this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinity">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinity">↩ Parent</a></sup></sup>
</h3>



Describes pod anti-affinity scheduling rules (e.g. avoid putting this pod in the same node, zone, etc. as some other pod(s)).

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">preferredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>The scheduler will prefer to schedule pods to nodes that satisfy the anti-affinity expressions specified by this field, but it may choose a node that violates one or more of the expressions. The node that is most preferred is the one with the greatest sum of weights, i.e. for each node that meets all of the scheduling requirements (resource request, requiredDuringScheduling anti-affinity expressions, etc.), compute a sum by iterating through the elements of this field and adding "weight" to the sum if the node has pods which matches the corresponding podAffinityTerm; the node(s) with the highest sum are the most preferred.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">requiredDuringSchedulingIgnoredDuringExecution</a></b></td>
        <td>[]object</td>
        <td>If the anti-affinity requirements specified by this field are not met at scheduling time, the pod will not be scheduled onto the node. If the anti-affinity requirements specified by this field cease to be met at some point during pod execution (e.g. due to a pod label update), the system may or may not try to eventually evict the pod from its node. When there are multiple elements, the lists of nodes corresponding to each podAffinityTerm are intersected, i.e. all terms must be satisfied.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



The weights of all of the matched WeightedPodAffinityTerm fields are added per-node to find the most preferred node(s)

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">podAffinityTerm</a></b></td>
        <td>object</td>
        <td>Required. A pod affinity term, associated with the corresponding weight.</td>
        <td>true</td>
      </tr><tr>
        <td><b>weight</b></td>
        <td>integer</td>
        <td>weight associated with matching the corresponding podAffinityTerm, in the range 1-100.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



Required. A pod affinity term, associated with the corresponding weight.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinityterm">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselectormatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.preferredDuringSchedulingIgnoredDuringExecution[index].podAffinityTerm.labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinitypreferredduringschedulingignoredduringexecutionindexpodaffinitytermlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinity">↩ Parent</a></sup></sup>
</h3>



Defines a set of pods (namely those matching the labelSelector relative to the given namespace(s)) that this pod should be co-located (affinity) or not co-located (anti-affinity) with, where co-located is defined as running on a node whose value of the label with key <topologyKey> matches that of any node on which a pod of the set of pods is running

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>This pod should be co-located (affinity) or not co-located (anti-affinity) with the pods matching the labelSelector in the specified namespaces, where co-located is defined as running on a node whose value of the label with key topologyKey matches that of any node on which any of the selected pods is running. Empty topologyKey is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>A label query over a set of resources, in this case pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>namespaces</b></td>
        <td>[]string</td>
        <td>namespaces specifies which namespaces the labelSelector applies to (matches against); null or empty list means "this pod's namespace"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindex">↩ Parent</a></sup></sup>
</h3>



A label query over a set of resources, in this case pods.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.affinity.podAntiAffinity.requiredDuringSchedulingIgnoredDuringExecution[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminaffinitypodantiaffinityrequiredduringschedulingignoredduringexecutionindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfig">
  PostgresCluster.spec.userInterface.pgAdmin.config
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



Configuration settings for the pgAdmin process. Changes to any of these values will be loaded without validation. Be careful, as you may put pgAdmin into an unusable state.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindex">files</a></b></td>
        <td>[]object</td>
        <td>Files allows the user to mount projected volumes into the pgAdmin container so that files can be referenced by pgAdmin as needed.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigldapbindpassword">ldapBindPassword</a></b></td>
        <td>object</td>
        <td>A Secret containing the value for the LDAP_BIND_PASSWORD setting. More info: https://www.pgadmin.org/docs/pgadmin4/latest/ldap.html</td>
        <td>false</td>
      </tr><tr>
        <td><b>settings</b></td>
        <td>object</td>
        <td>Settings for the pgAdmin server process. Keys should be uppercase and values must be constants. More info: https://www.pgadmin.org/docs/pgadmin4/latest/config_py.html</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindex">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfig">↩ Parent</a></sup></sup>
</h3>



Projection that may be projected along with other supported volume types

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexconfigmap">configMap</a></b></td>
        <td>object</td>
        <td>information about the configMap data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapi">downwardAPI</a></b></td>
        <td>object</td>
        <td>information about the downwardAPI data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexsecret">secret</a></b></td>
        <td>object</td>
        <td>information about the secret data to project</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexserviceaccounttoken">serviceAccountToken</a></b></td>
        <td>object</td>
        <td>information about the serviceAccountToken data to project</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexconfigmap">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].configMap
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the configMap data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexconfigmapitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced ConfigMap will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the ConfigMap, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the ConfigMap or its keys must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexconfigmapitemsindex">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].configMap.items[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexconfigmap">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapi">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].downwardAPI
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the downwardAPI data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>Items is a list of DownwardAPIVolume file</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindex">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].downwardAPI.items[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapi">↩ Parent</a></sup></sup>
</h3>



DownwardAPIVolumeFile represents information to create the file containing the pod field

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Required: Path is  the relative path name of the file to be created. Must not be absolute or contain the '..' path. Must be utf-8 encoded. The first item of the relative path must not start with '..'</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindexfieldref">fieldRef</a></b></td>
        <td>object</td>
        <td>Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.</td>
        <td>false</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file, must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindexresourcefieldref">resourceFieldRef</a></b></td>
        <td>object</td>
        <td>Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindexfieldref">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].downwardAPI.items[index].fieldRef
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Required: Selects a field of the pod: only annotations, labels, name and namespace are supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>fieldPath</b></td>
        <td>string</td>
        <td>Path of the field to select in the specified API version.</td>
        <td>true</td>
      </tr><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>Version of the schema the FieldPath is written in terms of, defaults to "v1".</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindexresourcefieldref">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].downwardAPI.items[index].resourceFieldRef
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexdownwardapiitemsindex">↩ Parent</a></sup></sup>
</h3>



Selects a resource of the container: only resources limits and requests (limits.cpu, limits.memory, requests.cpu and requests.memory) are currently supported.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>resource</b></td>
        <td>string</td>
        <td>Required: resource to select</td>
        <td>true</td>
      </tr><tr>
        <td><b>containerName</b></td>
        <td>string</td>
        <td>Container name: required for volumes, optional for env vars</td>
        <td>false</td>
      </tr><tr>
        <td><b>divisor</b></td>
        <td>int or string</td>
        <td>Specifies the output format of the exposed resources, defaults to "1"</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexsecret">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].secret
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the secret data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexsecretitemsindex">items</a></b></td>
        <td>[]object</td>
        <td>If unspecified, each key-value pair in the Data field of the referenced Secret will be projected into the volume as a file whose name is the key and content is the value. If specified, the listed keys will be projected into the specified paths, and unlisted keys will not be present. If a key is specified which is not present in the Secret, the volume setup will error unless it is marked optional. Paths must be relative and may not contain the '..' path or start with '..'.</td>
        <td>false</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexsecretitemsindex">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].secret.items[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindexsecret">↩ Parent</a></sup></sup>
</h3>



Maps a string key to a path within a volume.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key to project.</td>
        <td>true</td>
      </tr><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>The relative path of the file to map the key to. May not be an absolute path. May not contain the path element '..'. May not start with the string '..'.</td>
        <td>true</td>
      </tr><tr>
        <td><b>mode</b></td>
        <td>integer</td>
        <td>Optional: mode bits used to set permissions on this file. Must be an octal value between 0000 and 0777 or a decimal value between 0 and 511. YAML accepts both octal and decimal values, JSON requires decimal values for mode bits. If not specified, the volume defaultMode will be used. This might be in conflict with other options that affect the file mode, like fsGroup, and the result can be other mode bits set.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigfilesindexserviceaccounttoken">
  PostgresCluster.spec.userInterface.pgAdmin.config.files[index].serviceAccountToken
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfigfilesindex">↩ Parent</a></sup></sup>
</h3>



information about the serviceAccountToken data to project

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>path</b></td>
        <td>string</td>
        <td>Path is the path relative to the mount point of the file to project the token into.</td>
        <td>true</td>
      </tr><tr>
        <td><b>audience</b></td>
        <td>string</td>
        <td>Audience is the intended audience of the token. A recipient of a token must identify itself with an identifier specified in the audience of the token, and otherwise should reject the token. The audience defaults to the identifier of the apiserver.</td>
        <td>false</td>
      </tr><tr>
        <td><b>expirationSeconds</b></td>
        <td>integer</td>
        <td>ExpirationSeconds is the requested duration of validity of the service account token. As the token approaches expiration, the kubelet volume plugin will proactively rotate the service account token. The kubelet will start trying to rotate the token if the token is older than 80 percent of its time to live or if the token is older than 24 hours.Defaults to 1 hour and must be at least 10 minutes.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminconfigldapbindpassword">
  PostgresCluster.spec.userInterface.pgAdmin.config.ldapBindPassword
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadminconfig">↩ Parent</a></sup></sup>
</h3>



A Secret containing the value for the LDAP_BIND_PASSWORD setting. More info: https://www.pgadmin.org/docs/pgadmin4/latest/ldap.html

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>The key of the secret to select from.  Must be a valid secret key.</td>
        <td>true</td>
      </tr><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>Name of the referent. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/names/#names</td>
        <td>false</td>
      </tr><tr>
        <td><b>optional</b></td>
        <td>boolean</td>
        <td>Specify whether the Secret or its key must be defined</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminmetadata">
  PostgresCluster.spec.userInterface.pgAdmin.metadata
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



Metadata contains metadata for PostgresCluster resources

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>annotations</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b>labels</b></td>
        <td>map[string]string</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminresources">
  PostgresCluster.spec.userInterface.pgAdmin.resources
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



Compute resources of a pgAdmin container. Changing this value causes pgAdmin to restart. More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>limits</b></td>
        <td>map[string]int or string</td>
        <td>Limits describes the maximum amount of compute resources allowed. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr><tr>
        <td><b>requests</b></td>
        <td>map[string]int or string</td>
        <td>Requests describes the minimum amount of compute resources required. If Requests is omitted for a container, it defaults to Limits if that is explicitly specified, otherwise to an implementation-defined value. More info: https://kubernetes.io/docs/concepts/configuration/manage-compute-resources-container/</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadminservice">
  PostgresCluster.spec.userInterface.pgAdmin.service
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



Specification of the service that exposes pgAdmin.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>More info: https://kubernetes.io/docs/concepts/services-networking/service/#publishing-services-service-types</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmintolerationsindex">
  PostgresCluster.spec.userInterface.pgAdmin.tolerations[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



The pod this Toleration is attached to tolerates any taint that matches the triple <key,value,effect> using the matching operator <operator>.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>effect</b></td>
        <td>string</td>
        <td>Effect indicates the taint effect to match. Empty means match all taint effects. When specified, allowed values are NoSchedule, PreferNoSchedule and NoExecute.</td>
        <td>false</td>
      </tr><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>Key is the taint key that the toleration applies to. Empty means match all taint keys. If the key is empty, operator must be Exists; this combination means to match all values and all keys.</td>
        <td>false</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>Operator represents a key's relationship to the value. Valid operators are Exists and Equal. Defaults to Equal. Exists is equivalent to wildcard for value, so that a pod can tolerate all taints of a particular category.</td>
        <td>false</td>
      </tr><tr>
        <td><b>tolerationSeconds</b></td>
        <td>integer</td>
        <td>TolerationSeconds represents the period of time the toleration (which must be of effect NoExecute, otherwise this field is ignored) tolerates the taint. By default, it is not set, which means tolerate the taint forever (do not evict). Zero and negative values will be treated as 0 (evict immediately) by the system.</td>
        <td>false</td>
      </tr><tr>
        <td><b>value</b></td>
        <td>string</td>
        <td>Value is the taint value the toleration matches to. If the operator is Exists, the value should be empty, otherwise just a regular string.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindex">
  PostgresCluster.spec.userInterface.pgAdmin.topologySpreadConstraints[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmin">↩ Parent</a></sup></sup>
</h3>



TopologySpreadConstraint specifies how to spread matching pods among the given topology.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>maxSkew</b></td>
        <td>integer</td>
        <td>MaxSkew describes the degree to which pods may be unevenly distributed. When `whenUnsatisfiable=DoNotSchedule`, it is the maximum permitted difference between the number of matching pods in the target topology and the global minimum. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 1/1/0: | zone1 | zone2 | zone3 | |   P   |   P   |       | - if MaxSkew is 1, incoming pod can only be scheduled to zone3 to become 1/1/1; scheduling it onto zone1(zone2) would make the ActualSkew(2-0) on zone1(zone2) violate MaxSkew(1). - if MaxSkew is 2, incoming pod can be scheduled onto any zone. When `whenUnsatisfiable=ScheduleAnyway`, it is used to give higher precedence to topologies that satisfy it. It's a required field. Default value is 1 and 0 is not allowed.</td>
        <td>true</td>
      </tr><tr>
        <td><b>topologyKey</b></td>
        <td>string</td>
        <td>TopologyKey is the key of node labels. Nodes that have a label with this key and identical values are considered to be in the same topology. We consider each <key, value> as a "bucket", and try to put balanced number of pods into each bucket. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b>whenUnsatisfiable</b></td>
        <td>string</td>
        <td>WhenUnsatisfiable indicates how to deal with a pod if it doesn't satisfy the spread constraint. - DoNotSchedule (default) tells the scheduler not to schedule it. - ScheduleAnyway tells the scheduler to schedule the pod in any location, but giving higher precedence to topologies that would help reduce the skew. A constraint is considered "Unsatisfiable" for an incoming pod if and only if every possible node assigment for that pod would violate "MaxSkew" on some topology. For example, in a 3-zone cluster, MaxSkew is set to 1, and pods with the same labelSelector spread as 3/1/1: | zone1 | zone2 | zone3 | | P P P |   P   |   P   | If WhenUnsatisfiable is set to DoNotSchedule, incoming pod can only be scheduled to zone2(zone3) to become 3/2/1(3/1/2) as ActualSkew(2-1) on zone2(zone3) satisfies MaxSkew(1). In other words, the cluster can still be imbalanced, but scheduler won't make it *more* imbalanced. It's a required field.</td>
        <td>true</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindexlabelselector">labelSelector</a></b></td>
        <td>object</td>
        <td>LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindexlabelselector">
  PostgresCluster.spec.userInterface.pgAdmin.topologySpreadConstraints[index].labelSelector
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindex">↩ Parent</a></sup></sup>
</h3>



LabelSelector is used to find matching pods. Pods that match this label selector are counted to determine the number of pods in their corresponding topology domain.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindexlabelselectormatchexpressionsindex">matchExpressions</a></b></td>
        <td>[]object</td>
        <td>matchExpressions is a list of label selector requirements. The requirements are ANDed.</td>
        <td>false</td>
      </tr><tr>
        <td><b>matchLabels</b></td>
        <td>map[string]string</td>
        <td>matchLabels is a map of {key,value} pairs. A single {key,value} in the matchLabels map is equivalent to an element of matchExpressions, whose key field is "key", the operator is "In", and the values array contains only "value". The requirements are ANDed.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindexlabelselectormatchexpressionsindex">
  PostgresCluster.spec.userInterface.pgAdmin.topologySpreadConstraints[index].labelSelector.matchExpressions[index]
  <sup><sup><a href="#postgresclusterspecuserinterfacepgadmintopologyspreadconstraintsindexlabelselector">↩ Parent</a></sup></sup>
</h3>



A label selector requirement is a selector that contains values, a key, and an operator that relates the key and values.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>key</b></td>
        <td>string</td>
        <td>key is the label key that the selector applies to.</td>
        <td>true</td>
      </tr><tr>
        <td><b>operator</b></td>
        <td>string</td>
        <td>operator represents a key's relationship to a set of values. Valid operators are In, NotIn, Exists and DoesNotExist.</td>
        <td>true</td>
      </tr><tr>
        <td><b>values</b></td>
        <td>[]string</td>
        <td>values is an array of string values. If the operator is In or NotIn, the values array must be non-empty. If the operator is Exists or DoesNotExist, the values array must be empty. This array is replaced during a strategic merge patch.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecusersindex">
  PostgresCluster.spec.users[index]
  <sup><sup><a href="#postgresclusterspec">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>The name of this PostgreSQL user. The value may contain only lowercase letters, numbers, and hyphen so that it fits into Kubernetes metadata.</td>
        <td>true</td>
      </tr><tr>
        <td><b>databases</b></td>
        <td>[]string</td>
        <td>Databases to which this user can connect and create objects. Removing a database from this list does NOT revoke access. This field is ignored for the "postgres" user.</td>
        <td>false</td>
      </tr><tr>
        <td><b>options</b></td>
        <td>string</td>
        <td>ALTER ROLE options except for PASSWORD. This field is ignored for the "postgres" user. More info: https://www.postgresql.org/docs/current/role-attributes.html</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterspecusersindexpassword">password</a></b></td>
        <td>object</td>
        <td>Properties of the password generated for this user.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterspecusersindexpassword">
  PostgresCluster.spec.users[index].password
  <sup><sup><a href="#postgresclusterspecusersindex">↩ Parent</a></sup></sup>
</h3>



Properties of the password generated for this user.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>type</b></td>
        <td>enum</td>
        <td>Type of password to generate. Defaults to ASCII. Valid options are ASCII and AlphaNumeric. "ASCII" passwords contain letters, numbers, and symbols from the US-ASCII character set. "AlphaNumeric" passwords contain letters and numbers from the US-ASCII character set.</td>
        <td>true</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatus">
  PostgresCluster.status
  <sup><sup><a href="#postgrescluster">↩ Parent</a></sup></sup>
</h3>



PostgresClusterStatus defines the observed state of PostgresCluster

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterstatusconditionsindex">conditions</a></b></td>
        <td>[]object</td>
        <td>conditions represent the observations of postgrescluster's current state. Known .status.conditions.type are: "PersistentVolumeResizing", "ProxyAvailable"</td>
        <td>false</td>
      </tr><tr>
        <td><b>databaseInitSQL</b></td>
        <td>string</td>
        <td>DatabaseInitSQL state of custom database initialization in the cluster</td>
        <td>false</td>
      </tr><tr>
        <td><b>databaseRevision</b></td>
        <td>string</td>
        <td>Identifies the databases that have been installed into PostgreSQL.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatusinstancesindex">instances</a></b></td>
        <td>[]object</td>
        <td>Current state of PostgreSQL instances.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatusmonitoring">monitoring</a></b></td>
        <td>object</td>
        <td>Current state of PostgreSQL cluster monitoring tool configuration</td>
        <td>false</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>observedGeneration represents the .metadata.generation on which the status was based.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatuspatroni">patroni</a></b></td>
        <td>object</td>
        <td></td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatuspgbackrest">pgbackrest</a></b></td>
        <td>object</td>
        <td>Status information for pgBackRest</td>
        <td>false</td>
      </tr><tr>
        <td><b>postgresVersion</b></td>
        <td>integer</td>
        <td>Stores the current PostgreSQL major version following a successful major PostgreSQL upgrade.</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatusproxy">proxy</a></b></td>
        <td>object</td>
        <td>Current state of the PostgreSQL proxy.</td>
        <td>false</td>
      </tr><tr>
        <td><b>startupInstance</b></td>
        <td>string</td>
        <td>The instance that should be started first when bootstrapping and/or starting a PostgresCluster.</td>
        <td>false</td>
      </tr><tr>
        <td><b>startupInstanceSet</b></td>
        <td>string</td>
        <td>The instance set associated with the startupInstance</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatususerinterface">userInterface</a></b></td>
        <td>object</td>
        <td>Current state of the PostgreSQL user interface.</td>
        <td>false</td>
      </tr><tr>
        <td><b>usersRevision</b></td>
        <td>string</td>
        <td>Identifies the users that have been installed into PostgreSQL.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatusconditionsindex">
  PostgresCluster.status.conditions[index]
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>



Condition contains details for one aspect of the current state of this API Resource.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>lastTransitionTime</b></td>
        <td>string</td>
        <td>lastTransitionTime is the last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.</td>
        <td>true</td>
      </tr><tr>
        <td><b>message</b></td>
        <td>string</td>
        <td>message is a human readable message indicating details about the transition. This may be an empty string.</td>
        <td>true</td>
      </tr><tr>
        <td><b>reason</b></td>
        <td>string</td>
        <td>reason contains a programmatic identifier indicating the reason for the condition's last transition. Producers of specific condition types may define expected values and meanings for this field, and whether the values are considered a guaranteed API. The value should be a CamelCase string. This field may not be empty.</td>
        <td>true</td>
      </tr><tr>
        <td><b>status</b></td>
        <td>enum</td>
        <td>status of the condition, one of True, False, Unknown.</td>
        <td>true</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>type of condition in CamelCase.</td>
        <td>true</td>
      </tr><tr>
        <td><b>observedGeneration</b></td>
        <td>integer</td>
        <td>observedGeneration represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.conditions[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatusinstancesindex">
  PostgresCluster.status.instances[index]
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td></td>
        <td>true</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>Total number of ready pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>Total number of pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>updatedReplicas</b></td>
        <td>integer</td>
        <td>Total number of pods that have the desired specification.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatusmonitoring">
  PostgresCluster.status.monitoring
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>



Current state of PostgreSQL cluster monitoring tool configuration

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>exporterConfiguration</b></td>
        <td>string</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspatroni">
  PostgresCluster.status.patroni
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>switchover</b></td>
        <td>string</td>
        <td>Tracks the execution of the switchover requests.</td>
        <td>false</td>
      </tr><tr>
        <td><b>systemIdentifier</b></td>
        <td>string</td>
        <td>The PostgreSQL system identifier reported by Patroni.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspgbackrest">
  PostgresCluster.status.pgbackrest
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>



Status information for pgBackRest

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterstatuspgbackrestmanualbackup">manualBackup</a></b></td>
        <td>object</td>
        <td>Status information for manual backups</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatuspgbackrestrepohost">repoHost</a></b></td>
        <td>object</td>
        <td>Status information for the pgBackRest dedicated repository host</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatuspgbackrestreposindex">repos</a></b></td>
        <td>[]object</td>
        <td>Status information for pgBackRest repositories</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatuspgbackrestrestore">restore</a></b></td>
        <td>object</td>
        <td>Status information for in-place restores</td>
        <td>false</td>
      </tr><tr>
        <td><b><a href="#postgresclusterstatuspgbackrestscheduledbackupsindex">scheduledBackups</a></b></td>
        <td>[]object</td>
        <td>Status information for scheduled backups</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspgbackrestmanualbackup">
  PostgresCluster.status.pgbackrest.manualBackup
  <sup><sup><a href="#postgresclusterstatuspgbackrest">↩ Parent</a></sup></sup>
</h3>



Status information for manual backups

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>finished</b></td>
        <td>boolean</td>
        <td>Specifies whether or not the Job is finished executing (does not indicate success or failure).</td>
        <td>true</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>A unique identifier for the manual backup as provided using the "pgbackrest-backup" annotation when initiating a backup.</td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>integer</td>
        <td>The number of actively running manual backup Pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>completionTime</b></td>
        <td>string</td>
        <td>Represents the time the manual backup Job was determined by the Job controller to be completed.  This field is only set if the backup completed successfully. Additionally, it is represented in RFC3339 form and is in UTC.</td>
        <td>false</td>
      </tr><tr>
        <td><b>failed</b></td>
        <td>integer</td>
        <td>The number of Pods for the manual backup Job that reached the "Failed" phase.</td>
        <td>false</td>
      </tr><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>Represents the time the manual backup Job was acknowledged by the Job controller. It is represented in RFC3339 form and is in UTC.</td>
        <td>false</td>
      </tr><tr>
        <td><b>succeeded</b></td>
        <td>integer</td>
        <td>The number of Pods for the manual backup Job that reached the "Succeeded" phase.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspgbackrestrepohost">
  PostgresCluster.status.pgbackrest.repoHost
  <sup><sup><a href="#postgresclusterstatuspgbackrest">↩ Parent</a></sup></sup>
</h3>



Status information for the pgBackRest dedicated repository host

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>apiVersion</b></td>
        <td>string</td>
        <td>APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#resources</td>
        <td>false</td>
      </tr><tr>
        <td><b>kind</b></td>
        <td>string</td>
        <td>Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds</td>
        <td>false</td>
      </tr><tr>
        <td><b>ready</b></td>
        <td>boolean</td>
        <td>Whether or not the pgBackRest repository host is ready for use</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspgbackrestreposindex">
  PostgresCluster.status.pgbackrest.repos[index]
  <sup><sup><a href="#postgresclusterstatuspgbackrest">↩ Parent</a></sup></sup>
</h3>



RepoStatus the status of a pgBackRest repository

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>name</b></td>
        <td>string</td>
        <td>The name of the pgBackRest repository</td>
        <td>true</td>
      </tr><tr>
        <td><b>bound</b></td>
        <td>boolean</td>
        <td>Whether or not the pgBackRest repository PersistentVolumeClaim is bound to a volume</td>
        <td>false</td>
      </tr><tr>
        <td><b>replicaCreateBackupComplete</b></td>
        <td>boolean</td>
        <td>ReplicaCreateBackupReady indicates whether a backup exists in the repository as needed to bootstrap replicas.</td>
        <td>false</td>
      </tr><tr>
        <td><b>repoOptionsHash</b></td>
        <td>string</td>
        <td>A hash of the required fields in the spec for defining an Azure, GCS or S3 repository, Utilizd to detect changes to these fields and then execute pgBackRest stanza-create commands accordingly.</td>
        <td>false</td>
      </tr><tr>
        <td><b>stanzaCreated</b></td>
        <td>boolean</td>
        <td>Specifies whether or not a stanza has been successfully created for the repository</td>
        <td>false</td>
      </tr><tr>
        <td><b>volume</b></td>
        <td>string</td>
        <td>The name of the volume the containing the pgBackRest repository</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspgbackrestrestore">
  PostgresCluster.status.pgbackrest.restore
  <sup><sup><a href="#postgresclusterstatuspgbackrest">↩ Parent</a></sup></sup>
</h3>



Status information for in-place restores

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>finished</b></td>
        <td>boolean</td>
        <td>Specifies whether or not the Job is finished executing (does not indicate success or failure).</td>
        <td>true</td>
      </tr><tr>
        <td><b>id</b></td>
        <td>string</td>
        <td>A unique identifier for the manual backup as provided using the "pgbackrest-backup" annotation when initiating a backup.</td>
        <td>true</td>
      </tr><tr>
        <td><b>active</b></td>
        <td>integer</td>
        <td>The number of actively running manual backup Pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>completionTime</b></td>
        <td>string</td>
        <td>Represents the time the manual backup Job was determined by the Job controller to be completed.  This field is only set if the backup completed successfully. Additionally, it is represented in RFC3339 form and is in UTC.</td>
        <td>false</td>
      </tr><tr>
        <td><b>failed</b></td>
        <td>integer</td>
        <td>The number of Pods for the manual backup Job that reached the "Failed" phase.</td>
        <td>false</td>
      </tr><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>Represents the time the manual backup Job was acknowledged by the Job controller. It is represented in RFC3339 form and is in UTC.</td>
        <td>false</td>
      </tr><tr>
        <td><b>succeeded</b></td>
        <td>integer</td>
        <td>The number of Pods for the manual backup Job that reached the "Succeeded" phase.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatuspgbackrestscheduledbackupsindex">
  PostgresCluster.status.pgbackrest.scheduledBackups[index]
  <sup><sup><a href="#postgresclusterstatuspgbackrest">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>active</b></td>
        <td>integer</td>
        <td>The number of actively running manual backup Pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>completionTime</b></td>
        <td>string</td>
        <td>Represents the time the manual backup Job was determined by the Job controller to be completed.  This field is only set if the backup completed successfully. Additionally, it is represented in RFC3339 form and is in UTC.</td>
        <td>false</td>
      </tr><tr>
        <td><b>cronJobName</b></td>
        <td>string</td>
        <td>The name of the associated pgBackRest scheduled backup CronJob</td>
        <td>false</td>
      </tr><tr>
        <td><b>failed</b></td>
        <td>integer</td>
        <td>The number of Pods for the manual backup Job that reached the "Failed" phase.</td>
        <td>false</td>
      </tr><tr>
        <td><b>repo</b></td>
        <td>string</td>
        <td>The name of the associated pgBackRest repository</td>
        <td>false</td>
      </tr><tr>
        <td><b>startTime</b></td>
        <td>string</td>
        <td>Represents the time the manual backup Job was acknowledged by the Job controller. It is represented in RFC3339 form and is in UTC.</td>
        <td>false</td>
      </tr><tr>
        <td><b>succeeded</b></td>
        <td>integer</td>
        <td>The number of Pods for the manual backup Job that reached the "Succeeded" phase.</td>
        <td>false</td>
      </tr><tr>
        <td><b>type</b></td>
        <td>string</td>
        <td>The pgBackRest backup type for this Job</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatusproxy">
  PostgresCluster.status.proxy
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>



Current state of the PostgreSQL proxy.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterstatusproxypgbouncer">pgBouncer</a></b></td>
        <td>object</td>
        <td></td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatusproxypgbouncer">
  PostgresCluster.status.proxy.pgBouncer
  <sup><sup><a href="#postgresclusterstatusproxy">↩ Parent</a></sup></sup>
</h3>





<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>postgresRevision</b></td>
        <td>string</td>
        <td>Identifies the revision of PgBouncer assets that have been installed into PostgreSQL.</td>
        <td>false</td>
      </tr><tr>
        <td><b>readyReplicas</b></td>
        <td>integer</td>
        <td>Total number of ready pods.</td>
        <td>false</td>
      </tr><tr>
        <td><b>replicas</b></td>
        <td>integer</td>
        <td>Total number of non-terminated pods.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatususerinterface">
  PostgresCluster.status.userInterface
  <sup><sup><a href="#postgresclusterstatus">↩ Parent</a></sup></sup>
</h3>



Current state of the PostgreSQL user interface.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b><a href="#postgresclusterstatususerinterfacepgadmin">pgAdmin</a></b></td>
        <td>object</td>
        <td>The state of the pgAdmin user interface.</td>
        <td>false</td>
      </tr></tbody>
</table>


<h3 id="postgresclusterstatususerinterfacepgadmin">
  PostgresCluster.status.userInterface.pgAdmin
  <sup><sup><a href="#postgresclusterstatususerinterface">↩ Parent</a></sup></sup>
</h3>



The state of the pgAdmin user interface.

<table>
    <thead>
        <tr>
            <th>Name</th>
            <th>Type</th>
            <th>Description</th>
            <th>Required</th>
        </tr>
    </thead>
    <tbody><tr>
        <td><b>usersRevision</b></td>
        <td>string</td>
        <td>Hash that indicates which users have been installed into pgAdmin.</td>
        <td>false</td>
      </tr></tbody>
</table>
