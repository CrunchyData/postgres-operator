---
title: "Create a Postgres Cluster"
draft: false
weight: 110
---

If you came here through the [quickstart]({{< relref "quickstart/_index.md" >}}), you may have already [created a cluster]({{< relref "quickstart/_index.md" >}}#create-a-postgresql-cluster), in which case, feel free to skip ahead, or read onward for a more in depth look into cluster creation!

## Create a PostgreSQL Cluster

Creating a cluster is simple with the [`pgo create cluster`]({{< relref "pgo-client/reference/pgo_create_cluster.md" >}}) command:

```
pgo create cluster hippo
```

with output similar to:

```
created cluster: hippo
workflow id: 25c870a0-5d27-42c2-be00-92f0ba8768e7
database name: hippo
users:
	username: testuser password: securerandomlygeneratedpassword
```

This creates a new PostgreSQL cluster named `hippo` with a database in it named `hippo`. This operation may take a few moments to complete. Note the name of the database user (`testuser`) and password (`securerandomlygeneratedpassword`) for when we connect to the PostgreSQL cluster.

To make it easier to copy and paste statements used throughout this tutorial, you can set the password of `testuser` as part of creating the PostgreSQL cluster:

```
pgo create cluster hippo --password=securerandomlygeneratedpassword
```

You can check on the status of the cluster creation using the [`pgo test`]({{< relref "pgo-client/reference/pgo_test.md" >}}) command. The `pgo test` command checks to see if the Kubernetes Services and the Pods that comprise the PostgreSQL cluster are available to receive connections. This includes:

- Testing that the Kubernetes Endpoints are available and able to route requests to healthy Pods.
- Testing that each PostgreSQL instance is available and ready to accept client connections by performing a connectivity check similar to the one performed by [`pg_isready`](https://www.postgresql.org/docs/current/app-pg-isready.html).

For example, when the `hippo` cluster is ready,

```
pgo test hippo
```

will yield output similar to:

```
cluster : hippo
	Services
		primary (10.96.179.126:5432): UP
	Instances
		primary (hippo-57675d4f8f-wwx64): UP
```


### The Create Cluster Process

So what just happened? Let's break down what occurs during the create cluster process.

1. First, `pgo` client creates an entry in the PostgreSQL Operator [pgcluster custom resource definition]({{< relref "custom-resources/_index.md" >}}) with the attributes desired to create the cluster. In the case above, this fills in the name of the cluster (`hippo`) and leverages a lot of defaults from the [PostgreSQL Operator configuration]({{< relref "configuration/pgo-yaml-configuration.md" >}}). We'll discuss more about the PostgreSQL Operator configuration later in the tutorial.

2. Once the custom resource is added, the PostgreSQL Operator begins provisioning the PostgreSQL instace and a pgBackRest repository which is used to store backups. The following actions occur as part of this process:

  - Creating [persistent volume claims](https://kubernetes.io/docs/concepts/storage/persistent-volumes/) (PVCs) for the PostgreSQL instance and the pgBackRest repository.
  - Creating [services](https://kubernetes.io/docs/concepts/services-networking/service/) that provide a stable network interface for connecting to the PostgreSQL instance and pgBackRest repository.
  - Creating [deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/) that house each PostgreSQL instance and pgBackRest repository. Each of these is responsible for one Pod.
  - The PostgreSQL Pod, when it is started, provisions a PostgreSQL database and performs other bootstrapping functions, such as creating `testuer`.
  - The pgBackRest Pod, when it is started, initializes a pgBackRest repository. Note that the pgBackRest repository is not yet ready to start taking backups, but will be after the next step!

3. When the PostgreSQL Operator detects that the PostgreSQL and pgBackRest deployments are up and running, it creates a Kubenretes Job to create a pgBackRest stanza. This is necessary as part of intializing the pgBackRest repository to accept backups from our PostgreSQL cluster.

4. When the PostgreSQL Operator detects that the stanza creation is completed, it will take an initial backup of the cluster.

In order for a PostgreSQL cluster to be considered successfully created, all of these steps need to succeed. You can connect to the PostgreSQL cluster after step two completes, but note for the cluster to be considered "healthy", you need for pgBackRest to finish initializig.

You may ask yourself, "wait, why do I need for the pgBackRest repository to be initialized for a cluster to be successfully created?" That is a good question! The reason is that pgBackRest plays a fundamental role in both the [disaster recovery]({{< relref "architecture/disaster-recovery.md" >}}) AND [high availability]({{< relref "architecture/high-availability/_index.md" >}}) system with the PostgreSQL Operator, particularly around self-healing.

### What Is Created?

There are several Kubernetes objects that are created as part of the `pgo create cluster` command, including:

- A Deployment representing the primary PostgreSQL instance
  - A PVC that persists the data of this instance
  - A Service that can connect to this instance
- A Deployment representing the pgBackRest repository
  - A PVC that persists the data of this repository
  - A Service that can connect to this repository
- [Secrets](https://kubernetes.io/docs/concepts/configuration/secret/) representing the following three user accounts:
  - `postgres`: the database superuser for the PostgreSQL cluster. This is in a secret called `hippo-postgres-secret`.
  - `primaryuser`: the replication user. This is used for copying data between PostgreSQL instance. You should not need to login as this user. This is in a secret called `hippo-primaryuser-secret`.
  - `testuser`: the regular user account. This user has access to log into the `hippo` database that is created. This is the account you want to give out to your user / application. In a later section, we will see how we can change the default user that is created. This is in a secret called `hippo-testuser-secret`, where `testuser` can be substituted for the name of the user account.
- [ConfigMaps](https://kubernetes.io/docs/concepts/configuration/configmap/), including:
  - `hippo-pgha-config`, which allows you to [customize the configuration of your PostgreSQL cluster]({{< relref "advanced/custom-configuration.md">}}). We will cover more about this topic in later sections.
  - `hippo-config` and `hippo-leader`, which are used by the high availability system. You should not modify these ConfigMaps.

Each deployment contains a single Pod. **Do not scale the deployments!**: further into the tutorial, we will cover some commands that let you scale your PostgreSQL cluster.

Some Job artifacts may be left around after the cluster creation process completes, including the stanza creation job (`hippo-stanza-create`) and initial backup job (`backrest-backup-hippo`). If the jobs completed successfully, you can safely delete these objects.

## Troubleshooting

### PostgreSQL / pgBackRest Pods Stuck in `Pending` Phase

The most common occurrence of this is due to PVCs not being bound. Ensure that you have configure your [storage options]({{< relref "installation/configuration.md" >}}#storage-settings) correctly for your Kubernetes environment, if for some reason you cannot use your default storage class or it is unavailable.

Also ensure that you have enough persistent volumes available: your Kubernetes administrator may need to provision more.

### `stanza-create` Job Never Finishes

The most common occurrence of this is due to the Kubernetes network blocking SSH connections between Pods. Ensure that your Kubernetes networking layer allows for SSH connections over port 2022 in the Namespace that you are deploying your PostgreSQL clusters into.

## Next Steps

Once your cluster is created, the next step is to [connect to your PostgreSQL cluster]({{< relref "tutorial/connect-cluster.md" >}}).
