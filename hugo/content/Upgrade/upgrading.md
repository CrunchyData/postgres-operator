
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
 * primary and replica labels are only applicable at cluster creation and not updated after a cluster has executed a failover, a new service-name label is applied to PG cluster components to indidate whether a deployment/pod is a primary or replica, service-name is also the label now used by the cluster services to route with.  This scheme allows for an almost immediate failover promotion and avoids the pod having to be bounced as part of a failover.  Any existing
PG clusters will need to be updated to specify them as a primary or replica using the new service-name labeling scheme.  A sample upgrade script is included in the bin directory name upgrade-to-35.sh, you would run this script
to upgrade any existing clusters to the new labeling scheme whereby you can run a failover on existing PG clusters deployed prior to 3.5.0.
 * the autofail label was moved from deployments and pods to just the pgcluster CRD to support autofail toggling
 * the storage configurations in pgo.yaml support the MatchLabels attribute for NFS storage, this will allow users to have more than a single NFS backend, when set, this label (key=value) will be used to match the labels on PVs when a PVC is created.
 * the UpdateCluster permission was added to the sample pgorole file to support the new pgo update CLI command, and also added to the pgoperm file

