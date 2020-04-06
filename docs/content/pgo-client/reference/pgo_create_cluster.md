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
      --ccp-image string                      The CCPImage name to use for cluster creation. If specified, overrides the value crunchy-postgres.
  -c, --ccp-image-tag string                  The CCPImageTag to use for cluster creation. If specified, overrides the pgo.yaml setting.
      --cpu string                            Set the number of millicores to request for the CPU, e.g. "100m" or "0.1". Overrides the value in "resources-config"
      --custom-config string                  The name of a configMap that holds custom PostgreSQL configuration files used to override defaults.
  -d, --database string                       If specified, sets the name of the initial database that is created for the user. Defaults to the value set in the PostgreSQL Operator configuration, or if that is not present, the name of the cluster
      --disable-autofail                      Disables autofail capabitilies in the cluster following cluster initialization.
  -h, --help                                  help for cluster
  -l, --labels string                         The labels to apply to this cluster.
      --memory string                         Set the amount of RAM to request, e.g. 1GiB. Overrides the value in "resources-config"
      --metrics                               Adds the crunchy-collect container to the database pod.
      --node-label string                     The node label (key=value) to use in placing the primary database. If not set, any node is used.
      --password string                       The password to use for standard user account created during cluster initialization.
      --password-length int                   If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.
      --password-replication string           The password to use for the PostgreSQL replication user.
      --password-superuser string             The password to use for the PostgreSQL superuser.
      --pgbackrest-cpu string                 Set the number of millicores to request for CPU for the pgBackRest repository. Defaults to being unset.
      --pgbackrest-memory string              Set the amount of Memory to request for the pgBackRest repository. Defaults to server value (48Mi).
      --pgbackrest-pvc-size string            The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "local" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --pgbackrest-repo-path string           The pgBackRest repository path that should be utilized instead of the default. Required for standby
                                              clusters to define the location of an existing pgBackRest repository.
      --pgbackrest-s3-bucket string           The AWS S3 bucket that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-ca-secret string        If used, specifies a Kubernetes secret that uses a different CA certificate for S3 or a S3-like storage interface. Must contain a key with the value "aws-s3-ca.crt"
      --pgbackrest-s3-endpoint string         The AWS S3 endpoint that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-key string              The AWS S3 key that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-key-secret string       The AWS S3 key secret that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-region string           The AWS S3 region that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-storage-config string      The name of the storage config in pgo.yaml to use for the pgBackRest local repository.
      --pgbackrest-storage-type string        The type of storage to use with pgBackRest. Either "local", "s3" or both, comma separated. (default "local")
      --pgbadger                              Adds the crunchy-pgbadger container to the database pod.
      --pgbouncer                             Adds a crunchy-pgbouncer deployment to the cluster.
      --pgbouncer-cpu string                  Set the number of millicores to request for CPU for pgBouncer. Defaults to being unset.
      --pgbouncer-memory string               Set the amount of Memory to request for pgBouncer. Defaults to server value (24Mi).
      --pod-anti-affinity string              Specifies the type of anti-affinity that should be utilized when applying  default pod anti-affinity rules to PG clusters (default "preferred")
      --pod-anti-affinity-pgbackrest string   Set the Pod anti-affinity rules specifically for the pgBackRest repository. Defaults to the default cluster pod anti-affinity (i.e. "preferred"), or the value set by --pod-anti-affinity
      --pod-anti-affinity-pgbouncer string    Set the Pod anti-affinity rules specifically for the pgBouncer Pods. Defaults to the default cluster pod anti-affinity (i.e. "preferred"), or the value set by --pod-anti-affinity
  -z, --policies string                       The policies to apply when creating a cluster, comma separated.
      --pvc-size string                       The size of the PVC capacity for primary and replica PostgreSQL instances. Overrides the value set in the storage class. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --replica-count int                     The number of replicas to create as part of the cluster.
      --replica-storage-config string         The name of a Storage config in pgo.yaml to use for the cluster replica storage.
  -r, --resources-config string               The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits.
  -s, --secret-from string                    The cluster name to use when restoring secrets.
      --server-ca-secret string               The name of the secret that contains the certficate authority (CA) to use for enabling the PostgreSQL cluster to accept TLS connections. Must be used with "server-tls-secret"
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
  -u, --username string                       The username to use for creating the PostgreSQL user with standard permissions. Defaults to the value in the PostgreSQL Operator configuration.
```

### Options inherited from parent commands

```
      --apiserver-url string     The URL for the PostgreSQL Operator apiserver that will process the request from the pgo client.
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

###### Auto generated by spf13/cobra on 6-Apr-2020
