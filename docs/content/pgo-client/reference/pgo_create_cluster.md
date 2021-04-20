---
title: "pgo create cluster"
---
## pgo create cluster

Create a PostgreSQL cluster

### Synopsis

Create a PostgreSQL cluster consisting of a primary and a number of replica backends. For example:

    pgo create cluster mycluster

```
pgo create cluster [flags]
```

### Options

```
      --annotation strings                    Add an Annotation to all of the managed deployments (PostgreSQL, pgBackRest, pgBouncer)
                                              The format to add an annotation is "name=value"
                                              The format to remove an annotation is "name-"
                                              
                                              For example, to add two annotations: "--annotation=hippo=awesome,elephant=cool"
      --annotation-pgbackrest strings         Add an Annotation specifically to pgBackRest deployments
                                              The format to add an annotation is "name=value"
                                              The format to remove an annotation is "name-"
      --annotation-pgbouncer strings          Add an Annotation specifically to pgBouncer deployments
                                              The format to add an annotation is "name=value"
                                              The format to remove an annotation is "name-"
      --annotation-postgres strings           Add an Annotation specifically to PostgreSQL deployments
                                              The format to add an annotation is "name=value"
                                              The format to remove an annotation is "name-"
      --ccp-image string                      The CCPImage name to use for cluster creation. If specified, overrides the value crunchy-postgres.
      --ccp-image-prefix string               The CCPImagePrefix to use for cluster creation. If specified, overrides the global configuration.
  -c, --ccp-image-tag string                  The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.
      --cpu string                            Set the number of millicores to request for the CPU, e.g. "100m" or "0.1".
      --cpu-limit string                      Set the number of millicores to limit for the CPU, e.g. "100m" or "0.1".
      --custom-config string                  The name of a configMap that holds custom PostgreSQL configuration files used to override defaults.
  -d, --database string                       If specified, sets the name of the initial database that is created for the user. Defaults to the value set in the PostgreSQL Operator configuration, or if that is not present, the name of the cluster
      --disable-autofail                      Disables autofail capabitilies in the cluster following cluster initialization.
      --exporter-cpu string                   Set the number of millicores to request for CPU for the Crunchy Postgres Exporter sidecar container, e.g. "100m" or "0.1". Defaults to being unset.
      --exporter-cpu-limit string             Set the number of millicores to limit for CPU for the Crunchy Postgres Exporter sidecar container, e.g. "100m" or "0.1". Defaults to being unset.
      --exporter-memory string                Set the amount of memory to request for the Crunchy Postgres Exporter sidecar container. Defaults to server value (24Mi).
      --exporter-memory-limit string          Set the amount of memory to limit for the Crunchy Postgres Exporter sidecar container.
  -h, --help                                  help for cluster
      --label strings                         Add labels to apply to the PostgreSQL cluster, e.g. "key=value", "prefix/key=value". Can specify flag multiple times.
      --memory string                         Set the amount of RAM to request, e.g. 1GiB. Overrides the default server value.
      --memory-limit string                   Set the amount of RAM to limit, e.g. 1GiB.
      --metrics                               Adds the crunchy-postgres-exporter container to the database pod.
      --node-affinity-type string             Sets the type of node affinity to use. Can be either preferred (default) or required. Must be used with --node-label
      --node-label string                     The node label (key=value) to use in placing the primary database. If not set, any node is used.
      --password string                       The password to use for standard user account created during cluster initialization.
      --password-length int                   If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.
      --password-replication string           The password to use for the PostgreSQL replication user.
      --password-superuser string             The password to use for the PostgreSQL superuser.
      --password-type string                  The default Postgres password type to use for managed users. Either "scram-sha-256" or "md5". Defaults to "md5".
      --pgbackrest-cpu string                 Set the number of millicores to request for CPU for the pgBackRest repository.
      --pgbackrest-cpu-limit string           Set the number of millicores to limit for CPU for the pgBackRest repository.
      --pgbackrest-custom-config string       The name of a ConfigMap containing pgBackRest configuration files.
      --pgbackrest-gcs-bucket string          The GCS bucket that should be utilized for the cluster when the "gcs" storage type is enabled for pgBackRest.
      --pgbackrest-gcs-endpoint string        The GCS endpoint that should be utilized for the cluster when the "gcs" storage type is enabled for pgBackRest.
      --pgbackrest-gcs-key string             The GCS key that should be utilized for the cluster when the "gcs" storage type is enabled for pgBackRest. This must be a path to a file.
      --pgbackrest-gcs-key-type string        The GCS key type should be utilized for the cluster when the "gcs" storage type is enabled for pgBackRest. (default "service")
      --pgbackrest-memory string              Set the amount of memory to request for the pgBackRest repository. Defaults to server value (48Mi).
      --pgbackrest-memory-limit string        Set the amount of memory to limit for the pgBackRest repository.
      --pgbackrest-pvc-size string            The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "posix" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --pgbackrest-repo-path string           The pgBackRest repository path that should be utilized instead of the default. Required for standby
                                              clusters to define the location of an existing pgBackRest repository.
      --pgbackrest-s3-bucket string           The AWS S3 bucket that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-ca-secret string        If used, specifies a Kubernetes secret that uses a different CA certificate for S3 or a S3-like storage interface. Must contain a key with the value "aws-s3-ca.crt"
      --pgbackrest-s3-endpoint string         The AWS S3 endpoint that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-key string              The AWS S3 key that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-key-secret string       The AWS S3 key secret that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-region string           The AWS S3 region that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-uri-style string        Specifies whether "host" or "path" style URIs will be used when connecting to S3.
      --pgbackrest-s3-verify-tls              This sets if pgBackRest should verify the TLS certificate when connecting to S3. To disable, use "--pgbackrest-s3-verify-tls=false". (default true)
      --pgbackrest-storage-config string      The name of the storage config in pgo.yaml to use for the pgBackRest local repository.
      --pgbackrest-storage-type string        The type of storage to use with pgBackRest. Either "posix", "s3", "gcs", "posix,s3" or "posix,gcs". (default "posix")
      --pgbadger                              Adds the crunchy-pgbadger container to the database pod.
      --pgbouncer                             Adds a crunchy-pgbouncer deployment to the cluster.
      --pgbouncer-cpu string                  Set the number of millicores to request for CPU for pgBouncer. Defaults to being unset.
      --pgbouncer-cpu-limit string            Set the number of millicores to limit for CPU for pgBouncer. Defaults to being unset.
      --pgbouncer-memory string               Set the amount of memory to request for pgBouncer. Defaults to server value (24Mi).
      --pgbouncer-memory-limit string         Set the amount of memory to limit for pgBouncer.
      --pgbouncer-replicas int32              Set the total number of pgBouncer instances to deploy. If not set, defaults to 1.
      --pgbouncer-service-type string         The Service type to use for pgBouncer. Defaults to the Service type of the PostgreSQL cluster.
      --pgbouncer-tls-secret string           The name of the secret that contains the TLS keypair to use for enabling pgBouncer to accept TLS connections. Must also set server-tls-secret and server-ca-secret.
      --pgo-image-prefix string               The PGOImagePrefix to use for cluster creation. If specified, overrides the global configuration.
      --pod-anti-affinity string              Specifies the type of anti-affinity that should be utilized when applying  default pod anti-affinity rules to PG clusters (default "preferred")
      --pod-anti-affinity-pgbackrest string   Set the Pod anti-affinity rules specifically for the pgBackRest repository. Defaults to the default cluster pod anti-affinity (i.e. "preferred"), or the value set by --pod-anti-affinity
      --pod-anti-affinity-pgbouncer string    Set the Pod anti-affinity rules specifically for the pgBouncer Pods. Defaults to the default cluster pod anti-affinity (i.e. "preferred"), or the value set by --pod-anti-affinity
  -z, --policies string                       The policies to apply when creating a cluster, comma separated.
      --pvc-size string                       The size of the PVC capacity for primary and replica PostgreSQL instances. Overrides the value set in the storage class. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --replica-count int                     The number of replicas to create as part of the cluster.
      --replica-storage-config string         The name of a Storage config in pgo.yaml to use for the cluster replica storage.
      --replication-tls-secret string         The name of the secret that contains the TLS keypair to use for enabling certificate-based authentication between PostgreSQL instances, particularly for the purpose of replication. Must be used with "server-tls-secret" and "server-ca-secret".
      --restore-from string                   The name of cluster to restore from when bootstrapping a new cluster
      --restore-from-namespace string         The namespace for the cluster specified using --restore-from.  Defaults to the namespace of the cluster being created if not provided.
      --restore-opts string                   The options to pass into pgbackrest where performing a restore to bootrap the cluster. Only applicable when a "restore-from" value is specified
  -s, --secret-from string                    The cluster name to use when restoring secrets.
      --server-ca-secret string               The name of the secret that contains the certficate authority (CA) to use for enabling the PostgreSQL cluster to accept TLS connections. Must be used with "server-tls-secret".
      --server-tls-secret string              The name of the secret that contains the TLS keypair to use for enabling the PostgreSQL cluster to accept TLS connections. Must be used with "server-ca-secret"
      --service-type string                   The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.
      --show-system-accounts                  Include the system accounts in the results.
      --standby                               Creates a standby cluster that replicates from a pgBackRest repository in AWS S3.
      --storage-config string                 The name of a Storage config in pgo.yaml to use for the cluster storage.
      --sync-replication                      Enables synchronous replication for the cluster.
      --tablespace strings                    Create a PostgreSQL tablespace on the cluster, e.g. "name=ts1:storageconfig=nfsstorage". The format is a key/value map that is delimited by "=" and separated by ":". The following parameters are available:
                                              
                                              - name (required): the name of the PostgreSQL tablespace
                                              - storageconfig (required): the storage configuration to use, as specified in the list available in the "pgo-config" ConfigMap (aka "pgo.yaml")
                                              - pvcsize: the size of the PVC capacity, which overrides the value set in the specified storageconfig. Follows the Kubernetes quantity format.
                                              
                                              For example, to create a tablespace with the NFS storage configuration with a PVC of size 10GiB:
                                              
                                              --tablespace=name=ts1:storageconfig=nfsstorage:pvcsize=10Gi
      --tls-only                              If true, forces all PostgreSQL connections to be over TLS. Must also set "server-tls-secret" and "server-ca-secret"
      --toleration strings                    Set Pod tolerations for each PostgreSQL instance in a cluster.
                                              The general format is "key=value:Effect"
                                              For example, to add an Exists and an Equals toleration: "--toleration=ssd:NoSchedule,zone=east:NoSchedule"
  -u, --username string                       The username to use for creating the PostgreSQL user with standard permissions. Defaults to the value in the PostgreSQL Operator configuration.
      --wal-storage-config string             The name of a storage configuration in pgo.yaml to use for PostgreSQL's write-ahead log (WAL).
      --wal-storage-size string               The size of the capacity for WAL storage, which overrides any value in the storage configuration. Follows the Kubernetes quantity format.
```

### Options inherited from parent commands

```
      --apiserver-url string     The URL for the PostgreSQL Operator apiserver that will process the request from the pgo client. Note that the URL should **not** end in a '/'.
      --debug                    Enable additional output for debugging.
      --disable-tls              Disable TLS authentication to the Postgres Operator.
      --exclude-os-trust         Exclude CA certs from OS default trust store
  -n, --namespace string         The namespace to use for pgo requests.
      --pgo-ca-cert string       The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver.
      --pgo-client-cert string   The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.
      --pgo-client-key string    The Client Key file path for authenticating to the PostgreSQL Operator apiserver.
```

### SEE ALSO

* [pgo create](/pgo-client/reference/pgo_create/)	 - Create a Postgres Operator resource

###### Auto generated by spf13/cobra on 19-Apr-2021
