
---
title: "Upgrading"
date: {docdate}
draft: false

weight: 70
---


## Upgrading the Operator
Various Operator releases will require action by the Operator administrator of your organization in order to upgrade to the next release of the Operator.  Some upgrade steps are automated within the Operator but not all are possible at this time.

This section of the documentation shows specific steps required to the
latest version from the previous version.

### Upgrading to Version 3.5.0 From Previous Versions
 * the pgingest CRD is removed, you will need to manually remove it from any deployments of the operator after upgrading to this version, this includes removing ingest related permissions from the pgorole file, the API server also
removes the ingest related API endpoints
 * primary and replica labels are only applicable at cluster creation and not updated after a cluster has executed a failover, a new *service-name* label is applied to PG cluster components to indidate whether a deployment/pod is a primary or replica, *service-name* is also the label now used by the cluster services to route with.  This scheme allows for an almost immediate failover promotion and avoids the pod having to be bounced as part of a failover.  Any existing PG clusters will need to be updated to specify them as a primary or replica using the new *service-name* labeling scheme.  
 * the autofail label was moved from deployments and pods to just the pgcluster CRD to support autofail toggling
 * the storage configurations in *pgo.yaml* support the MatchLabels attribute for NFS storage, this will allow users to have more than a single NFS backend, when set, this label (key=value) will be used to match the labels on PVs when a PVC is created.
 * the UpdateCluster permission was added to the sample pgorole file to support the new pgo update CLI command, and also added to the pgoperm file
 * the pgo.yaml adds the PreferredFailoverNode setting, this is a Kube selector string (e.g. key=value).  This value if set, will cause fail-over targets to be preferred based on the node they run on if that node is in the set of *preferred* 
 * nodes based on this selector string.  For this to feature to be used, multiple replicas have to be in a ready state, and also at the same replication status.  If those conditions are not met, the default fail-over target selection is used.
 * the pgo.yaml file now includes a new storage configuration, XlogStorage, when set will cause the xlog volume to be allocated using this storage configuration, if not set, the PrimaryStorage configuration will be used.
 * the pgo.yaml file now includes a setting, AutofailReplaceReplica,  which will enable or disable whether a new replica is created as part of a fail-over, this is turned off by default.

See the github Release notes for the the features and other notes about a specific release.

