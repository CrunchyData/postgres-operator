---
title: "Delete a Postgres Cluster"
date:
draft: false
weight: 110
---

There comes a time when it is necessary to delete your cluster. If you have been [following along with the example](https://github.com/CrunchyData/postgres-operator-examples), you can delete your Postgres cluster by simply running:

```
kubectl delete -k kustomize/postgres
```

PGO will remove all of the objects associated with your cluster.

With data retention, this is subject to the [retention policy of your PVC](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming). For more information on how Kubernetes manages data retention, please refer to the [Kubernetes docs on volume reclaiming](https://kubernetes.io/docs/concepts/storage/persistent-volumes/#reclaiming).
