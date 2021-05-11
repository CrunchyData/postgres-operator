---
title: "Setup TLS"
draft: false
weight: 160
---

TLS allows secure TCP connections to PostgreSQL, and the PostgreSQL Operator makes it easy to enable this PostgreSQL feature. The TLS support in the PostgreSQL Operator does not make an opinion about your PKI, but rather loads in your TLS key pair that you wish to use for the PostgreSQL server as well as its corresponding certificate authority (CA) certificate. Both of these Secrets are
required to enable TLS support for your PostgreSQL cluster when using the PostgreSQL Operator, but it in turn allows seamless TLS support.

## Prerequisites

There are three items that are required to enable TLS in your PostgreSQL clusters:

- A CA certificate
- A TLS private key
- A TLS certificate

There are a variety of methods available to generate these items: in fact, Kubernetes comes with its own [certificate management system](https://kubernetes.io/docs/tasks/tls/managing-tls-in-a-cluster/)! It is up to you to decide how you want to manage this for your cluster. The PostgreSQL documentation also provides an example for how to [generate a TLS certificate](https://blog.crunchydata.com/blog/tls-postgres-kubernetes-openssl) as well.

To set up TLS for your PostgreSQL cluster, you have to create two [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/): one that contains the CA certificate, and the other that contains the server TLS key pair.

First, create the Secret that contains your CA certificate. Create the Secret as a generic Secret, and note that the following requirements **must** be met:

- The Secret must be created in the same Namespace as where you are deploying your PostgreSQL cluster
- The `name` of the key that is holding the CA **must** be `ca.crt`

There are optional settings for setting up the CA secret:

- You can pass in a certificate revocation list (CRL) for the CA secret by passing in the CRL using the `ca.crl` key name in the Secret.

For example, to create a CA Secret with the trusted CA to use for the PostgreSQL clusters, you could execute the following command:

```
kubectl create secret generic postgresql-ca -n pgo --from-file=ca.crt=/path/to/ca.crt
```

To create a CA Secret that includes a CRL, you could execute the following command:

```
kubectl create secret generic postgresql-ca -n pgo \
  --from-file=ca.crt=/path/to/ca.crt \
  --from-file=ca.crl=/path/to/ca.crl
```

Note that you can reuse this CA Secret for other PostgreSQL clusters deployed by the PostgreSQL Operator.

Next, create the Secret that contains your TLS key pair. Create the Secret as a a TLS Secret, and note the following requirement must be met:

- The Secret must be created in the same Namespace as where you are deploying your PostgreSQL cluster

```
kubectl create secret tls hippo-tls-keypair -n pgo \
  --cert=/path/to/server.crt \
  --key=/path/to/server.key
```

Now you can create a TLS-enabled PostgreSQL cluster!

## Create a Postgres Cluster with TLS

Using the above example, to create a TLS-enabled PostgreSQL cluster that can accept both TLS and non-TLS connections, execute the following command:

```
pgo create cluster hippo \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hippo-tls-keypair
```

Including the `--server-ca-secret` and `--server-tls-secret` flags automatically enable TLS connections in the PostgreSQL cluster that is deployed. These flags should reference the CA Secret and the TLS key pair Secret, respectively.

If deployed successfully, when you connect to the PostgreSQL cluster, assuming your `PGSSLMODE` is set to `prefer` or higher, you will see something like this in your `psql` terminal:

```
SSL connection (protocol: TLSv1.2, cipher: ECDHE-RSA-AES256-GCM-SHA384, bits: 256, compression: off)
```

## Force TLS For All Connections

There are many environments where you want to force all remote connections to occur over TLS, for example, if you deploy your PostgreSQL cluster's in a public cloud or on an untrusted network. The PostgreSQL Operator lets you force all remote connections to occur over TLS by using the `--tls-only` flag.

For example, using the setup above, you can force TLS in a PostgreSQL cluster by executing the following command:

```
pgo create cluster hippo \
  --tls-only \
  --server-ca-secret=postgresql-ca --server-tls-secret=hippo-tls-keypair
```

If deployed successfully, when you connect to the PostgreSQL cluster, assuming your `PGSSLMODE` is set to `prefer` or higher, you will see something like this in your `psql` terminal:

```
SSL connection (protocol: TLSv1.2, cipher: ECDHE-RSA-AES256-GCM-SHA384, bits: 256, compression: off)
```

If you try to connect to a PostgreSQL cluster that is deployed using the `--tls-only` with TLS disabled (i.e. `PGSSLMODE=disable`), you will receive an error that connections without TLS are unsupported.

### TLS Authentication for Replicas

PostgreSQL supports [certificate-based authentication](https://www.postgresql.org/docs/current/auth-cert.html), which allows for PostgreSQL to authenticate users based on the common name (CN) in a certificate. Using this feature, the PostgreSQL Operator allows you to configure PostgreSQL replicas in a cluster to authenticate using a certificate instead of a password.

To use this feature, first you will need to set up a Kubernetes TLS Secret that has a CN of `primaryuser`. If you do not wish to have this as your CN, you will need to map the CN of this certificate to the value of `primaryuser` using a [pg_ident](https://www.postgresql.org/docs/current/auth-username-maps.html) username map, which you can configure as part of a [custom PostgreSQL configuration]({{< relref "/advanced/custom-configuration.md" >}}).

You also need to ensure that the certificate is verifiable by the certificate authority (CA) chain that you have provided for your PostgreSQL cluster. The CA is provided as part of the `--server-ca-secret` flag in the [`pgo create cluster`]({{< relref "/pgo-client/reference/pgo_create_cluster.md" >}}) command.

To create a PostgreSQL cluster that uses TLS authentication for replication, first create Kubernetes Secrets for the server and the CA. For the purposes of this example, we will use the ones that were created earlier: `postgresql-ca` and `hippo-tls-keypair`. After generating a certificate that has a CN of `primaryuser`, create a Kubernetes Secret that references this TLS keypair called `hippo-tls-replication-keypair`:

```
kubectl create secret tls hippo-tls-replication-keypair -n pgo \
  --cert=/path/to/replication.crt \
  --key=/path/to/replication.key
```

We can now create a PostgreSQL cluster and allow for it to use TLS authentication for its replicas! Let's create a PostgreSQL cluster with two replicas that also requires TLS for any connection:

```
pgo create cluster hippo \
  --tls-only \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hippo-tls-keypair \
  --replication-tls-secret=hippo-tls-replication-keypair \
  --replica-count=2
```

By default, the PostgreSQL Operator has each replica connect to PostgreSQL using a [PostgreSQL TLS mode](https://www.postgresql.org/docs/current/libpq-ssl.html#LIBPQ-SSL-SSLMODE-STATEMENTS) of `verify-ca`. If you wish to perform TLS mutual authentication between PostgreSQL instances (i.e. certificate-based authentication with SSL mode of `verify-full`), you will need to create a [PostgreSQL custom configuration]({{< relref "/advanced/custom-configuration.md" >}}).

## Add TLS to an Existing PostgreSQL Cluster

You can add TLS to an existing PostgreSQL cluster using the [`pgo update cluster`]({{< relref "/pgo-client/reference/pgo_update_cluster.md" >}}) or by modifying the `pgclusters.crunchydata.com` custom resource directly. `pgo update cluster` provides several flags for TLS management, including:

- `--disable-server-tls`: removes TLS from a cluster
- `--disable-tls-only`: removes the TLS-only requirement from a cluster
- `--enable-tls-only`: adds the TLS-only requirement to a cluster
- `--server-ca-secret`: combined with `--server-tls-secret`, enables TLS in a cluster
- `--server-tls-secret`: combined with `--server-ca-secret`, enables TLS in a cluster
- `--replication-tls-secret`: enables certificate-based authentication between Postgres instances.

If you have an existing cluster named `hippo` that does not have TLS, and have a TLS keypair in a Secret named `hippo-tls-keypair` and a CA in a Secret name `postgresql-ca` and want to require all connections to use TLS, you could use the following command:

```
pgo update cluster hippo \
  --enable-tls-only \
  --server-ca-secret=postgresql-ca \
  --server-tls-secret=hippo-tls-keypair
```

While PGO attempts to leave any `pg_hba.conf` customizations you have in place, there are circumstance where it can override them when enabling/disabling TLS. If you do have custom `pg_hba.conf` rules, after adding or removing TLS from an existing Posgres cluster, check your `pg_hba.conf` values to ensure it matches your expectations.

## Troubleshooting

### Replicas Cannot Connect to Primary

If your primary is forcing all connections over TLS, ensure that your replicas are connecting with a `sslmode` of `prefer` or higher.

If using TLS authentication with your replicas, ensure that the common name (`CN`) for the replicas is `primaryuser` or that you have set up an entry in `pg_ident` that provides a mapping from your `CN` to `primaryuser`.

### `pg_hba.conf` Values Have Changed After TLS Update

PGO will attempt to preserve all of your custom TLS rules, but there are cases where it may make modifications. This a normal part of adding/removing TLS from an existing Postgres cluster. You can safely update your `pg_hba.conf` rules after the TLS changes are completed, and they will be preserved.

## Next Steps

You've now secured connections to your database. However, how do you scale and pool your PostgreSQL connections? Learn how to [set up and configure pgBouncer]({{< relref "tutorial/pgbouncer.md" >}})!
