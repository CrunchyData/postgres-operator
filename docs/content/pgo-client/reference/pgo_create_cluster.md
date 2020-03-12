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
      --custom-config string                  The name of a configMap that holds custom PostgreSQL configuration files used to override defaults.
  -d, --database string                       If specified, sets the name of the initial database that is created for the user. Defaults to the value set in the PostgreSQL Operator configuration, or if that is not present, the name of the cluster
      --disable-autofail                      Disables autofail capabitilies in the cluster following cluster initialization.
  -h, --help                                  help for cluster
  -l, --labels string                         The labels to apply to this cluster.
      --metrics                               Adds the crunchy-collect container to the database pod.
      --node-label string                     The node label (key=value) to use in placing the primary database. If not set, any node is used.
  -w, --password string                       The password to use for initial database user.
      --password-length int                   If no password is supplied, sets the length of the automatically generated password. Defaults to the value set on the server.
      --pgbackrest-pvc-size string            The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "local" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --pgbackrest-s3-bucket string           The AWS S3 bucket that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-endpoint string         The AWS S3 endpoint that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-key string              The AWS S3 key that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-key-secret string       The AWS S3 key secret that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-s3-region string           The AWS S3 region that should be utilized for the cluster when the "s3" storage type is enabled for pgBackRest.
      --pgbackrest-storage-type string        The type of storage to use with pgBackRest. Either "local", "s3" or both, comma separated. (default "local")
      --pgbadger                              Adds the crunchy-pgbadger container to the database pod.
      --pgbouncer                             Adds a crunchy-pgbouncer deployment to the cluster.
      --pod-anti-affinity string              Specifies the type of anti-affinity that should be utilized when applying  default pod anti-affinity rules to PG clusters (default "preferred")
      --pod-anti-affinity-pgbackrest string   Set the Pod anti-affinity rules specifically for the pgBackRest repository. Defaults to the default cluster pod anti-affinity (i.e. "preferred"), or the value set by --pod-anti-affinity
      --pod-anti-affinity-pgbouncer string    Set the Pod anti-affinity rules specifically for the pgBouncer Pods. Defaults to the default cluster pod anti-affinity (i.e. "preferred"), or the value set by --pod-anti-affinity
  -z, --policies string                       The policies to apply when creating a cluster, comma separated.
      --pvc-size string                       The size of the PVC capacity for primary and replica PostgreSQL instances. Overrides the value set in the storage class. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --replica-count int                     The number of replicas to create as part of the cluster.
      --replica-storage-config string         The name of a Storage config in pgo.yaml to use for the cluster replica storage.
  -r, --resources-config string               The name of a container resource configuration in pgo.yaml that holds CPU and memory requests and limits.
  -s, --secret-from string                    The cluster name to use when restoring secrets.
      --service-type string                   The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.
      --show-system-accounts                  Include the system accounts in the results.
      --storage-config string                 The name of a Storage config in pgo.yaml to use for the cluster storage.
      --sync-replication                      Enables synchronous replication for the cluster.
      --tablespace strings                    Create a PostgreSQL tablespace on the cluster, e.g. "name=ts1:storageconfig=nfsstorage". The format is a key/value map that is delimited by "=" and separated by ":". The following parameters are available:

                                              - name (required): the name of the PostgreSQL tablespace
                                              - storageconfig (required): the storage configuration to use, as specified in the list available in the "pgo-config" ConfigMap (aka "pgo.yaml")
                                              - pvcsize: the size of the PVC capacity, which overrides the value set in the specified storageconfig. Follows the Kubernetes quantity format.

                                              For example, to create a tablespace with the NFS storage configuration with a PVC of size 10GiB:

                                              --tablespace=name=ts1:storageconfig=nfsstorage:pvcsize=10Gi
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

###### Auto generated by spf13/cobra on 12-Mar-2020
