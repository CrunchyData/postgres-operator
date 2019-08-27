---
title: "Node Affinity in PostgreSQL Operator"
date:
draft: false
weight: 3
---

## Node Affinity in PostgreSQL Operator 

Kubernetes node affinity allows you to constrain which nodes your pod is eligible to be scheduled on, based on labels on the node.

The PostgreSQL Operator provides users with the ability to add a node affinity section to a new Cluster Deployment.  By adding a node affinity section to the Cluster Deployment, users can direct Kubernetes to attempt to schedule a primary PostgreSQL instance within a cluster on a specific Kubernetes node.

As an example, you can see the nodes on your Kubernetes cluster by running the following:
```
kubectl get nodes
```

You can then specify one of those Kubernetes node names (e.g. kubeadm-node2) when creating a PostgreSQL cluster;
```
pgo create cluster thatcluster --node-label=kubeadm-node2
```

The node affinity rule inserted in the Deployment uses a *preferred* strategy so that if the node were down or not available, Kubernetes will go ahead and schedule the Pod on another node.

When you scale up a PostgreSQL cluster by adding a PostgreSQL replica instance, the scaling will take into account the use of `--node-label`.  If it sees that a PostgreSQL cluster was created with a specific node name, then the PostgreSQL replica Deployment will add an affinity rule to attempt to schedule the Pod. 






