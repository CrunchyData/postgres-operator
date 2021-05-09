---
title: TODO - Content that needs a home
draft: false
weight: 200
---

This content needs to find a permanent home.

## Scale a Postgres Cluster

There are two ways to scale a cluster:

### Method 1

In the `examples/postgrescluster/postgrescluster.yaml` file, add `replicas: 2` to one of the instances entry.

Then run:

```
kubectl apply -k examples/postgrescluster
```

### Method 2

In the `examples/postgrescluster/postgrescluster.yaml` file, add a new array item name `instance2`, e.g.:

```
spec:
  instances:
    - name: instance1
    - name: instance2
```

## Add pgBackRest Backup Schedules

Scheduled pgBackRest `full`, `differential` and `incremental` backups can be added for each defined pgBackRest
repo. This is done by adding, under the `repos` section, a `schedules` section with the designated CronJob
schedule defined for each backup type desired. For example, for `repo1`, we defined the following:
```
  archive:
    pgbackrest:
      repoHost:
        dedicated: {}
        image: gcr.io/crunchy-dev-test/crunchy-pgbackrest:centos8-12.6-multi.dev2
      repos:
      - name: repo1
        schedules:
          full: "* */1 * * *"
          differential: "*/10 * * * *"
          incremental: "*/5 * * * *"
```
For any type not listed, no CronJob will be created. For more information on CronJobs and the necessary scheduling
syntax, please see https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs/#cron-schedule-syntax.

## Add Custom Certificates to a Postgres Cluster

Create a Secret containing your TLS certificate, TLS key and the Certificate Authority certificate such that
the data of the secret in the `postgrescluster`'s namespace that contains the three values, similar to

```
data:
  ca.crt: <value>
  tls.crt: <value>
  tls.key: <value>
```

Then, in `examples/postgrescluster/postgrescluster.yaml`, add the following:

```
spec:
  customTLSSecret:
    name: customcert
```
where 'customcert' is the name of your created secret. Your cluster will now use the provided certificate in place of
a dynamically generated one. Please note, if `CustomTLSSecret` is provided, `CustomReplicationClientTLSSecret` MUST
be provided and the `ca.crt` provided must be the same in both.

In cases where the key names cannot be controlled, the item key and path values can be specified explicitly, as shown
below:
```
spec:
  customTLSSecret:
    name: customcert
    items:
      - key: <tls.crt key>
        path: tls.crt
      - key: <tls.key key>
        path: tls.key
      - key: <ca.crt key>
        path: ca.crt
```
The Common Name setting will be expected to include the primary service name. For a `postgrescluster` named 'hippo', the
expected primary service name will be `hippo-primary`.

## Add Replication Client Certificates to a Postgres Cluster

Similar to the above, to provide your own TLS client certificates for use by the replication system account,
you will need to create a Secret containing your TLS certificate, TLS key and the Certificate Authority certificate
such that the data of the secret in the `postgrescluster`'s namespace that contains the three values, similar to

```
data:
  ca.crt: <value>
  tls.crt: <value>
  tls.key: <value>
```

Then, in `examples/postgrescluster/postgrescluster.yaml`, add the following:

```
spec:
  customReplicationTLSSecret:
    name: customReplicationCert
```
where 'customReplicationCert' is the name of your created secret. Your cluster will now use the provided certificate in place of
a dynamically generated one. Please note, if `CustomReplicationClientTLSSecret` is provided, `CustomTLSSecret`
MUST be provided and the `ca.crt` provided must be the same in both.

In cases where the key names cannot be controlled, the item key and path values can be specified explicitly, as shown
below:
```
spec:
  customReplicationTLSSecret:
    name: customReplicationCert
    items:
      - key: <tls.crt key>
        path: tls.crt
      - key: <tls.key key>
        path: tls.key
      - key: <ca.crt key>
        path: ca.crt
```

The Common Name setting will be expected to include the replication user name, `_crunchyrepl`.

For more information regarding secret projections, please see
https://k8s.io/docs/concepts/configuration/secret/#projection-of-secret-keys-to-specific-paths

## Delete a Postgres Cluster

```
kubectl delete -k examples/postgrescluster
```
