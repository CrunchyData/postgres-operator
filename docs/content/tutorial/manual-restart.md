---
title: "Manually Restart PostgreSQL"
date:
draft: false
weight: 105
---

There are times when you might need to manually restart PostgreSQL. This can be done utilizing PGO's rolling updates with minimal downtime by simply adding or updating a custom annotation for the cluster's instance sets `spec.metadata.annotations`, the Pod template will change for all instances and will trigger a rolling update for the cluster.

For example, if I have a cluster named `hippo` in the namespace `postgres-operator`, I would set the following:

```
spec:
  metadata:
    annotations:
      restart: "yes"
```

When your configuration is saved, you can redeploy your cluster:

```
kubectl apply -k kustomize/postgres/
```

Watch your hippo cluster: you will see the rolling update has been triggered and the restart has begun.

## Next Steps

We've covered a lot in terms of building, maintaining, scaling, customizing, restarting, and expanding our Postgres cluster. However, there may come a time where we need to [delete our Postgres cluster]({{< relref "delete-cluster.md" >}}). How do we do that?

