---
title: "Pod Anti-Affinity in PostgreSQL Operator"
date:
draft: false
weight: 3
---

## Pod Anti-Affinity in PostgreSQL Operator

By default, when a new PostgreSQL cluster is created using the PostgreSQL Operator, pod 
anti-affinity rules will be applied to any deployments comprising the full PG cluster
(please note that default pod anti-affinity does not apply to any Kubernetes jobs created by
the PostgreSQL Operator).  This includes:

- The primary PG deployment
- The deployments for each PG replica
- The `pgBackrest` dedicated repostiory deployment
- The `pgBouncer` deployment (if enabled for the cluster)

With default pod anti-affinity enabled, Kubernetes will attempt to schedule the pods created
by each individual deployment above on a unique node, but at the same time will not guarentee that
each deployment will be scheduled on a unqiue node.  This therefore ensures that the various pods
comprising the PG cluster can always be scheduled, even if though they might not end up on the 
desired node.  This is specifically done using the following:

- The `preferredDuringSchedulingIgnoredDuringExecution` affinity type, (as defined by the Kubernetes
API), which defines an anti-affinity rule that Kubernetes will attempt to adhere to, but will not
guarantee
- A combination of labels that uniquely identify the pods created by the various deployments 
listed above as those that the default pod anti-affinity should apply to
- A topology key of `kubernetes.io/hostname`, which instructs Kubernetes to schedule a pod on
specific node (i.e. host) only if there is not already another pod in the PG cluster
scheduled on that same node

While the pod anti-affinity rule discussed above will be enabled by default for all PG clusters,
it can also be explicitly enabled via the `pgo` CLI by setting the `pod-anti-affinity` option
to `preferred`:

```bash
pgo create cluster mycluster --replica-count=2 --pod-anti-affinity=preferred
```

Or it can also be explicitly enabled globally for all clusters by setting `PodAntiAffinity` to
`preferred` in the `pgo.yaml` configuration file.

In addition to providing the ability to set default pod anti-affinity using the `preferred`
option, i.e. an affinity rule that utilizes the `preferredDuringSchedulingIgnoredDuringExecution`
affinity type, it is also possible to use `requiredDuringSchedulingIgnoredDuringExecution` for
the default pod anti-affinity, specifically by setting `pod-anti-affinity` to `required`:

```bash
pgo create cluster mycluster --replica-count=2 --pod-anti-affinity=required
```

Or similarly the `require` option can be enabled globally for all clusters by setting 
`PodAntiAffinity` to `require` in the `pgo.yaml` configuration file.

When `require` is utilized for the default pod anti-affinity, a separate node is required for each
deployment listed above comprising the PG cluster.  This ensures that the cluster remains 
highly-available by ensuring that node failures do not impact any other deployments in the cluster.
However, this does mean that the PG primary, each PG replica, the `pgBackRestRepo` and `pgBouncer`,
will each require a unique node, meaning the minimum number of nodes required for the Kubernetes 
cluster will increase as more deployments (e.g. PG replicas) are added to the PG cluster.  Further,
if an insufficient number of nodes are available to support this configuration, certain deployments 
will fail, since it will not be possible for Kubernetes to successfully schedule the pods for each 
deployment.

And finally, it is also possible to disable the default pod anti-affinity settings all together.
Using the `pgo` CLI this is done by setting `pod-anti-affinity` to `disabled`, e.g.:

```bash
pgo create cluster mycluster --replica-count=2 --pod-anti-affinity=disabled
```

Or the same setting can be applied globally to all clusters created by setting `PodAntiAffinity` 
to `require` in the `pgo.yaml` configuration file.
