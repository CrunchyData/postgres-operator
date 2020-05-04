---
title: "Multi-Zone Cloud Considerations"
date:
draft: false
weight: 5
---

## Considerations for PostgreSQL Operator Deployments in Multi-Zone Cloud Environments

#### Overview

When using the PostgreSQL Operator in a Kubernetes cluster consisting of nodes that span multiple zones, special consideration
must be taken to ensure all pods and the associated volumes re scheduled and provisioned within the same zone.  

Given that a pod is unable mount a volume that is located in another zone, any volumes that are dynamically provisioned must
be provisioned in a topology-aware manner according to the specific scheduling requirements for the pod. 

This means that when a new PostgreSQL cluster is created, it is necessary to ensure that the volume containing the database
files for the primary PostgreSQL database within the PostgreSQL clluster is provisioned in the same zone as the node containing the PostgreSQL primary pod that will be accesing the applicable volume.

#### Dynamic Provisioning of Volumes: Default Behavior

By default, the Kubernetes scheduler will ensure any pods created that claim a specific volume via a PVC are scheduled on a 
node in the same zone as that volume.  This is part of the default Kubernetes [multi-zone support](https://kubernetes.io/docs/setup/multiple-zones/). 

However, when using Kubernetes [dynamic provisioning](https://kubernetes.io/docs/concepts/storage/dynamic-provisioning/),
volumes are not provisioned in a topology-aware manner.

More specifically, when using dynamnic provisioning, volumes wills not be provisioned according to the same scheduling
requirements that will be placed on the pod that will be using it (e.g. it will not consider node selectors, resource
requirements, pod affinity/anti-affinity, and various other scheduling requirements).  Rather, PVCs are immediately bound as
soon as they are requested, which means volumes are provisioned without knowledge of these scheduling requirements.

This behavior defined using the `volumeBindingMode` configuration applicable to the Storage Class being utilized to
dynamically provision the volume.  By default,`volumeBindingMode` is set to `Immediate`.  

This default behavior for dynamic provisioning can be seen in the Storage Class definition for a Google Cloud Engine Persistent Disk (GCE PD):

```bash
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: example-sc
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
volumeBindingMode: Immediate
```
As indicated, `volumeBindingMode` indicates the default value of `Immediate`.

#### Issues with Dynamic Provisioning of Volumes in PostgreSQL Operator

Unfortunately, the default setting for dynamic provisinoing of volumes in mulit-zone Kubernetes cluster environments results in undesired behavior when using the PostgreSQL Operator.  

Within the PostgreSQL Operator, a **node label** is implemented as a `preferredDuringSchedulingIgnoredDuringExecution` node
affinity rule, which is an affinity rule that Kubernetes will attempt to adhere to when scheduling any pods for the cluster,
but _will not guarantee_. More information on node affinity rules can be found [here](https://kubernetes.i/docs/concepts/configuration/assign-pod-node/#affinity-and-anti-affinity)). 

By using `Immediate` for the `volumeBindingMode` in a multi-zone cluster environment, the scheduler will ignore any requested
_(but not mandatory)_ scheduling requirements if necessary to ensure the pod can be scheduled. The scheduler will ultimately
schedule the pod on a node in the same zone as the volume, even if another node was requested for scheduling that pod. 

As it relates to the PostgreSQL Operator specifically, a node label specified using the `--node-label` option when creating a
cluster using the `pgo create cluster` command in order target a specific node (or nodes) for the deployment of that cluster. 

Therefore, if the volume ends up in a zone other than the zone containing the node (or nodes) defined by the node label, the
node label will be ignored, and the pod will be scheduled according to the zone containing the volume.  

#### Configuring Volumes to be Topology Aware

In order to overcome this default behavior, it is necessary to make the dynamically provisioned volumes topology aware.  

This is accomplished by setting the `volumeBindingMode` for the storage class to `WaitForFirstConsumer`, which delays the
dynamic provisioning of a volume until a pod using it is created. 

In other words, the PVC is no longer bound as soon as it is requested, but rather waits for a pod utilizing it to be creating
prior to binding.  This change ensures that volume can take into account the scheduling requirements for the pod, which in the
case of a multi-zone cluster means ensuring the volume is provisioned in the same zone containing the node where the pod has
be scheduled.  This also means the scheduler should no longer ignore a node label in order to follow a volume to another zone
when scheduling a pod, since the volume will now follow the pod according to the pods specificscheduling requirements.  

The following is an example of the the same Storage Class defined above, only with `volumeBindingMode` now set to `WaitForFirstConsumer`:

```bash
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: example-sc
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
volumeBindingMode: WaitForFirstConsumer
```

#### Additional Solutions

If you are using a version of Kubernetes that does not support `WaitForFirstConsumer`, an alternate _(and now deprecated)_
solution exists in the form of parameters that can be defined on the Storage Class definition to ensure volumes are
provisioned in a specific zone (or zones).  

For instance, when defining a Storage Class for a GCE PD for use in Google Kubernetes Engine (GKE) cluster, the **zone**
parameter can be used to ensure any volumes dynamically provisioned using that Storage Class are located in that specific
zone.  The following is an example of a Storage Class for a GKE cluster that will provision volumes in the **us-east1** zone:

```bash
kind: StorageClass
apiVersion: storage.k8s.io/v1
metadata:
  name: example-sc
provisioner: kubernetes.io/gce-pd
parameters:
  type: pd-standard
  replication-type: none
  zone: us-east1
```

Once storage classes have been defined for one or more zones, they can then be defined as one or more storage configurations
within the pgo.yaml configuration file (as described in the [PGO YAML configuration guide](/configuration/pgo-yaml
configuration)).  

From there those storage configurations can then be selected when creating a new cluster, as shown in the following example:

```bash
pgo create cluster mycluster --storage-config=example-sc
```

With this approach, the pod will once again be scheduled according to the zone in which the volume was provisioned. 

However, the zone parameters defined on the Storage Class bring consistency to scheduling by guaranteeing that the volume, and
therefore also the pod using that volume, are scheduled in a specific zone as defined by the user, bringing consistency
and predictability to volume provisioning and pod scheduling in multi-zone clusters.

For more information regarding the specific parameters available for the Storage Classes being utilizing in your cloud 
environment, please see the
[Kubernetes documentation for Storage Classes](https://kubernetes.io/docs/concepts/storage/storage-classes/).

Lastly, while the above applies to the dynamic provisioning of volumes, it should be noted that volumes can also be manually
provisioned in desired zones in order to achieve the desired topology requirements for any pods and their volumes.
