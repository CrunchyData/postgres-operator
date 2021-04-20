---
title: "pgo update cluster"
---
## pgo update cluster

Update a PostgreSQL cluster

### Synopsis

Update a PostgreSQL cluster. For example:

    pgo update cluster mycluster --disable-autofail
    pgo update cluster mycluster myothercluster --disable-autofail
    pgo update cluster --selector=name=mycluster --disable-autofail
    pgo update cluster --all --enable-autofail

```
pgo update cluster [flags]
```

### Options

```
      --all                              all resources.
      --annotation strings               Add an Annotation to all of the managed deployments (PostgreSQL, pgBackRest, pgBouncer)
                                         The format to add an annotation is "name=value"
                                         The format to remove an annotation is "name-"
                                         
                                         For example, to add two annotations: "--annotation=hippo=awesome,elephant=cool"
      --annotation-pgbackrest strings    Add an Annotation specifically to pgBackRest deployments
                                         The format to add an annotation is "name=value"
                                         The format to remove an annotation is "name-"
      --annotation-pgbouncer strings     Add an Annotation specifically to pgBouncer deployments
                                         The format to add an annotation is "name=value"
                                         The format to remove an annotation is "name-"
      --annotation-postgres strings      Add an Annotation specifically to PostgreSQL deploymentsThe format to add an annotation is "name=value"
                                         The format to remove an annotation is "name-"
      --cpu string                       Set the number of millicores to request for the CPU, e.g. "100m" or "0.1".
      --cpu-limit string                 Set the number of millicores to limit for the CPU, e.g. "100m" or "0.1".
      --disable-autofail                 Disables autofail capabitilies in the cluster.
      --disable-metrics                  Disable the metrics collection sidecar. May cause brief downtime.
      --disable-pgbadger                 Disable the pgBadger sidecar. May cause brief downtime.
      --disable-server-tls               Remove TLS from the cluster.
      --disable-tls-only                 Remove TLS enforcement for the cluster.
      --enable-autofail                  Enables autofail capabitilies in the cluster.
      --enable-metrics                   Enable the metrics collection sidecar. May cause brief downtime.
      --enable-pgbadger                  Enable the pgBadger sidecar. May cause brief downtime.
      --enable-standby                   Enables standby mode in the cluster(s) specified.
      --enable-tls-only                  Enforce TLS on the cluster.
      --exporter-cpu string              Set the number of millicores to request for CPU for the Crunchy Postgres Exporter sidecar container, e.g. "100m" or "0.1".
      --exporter-cpu-limit string        Set the number of millicores to limit for CPU for the Crunchy Postgres Exporter sidecar container, e.g. "100m" or "0.1".
      --exporter-memory string           Set the amount of memory to request for the Crunchy Postgres Exporter sidecar container.
      --exporter-memory-limit string     Set the amount of memory to limit for the Crunchy Postgres Exporter sidecar container.
      --exporter-rotate-password         Used to rotate the password for the metrics collection agent.
  -h, --help                             help for cluster
      --memory string                    Set the amount of RAM to request, e.g. 1GiB.
      --memory-limit string              Set the amount of RAM to limit, e.g. 1GiB.
      --no-prompt                        No command line confirmation.
      --pgbackrest-cpu string            Set the number of millicores to request for CPU for the pgBackRest repository.
      --pgbackrest-cpu-limit string      Set the number of millicores to limit for CPU for the pgBackRest repository.
      --pgbackrest-memory string         Set the amount of memory to request for the pgBackRest repository.
      --pgbackrest-memory-limit string   Set the amount of memory to limit for the pgBackRest repository.
      --pgbackrest-pvc-size string       The size of the PVC capacity for the pgBackRest repository. Overrides the value set in the storage class. This is ignored if the storage type of "posix" is not used. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --promote-standby                  Disables standby mode (if enabled) and promotes the cluster(s) specified.
      --pvc-size string                  The size of the PVC capacity for primary and replica PostgreSQL instances. Must follow the standard Kubernetes format, e.g. "10.1Gi"
      --replication-tls-secret string    The name of the secret that contains the TLS keypair to use for enabling certificate-based authentication between PostgreSQL instances, particularly for the purpose of replication. TLS must be enabled in the cluster.
  -s, --selector string                  The selector to use for cluster filtering.
      --server-ca-secret string          The name of the secret that contains the certficate authority (CA) to use for enabling the PostgreSQL cluster to accept TLS connections. Must be used with "server-tls-secret".
      --server-tls-secret string         The name of the secret that contains the TLS keypair to use for enabling the PostgreSQL cluster to accept TLS connections. Must be used with "server-ca-secret"
      --service-type string              The Service type to use for the PostgreSQL cluster. If not set, the pgo.yaml default will be used.
      --shutdown                         Shutdown the database cluster if it is currently running.
      --startup                          Restart the database cluster if it is currently shutdown.
      --tablespace strings               Add a PostgreSQL tablespace on the cluster, e.g. "name=ts1:storageconfig=nfsstorage". The format is a key/value map that is delimited by "=" and separated by ":". The following parameters are available:
                                         
                                         - name (required): the name of the PostgreSQL tablespace
                                         - storageconfig (required): the storage configuration to use, as specified in the list available in the "pgo-config" ConfigMap (aka "pgo.yaml")
                                         - pvcsize: the size of the PVC capacity, which overrides the value set in the specified storageconfig. Follows the Kubernetes quantity format.
                                         
                                         For example, to create a tablespace with the NFS storage configuration with a PVC of size 10GiB:
                                         
                                         --tablespace=name=ts1:storageconfig=nfsstorage:pvcsize=10Gi
      --toleration strings               Set Pod tolerations for each PostgreSQL instance in a cluster.
                                         The general format is "key=value:Effect"
                                         For example, to add an Exists and an Equals toleration: "--toleration=ssd:NoSchedule,zone=east:NoSchedule"
                                         A toleration can be removed by adding a "-" to the end, for example:
                                         --toleration=ssd:NoSchedule-
      --wal-pvc-size string              The size of the capacity for WAL storage, which overrides any value in the storage configuration.  Must follow the standard Kubernetes format, e.g. "10.1Gi".
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

* [pgo update](/pgo-client/reference/pgo_update/)	 - Update a pgouser, pgorole, or cluster

###### Auto generated by spf13/cobra on 19-Apr-2021
