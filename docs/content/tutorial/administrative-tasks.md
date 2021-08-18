---
title: "Administrative Tasks"
date:
draft: false
weight: 105
---

## Manually Restarting PostgreSQL

There are times when you might need to manually restart PostgreSQL. This can be done, with minimal downtime, by using PGO's rolling update functionality. By simply adding or updating a custom annotation to the cluster instance-set's `spec.metadata.annotations` section, the Pod template will change for all instances and trigger a rolling update for the cluster. 

For example, if I have a cluster named `hippo` in the namespace `postgres-operator`, all you need to do is patch the hippo postgrescluster with the following:

```
kubectl patch postgrescluster/hippo --type=merge --patch='{"spec":{"metadata":{"annotations":{"restart":"'"$(date)"'"}}}}' -n postgres-operator
```

Watch your hippo cluster: you will see the rolling update has been triggered and the restart has begun.

## Next Steps

We've covered a lot in terms of building, maintaining, scaling, customizing, restarting, and expanding our Postgres cluster. However, there may come a time where we need to [delete our Postgres cluster]({{< relref "delete-cluster.md" >}}). How do we do that?

