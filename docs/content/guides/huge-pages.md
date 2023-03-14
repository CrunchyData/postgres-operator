---
title: "Huge Pages"
date:
draft: false
weight: 100
---

# Huge Pages

Huge Pages, a.k.a. "Super Pages" or "Large Pages", are larger chunks of memory that can speed up your system. Normally, the chunks of memory, or "pages", used by the CPU are 4kB in size. The more memory a process needs, the more pages the CPU needs to manage. By using larger pages, the CPU can manage fewer pages and increase its efficiency. For this reason, it is generally recommended to use Huge Pages with your Postgres databases.

# Configuring Huge Pages with PGO

To turn Huge Pages on with PGO, you first need to have Huge Pages turned on at the OS level. This means having them enabled, and a specific number of pages preallocated, on the node(s) where you plan to schedule your pods. All processes that run on a given node and request Huge pages will be sharing this pool of pages, so it is important to allocate enough pages for all the different processes to get what they need. This system/kube-level configuration is outside the scope of this document, since the way that Huge Pages are configured at the OS/node level is dependent on your Kube environment. Consult your Kube environment documentation and any IT support you have for assistance with this step.

When you enable Huge Pages in your Kube cluster, it is important to keep a few things in mind during the rest of the configuration process:
1. What size of Huge Pages are enabled? If there are multiple sizes enabled, which one is the default? Which one do you want Postgres to use?
2. How many pages were preallocated? Are there any other applications or processes that will be using these pages?
3. Which nodes have Huge Pages enabled? Is it possible that more nodes will be added to the cluster? If so, will they also have Huge Pages enabled?

Once Huge Pages are enabled on one or more nodes in your Kubernetes cluster, you can tell Postgres to start using them by adding some configuration to your PostgresCluster spec (Warning: setting/changing this setting will cause your database to restart):

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: registry.developers.crunchydata.com/crunchydata/crunchy-postgres:ubi8-14.6-2
  postgresVersion: 14
  instances:
    - name: instance1
      resources:
        limits:
          hugepages-2Mi: 16Mi
          memory: 4Gi
```

This is where it is important to know the size and the number of Huge Pages available. In the spec above, the `hugepages-2Mi` line indicates that we want to use 2MiB sized pages. If your system only has 1GiB sized pages available, then you will want to use `hugepages-1Gi` as the setting instead. The value after it, `16Mi` in our example, determines the amount of pages to be allocated to this Postgres instance. If you have multiple instances, you will need to enable/allocate Huge Pages on an instance by instance basis. Keep in mind that if you have a "Highly Available" cluster, meaning you have multiple replicas, each replica will also request Huge Pages. You therefore need to be cognizant of the total amount of Huge Pages available on the node(s) and the amount your cluster is requesting. If you request more pages than are available, you might see some replicas/instances fail to start.

Note: In the `instances.#.resources` spec, there are `limits` and `requests`. If a request value is not specified (like in the example above), it is presumed to be equal to the limit value. For Huge Pages, the request value must always be equal to the limit value, therefore, it is perfectly acceptable to just specify it in the `limits` section.

Note: Postgres uses the system default size by default. This means that if there are multiple sizes of Huge Pages available on the node(s) and you attempt to use a size in your PostgresCluster that is not the system default, it will fail. To use a non-default size you will need to tell Postgres the size to use with the `huge_page_size` variable, which can be set via dynamic configuration (Warning: setting/changing this parameter will cause your database to restart):

```yaml
patroni:
  dynamicConfiguration:
    postgresql:
      parameters:
        huge_page_size: 1GB
```

# The Kubernetes Issue

There is an issue in Kubernetes where essentially, if Huge Pages are available on a node, it will tell the processes running in the pods on that node that it has Huge Pages available even if the pod has not actually requested any Huge Pages. This is an issue because by default, Postgres is set to "try" to use Huge Pages. When Postgres is led to believe that Huge Pages are available and it attempts to use Huge Pages only to find that the pod doesn't actually have any Huge Pages allocated since they were never requested, Postgres will fail.

We have worked around this issue by setting `huge_pages = off` in our newest Crunchy Postgres images. PGO will automatically turn `huge_pages` back to `try` whenever Huge Pages are requested in the resources spec. Those who were already happily using Huge Pages will be unaffected, and those who were not using Huge Pages, but were attempting to run their Postgres containers on nodes that have Huge Pages enabled, will no longer see their databases crash.

The only dilemma that remains is that those whose PostgresClusters are not using Huge Pages, but are running on nodes that have Huge Pages enabled, will see their `shared_buffers` set to their lowest possible setting. This is due to the way that Postgres' `initdb` works when bootstrapping a database. There are few ways to work around this issue:

1. Use Huge Pages! You're already running your Postgres containers on nodes that have Huge Pages enabled, why not use them in Postgres?
2. Create nodes in your Kubernetes cluster that don't have Huge Pages enabled, and put your Postgres containers on those nodes.
3. If for some reason you cannot use Huge Pages in Postgres, but you must run your Postgres containers on nodes that have Huge Pages enabled, you can manually set the `shared_buffers` parameter back to a good setting using dynamic configuration (Warning: setting/changing this parameter will cause your database to restart):

```yaml
patroni:
  dynamicConfiguration:
    postgresql:
      parameters:
        shared_buffers: 128MB
```
