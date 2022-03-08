---
title: "Backup Configuration"
date:
draft: false
weight: 80
---

An important part of a healthy Postgres cluster is maintaining backups. PGO optimizes its use of open source [pgBackRest](https://pgbackrest.org/) to be able to support terabyte size databases. What's more, PGO makes it convenient to perform many common and advanced actions that can occur during the lifecycle of a database, including:

- Setting automatic backup schedules and retention policies
- Backing data up to multiple locations
  - Support for backup storage in Kubernetes, AWS S3 (or S3-compatible systems like MinIO), Google Cloud Storage (GCS), and Azure Blog Storage
- Taking one-off / ad hoc backups
- Performing a "point-in-time-recovery"
- Cloning data to a new instance

and more.

Let's explore the various disaster recovery features in PGO by first looking at how to set up backups.

## Understanding Backup Configuration and Basic Operations

The backup configuration for a PGO managed Postgres cluster resides in the
`spec.backups.pgbackrest` section of a custom resource. In addition to indicating which
version of pgBackRest to use, this section allows you to configure the fundamental
backup settings for your Postgres cluster, including:

- `spec.backups.pgbackrest.configuration` - allows to add additional configuration and references to Secrets that are needed for configuration your backups. For example, this may reference a Secret that contains your S3 credentials.
- `spec.backups.pgbackrest.global` - a convenience to apply global [pgBackRest configuration](https://pgbackrest.org/configuration.html). An example of this may be setting the global pgBackRest logging level (e.g. `log-level-console: info`), or provide configuration to optimize performance.
- `spec.backups.pgbackrest.repos` - information on each specific pgBackRest backup repository.
  This allows you to configure where and how your backups and WAL archive are stored.
  You can keep backups in up to four (4) different locations!

You can configure the `repos` section based on the backup storage system you are looking to use. Specifically, you configure your `repos` section according to the storage type you are using. There are four storage types available in `spec.backups.pgbackrest.repos`:

| Storage Type | Description  |
|--------------| ------------ |
| `azure`      | For use with Azure Blob Storage. |
| `gcs`        | For use with Google Cloud Storage (GCS). |
| `s3`         | For use with Amazon S3 or any S3 compatible storage system such as MinIO. |
| `volume`     | For use with a Kubernetes [Persistent Volume](https://kubernetes.io/docs/concepts/storage/persistent-volumes/). |


Regardless of the backup storage system you select, you **must** assign a name to `spec.backups.pgbackrest.repos.name`, e.g. `repo1`. pgBackRest follows the convention of assigning configuration to a specific repository using a `repoN` format, e.g. `repo1`, `repo2`, etc. You can customize your configuration based upon the name that you assign in the spec. We will cover this topic further in the multi-repository example.

By default, backups are stored in a directory that follows the pattern `pgbackrest/repoN` where `N` is the number of the repo. This typically does not present issues when storing your backup information in a Kubernetes volume, but it can present complications if you are storing all of your backups in the same backup in a blob storage system like S3/GCS/Azure. You can avoid conflicts by setting the `repoN-path` variable in `spec.backups.pgbackrest.global`. The convention we recommend for setting this variable is `/pgbackrest/$NAMESPACE/$CLUSTER_NAME/repoN`. For example, if I have a cluster named `hippo` in the namespace `postgres-operator`, I would set the following:

```
spec:
  backups:
    pgbackrest:
      global:
        repo1-path: /pgbackrest/postgres-operator/hippo/repo1
```

As mentioned earlier, you can store backups in up to four different repositories. You can also mix and match, e.g. you could store your backups in two different S3 repositories. Each storage type does have its own required attributes that you need to set. We will cover that later in this section.

Now that we've covered the basics, let's learn how to set up our backup repositories!

## Setting Up a Backup Repository

As mentioned above, PGO, the Postgres Operator from Crunchy Data, supports multiple ways to store backups. Let's look into each method and see how you can ensure your backups and archives are being safely stored!

## Using Kubernetes Volumes

The simplest way to get started storing backups is to use a Kubernetes Volume. This was already configure as part of the [create a Postgres cluster]({{< relref "./create-cluster.md">}}) example. Let's take a closer look at some of that configuration:

```
- name: repo1
  volume:
    volumeClaimSpec:
      accessModes:
      - "ReadWriteOnce"
      resources:
        requests:
          storage: 1Gi
```

The one requirement of volume is that you need to fill out the `volumeClaimSpec` attribute. This attribute uses the same format as a [persistent volume claim](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) spec! In fact, we performed a similar set up when we [created a Postgres cluster]({{< relref "./create-cluster.md">}}).

In the above example, we assume that the Kubernetes cluster is using a default storage class. If your cluster does not have a default storage class, or you wish to use a different storage class, you will have to set `spec.backups.pgbackrest.repos.volume.volumeClaimSpec.storageClassName`.

## Using S3

Setting up backups in S3 requires a few additional modifications to your custom resource spec
and either
- the use of a Secret to protect your S3 credentials, or
- setting up identity providers in AWS to allow pgBackRest to assume a role with permissions.

### Using S3 Credentials

There is an example for creating a Postgres cluster that uses S3 for backups in the `kustomize/s3` directory in the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repository. In this directory, there is a file called `s3.conf.example`. Copy this example file to `s3.conf`:

```
cp s3.conf.example s3.conf
```

Note that `s3.conf` is protected from commit by a `.gitignore`.

Open up `s3.conf`, you will see something similar to:

```
[global]
repo1-s3-key=<YOUR_AWS_S3_KEY>
repo1-s3-key-secret=<YOUR_AWS_S3_KEY_SECRET>
```

Replace the values with your AWS S3 credentials and save.

Now, open up `kustomize/s3/postgres.yaml`. In the `s3` section, you will see something similar to:

```
s3:
  bucket: "<YOUR_AWS_S3_BUCKET_NAME>"
  endpoint: "<YOUR_AWS_S3_ENDPOINT>"
  region: "<YOUR_AWS_S3_REGION>"
```

Again, replace these values with the values that match your S3 configuration. For `endpoint`, only use the domain and, if necessary, the port (e.g. `s3.us-east-2.amazonaws.com`).

Note that `region` is required by S3, as does pgBackRest. If you are using a storage system with a S3 compatibility layer that does not require `region`, you can fill in region with a random value.

If you are using MinIO, you may need to set the URI style to use `path` mode. You can do this from the global settings, e.g. for `repo1`:

```yaml
spec:
  backups:
    pgbackrest:
      global:
        repo1-s3-uri-style: path
```

When your configuration is saved, you can deploy your cluster:

```
kubectl apply -k kustomize/s3
```

Watch your cluster: you will see that your backups and archives are now being stored in S3!

### Using an AWS-integrated identity provider and role

If you deploy PostgresClusters to AWS Elastic Kubernetes Service, you can take advantage of their
IAM role integration. When you attach a certain annotation to your PostgresCluster spec, AWS will
automatically mount an AWS token and other needed environment variables. These environment
variables will then be used by pgBackRest to assume the identity of a role that has permissions
to upload to an S3 repository.

This method requires [additional setup in AWS IAM](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html).
Use the procedure in the linked documentation for the first two steps described below:

1. Create an OIDC provider for your EKS cluster.
2. Create an IAM policy for bucket access and an IAM role with a trust relationship with the
OIDC provider in step 1.

The third step is to associate that IAM role with a ServiceAccount, but there's no need to
do that manually, as PGO does that for you. First, make a note of the IAM role's `ARN`.

You can then make the following changes to the files in the `kustomize/s3` directory in the
[Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repository:

1\. Add the `s3` section to the spec in `kustomize/s3/postgres.yaml` as discussed in the
[Using S3 Credentials](#using-s3-credentials) section above. In addition to that, add the required `eks.amazonaws.com/role-arn`
annotation to the PostgresCluster spec using the IAM `ARN` that you noted above.

For instance, given an IAM role with the ARN `arn:aws:iam::123456768901:role/allow_bucket_access`,
you would add the following to the PostgresCluster spec:

```
spec:
  metadata:
    annotations:
      eks.amazonaws.com/role-arn: "arn:aws:iam::123456768901:role/allow_bucket_access"
```

That `annotations` field will get propagated to the ServiceAccounts that require it automatically.

2\. Copy the `s3.conf.example` file to `s3.conf`:

```
cp s3.conf.example s3.conf
```

Update that `kustomize/s3/s3.conf` file so that it looks like this:

```
[global]
repo1-s3-key-type=web-id
```

That `repo1-s3-key-type=web-id` line will tell
[pgBackRest](https://pgbackrest.org/configuration.html#section-repository/option-repo-s3-key-type)
to use the IAM integration.

With those changes saved, you can deploy your cluster:

```
kubectl apply -k kustomize/s3
```

And watch as it spins up and backs up to S3 using pgBackRest's IAM integration.

## Using Google Cloud Storage (GCS)

Similar to S3, setting up backups in Google Cloud Storage (GCS) requires a few additional modifications to your custom resource spec and the use of a Secret to protect your GCS credentials.

There is an example for creating a Postgres cluster that uses GCS for backups in the `kustomize/gcs` directory in the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repository. In order to configure this example to use GCS for backups, you will need do two things.

First, copy your GCS key secret (which is a JSON file) into `kustomize/gcs/gcs-key.json`. Note that a `.gitignore` directive prevents you from committing this file.

Next, open the `postgres.yaml` file and edit `spec.backups.pgbackrest.repos.gcs.bucket` to the name of the GCS bucket that you want to back up to.

Save this file, and then run:

```
kubectl apply -k kustomize/gcs
```

Watch your cluster: you will see that your backups and archives are now being stored in GCS!

## Using Azure Blob Storage

Similar to the above, setting up backups in Azure Blob Storage requires a few additional modifications to your custom resource spec and the use of a Secret to protect your GCS credentials.

There is an example for creating a Postgres cluster that uses Azure for backups in the `kustomize/azure` directory in the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repository. In this directory, there is a file called `azure.conf.example`. Copy this example file to `azure.conf`:

```
cp azure.conf.example azure.conf
```

Note that `azure.conf` is protected from commit by a `.gitignore`.

Open up `azure.conf`, you will see something similar to:

```
[global]
repo1-azure-account=<YOUR_AZURE_ACCOUNT>
repo1-azure-key=<YOUR_AZURE_KEY>
```

Replace the values with your Azure credentials and save.

Now, open up `kustomize/azure/postgres.yaml`. In the `azure` section, you will see something similar to:

```
azure:
  container: "<YOUR_AZURE_CONTAINER>"
```

Again, replace these values with the values that match your Azure configuration.

When your configuration is saved, you can deploy your cluster:

```
kubectl apply -k kustomize/azure
```

Watch your cluster: you will see that your backups and archives are now being stored in Azure!

## Set Up Multiple Backup Repositories

It is possible to store backups in multiple locations! For example, you may want to keep your backups both within your Kubernetes cluster and S3. There are many reasons for doing this:

- It is typically faster to heal Postgres instances when your backups are closer
- You can set different backup retention policies based upon your available storage
- You want to ensure that your backups are distributed geographically

and more.

PGO lets you store your backups in up to four locations simultaneously. You can mix and match: for example, you can store backups both locally and in GCS, or store your backups in two different GCS repositories. It's up to you!

There is an example in the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repository in the `kustomize/multi-backup-repo` folder that sets up backups in four different locations using each storage type. You can modify this example to match your desired backup topology.

### Additional Notes

While storing Postgres archives (write-ahead log [WAL] files) occurs in parallel when saving data to multiple pgBackRest repos, you cannot take parallel backups to different repos at the same time. PGO will ensure that all backups are taken serially. Future work in pgBackRest will address parallel backups to different repos. Please don't confuse this with parallel backup: pgBackRest does allow for backups to use parallel processes when storing them to a single repo!

## Encryption

You can encrypt your backups using AES-256 encryption using the CBC mode. This can be used independent of any encryption that may be supported by an external backup system.

To encrypt your backups, you need to set the cipher type and provide a passphrase. The passphrase should be long and random (e.g. the pgBackRest documentation recommends `openssl rand -base64 48`). The passphrase should be kept in a Secret.

Let's use our `hippo` cluster as an example. Let's create a new directory. First, create a file called `pgbackrest-secrets.conf` in this directory. It should look something like this:

```
[global]
repo1-cipher-pass=your-super-secure-encryption-key-passphrase
```

This contains the passphrase used to encrypt your data.

Next, create a `kustomization.yaml` file that looks like this:

```yaml
namespace: postgres-operator

secretGenerator:
- name: hippo-pgbackrest-secrets
  files:
  - pgbackrest-secrets.conf

generatorOptions:
  disableNameSuffixHash: true

resources:
- postgres.yaml
```

Finally, create the manifest for the Postgres cluster in a file named `postgres.yaml` that is similar to the following:

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
  instances:
    - dataVolumeClaimSpec:
        accessModes:
        - "ReadWriteOnce"
        resources:
          requests:
            storage: 1Gi
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
      configuration:
      - secret:
          name: hippo-pgbackrest-secrets
      global:
        repo1-cipher-type: aes-256-cbc
      repos:
      - name: repo1
        volume:
          volumeClaimSpec:
            accessModes:
            - "ReadWriteOnce"
            resources:
              requests:
                storage: 1Gi

```

Notice the reference to the Secret that contains the encryption key:

```yaml
spec:
  backups:
    pgbackrest:
      configuration:
      - secret:
          name: hippo-pgbackrest-secrets
```

as well as the configuration for enabling AES-256 encryption using the CBC mode:

```yaml
spec:
  backups:
    pgbackrest:
      global:
        repo1-cipher-type: aes-256-cbc
```

You can now create a Postgres cluster that has encrypted backups!

### Limitations

Currently the encryption settings cannot be changed on backups after they are established.

## Custom Backup Configuration

Most of your backup configuration can be configured through the `spec.backups.pgbackrest.global` attribute, or through information that you supply in the ConfigMap or Secret that you refer to in `spec.backups.pgbackrest.configuration`. You can also provide additional Secret values if need be, e.g. `repo1-cipher-pass` for encrypting backups.

The full list of [pgBackRest configuration options](https://pgbackrest.org/configuration.html) is available here:

[https://pgbackrest.org/configuration.html](https://pgbackrest.org/configuration.html)

## Next Steps

We've now seen how to use PGO to get our backups and archives set up and safely stored. Now let's take a look at [backup management]({{< relref "./backup-management.md" >}}) and how we can do things such as set backup frequency, set retention policies, and even take one-off backups!
