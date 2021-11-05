---
title: "Apply Software Updates"
date:
draft: false
weight: 70
---

Did you know that Postgres releases bug fixes [once every three months](https://www.postgresql.org/developer/roadmap/)? Additionally, we periodically refresh the container images to ensure the base images have the latest software that may fix some CVEs.

It's generally good practice to keep your software up-to-date for stability and security purposes, so let's learn how PGO helps to you accept low risk, "patch" type updates.

The good news: you do not need to update PGO itself to apply component updates: you can update each Postgres cluster whenever you want to apply the update! This lets you choose when you want to apply updates to each of your Postgres clusters, so you can update it on your own schedule. If you have a [high availability Postgres]({{< relref "./high-availability.md" >}}) cluster, PGO uses a rolling update to minimize or eliminate any downtime for your application.

## Applying Minor Postgres Updates

The Postgres image is referenced using the `spec.image` and looks similar to the below:

```
spec:
  image: registry.developers.crunchydata.com/crunchydata/crunchy-postgres:centos8-13.4-0
```

Diving into the tag a bit further, you will notice the `13.4-0` portion. This represents the Postgres minor version (`13.4`) and the patch number of the release `0`. If the patch number is incremented (e.g. `13.4-1`), this means that the container is rebuilt, but there are no changes to the Postgres version. If the minor version is incremented (e.g. `13.4-0`), this means that the is a newer bug fix release of Postgres within the container.

To update the image, you just need to modify the `spec.image` field with the new image reference, e.g.

```
spec:
  image: registry.developers.crunchydata.com/crunchydata/crunchy-postgres:centos8-13.4-1
```

You can apply the changes using `kubectl apply`. Similar to the rolling update example when we [resized the cluster]({{< relref "./resize-cluster.md" >}}), the update is first applied to the Postgres replicas, then a controlled switchover occurs, and the final instance is updated.

For the `hippo` cluster, you can see the status of the rollout by running the command below:

```
kubectl -n postgres-operator get pods \
  --selector=postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/instance \
  -o=jsonpath='{range .items[*]}{.metadata.name}{"\t"}{.metadata.labels.postgres-operator\.crunchydata\.com/role}{"\t"}{.status.phase}{"\t"}{.spec.containers[].image}{"\n"}{end}'
```

or by running a watch:

```
watch "kubectl -n postgres-operator get pods \
  --selector=postgres-operator.crunchydata.com/cluster=hippo,postgres-operator.crunchydata.com/instance \
  -o=jsonpath='{range .items[*]}{.metadata.name}{\"\t\"}{.metadata.labels.postgres-operator\.crunchydata\.com/role}{\"\t\"}{.status.phase}{\"\t\"}{.spec.containers[].image}{\"\n\"}{end}'"
```

## Rolling Back Minor Postgres Updates

This methodology also allows you to rollback changes from minor Postgres updates. You can change the `spec.image` field to your desired container image. PGO will then ensure each Postgres instance in the cluster rolls back to the desired image.

## Applying Other Component Updates

There are other components that go into a PGO Postgres cluster. These include pgBackRest, PgBouncer and others. Each one of these components has its own image: for example, you can find a reference to the pgBackRest image in the `spec.backups.pgbackrest.image` attribute.

Applying software updates for the other components in a Postgres cluster works similarly to the above. As pgBackRest and PgBouncer are Kubernetes [Deployments](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/), Kubernetes will help manage the rolling update to minimize disruption.

## Next Steps

Now that we know how to update our software components, let's look at how PGO handles [disaster recovery]({{< relref "./backups.md" >}})!
