---
title: "Upgrade to PGO 3.5 From Previous Versions"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---

## Upgrading to Version 3.5.0 From Previous Versions

This procedure will give instructions on how to upgrade to Postgres Operator 3.5

{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant data!

{{% / notice %}}

For clusters created in prior versions that used pgbackrest, you
will be required to first create a pgbasebackup for those clusters.

After upgrading to Operator 3.5, you will need to restore those clusters
from the pgbasebackup into a new cluster with `--pgbackrest` enabled. This
is due to the new pgbackrest shared repository being implemented in 3.5.  This
is a breaking change for anyone that used pgbackrest in Operator versions
prior to 3.5.

The pgingest CRD is removed in Operator 3.5. You will need to manually remove it from any deployments of the operator after upgrading to this version. This includes removing ingest related permissions from the pgorole file. Additionally, the API server now removes the ingest related API endpoints.

Primary and replica labels are only applicable at cluster creation and are not updated after a cluster has executed a failover. A new service-name label is applied to PG cluster components to indicate whether a deployment/pod is a primary or replica. service-name is also the label now used by the cluster services to route with. This scheme allows for an almost immediate failover promotion and avoids the pod having to be bounced as part of a failover.  Any existing PostgreSQL clusters will need to be updated to specify them as a primary or replica using the new service-name labeling scheme.  

The autofail label was moved from deployments and pods to just the pgcluster CRD to support autofail toggling.

The storage configurations in pgo.yaml support the MatchLabels attribute for NFS storage. This will allow users to have more than a single NFS backend,. When set, this label (key=value) will be used to match the labels on PVs when a PVC is created.

The UpdateCluster permission was added to the sample pgorole file to support the new pgo update CLI command. It was also added to the pgoperm file.

The pgo.yaml adds the PreferredFailoverNode setting. This is a Kubernetes selector string (e.g. key=value).  This value if set, will cause fail-over targets to be preferred based on the node they run on if that node is in the set of *preferred*.

The ability to select nodes based on a selector string was added.  For this to feature to be used, multiple replicas have to be in a ready state, and also at the same replication status.  If those conditions are not met, the default fail-over target selection is used.

The pgo.yaml file now includes a new storage configuration, XlogStorage, which when set will cause the xlog volume to be allocated using this storage configuration. If not set, the PrimaryStorage configuration will be used.

The pgo.yaml file now includes a new storage configuration, BackrestStorage, will cause the pgbackrest shared repository volume to be allocated using this storage configuration.

The pgo.yaml file now includes a setting, AutofailReplaceReplica, which will enable or disable whether a new replica is created as part of a fail-over. This is turned off by default.

See the GitHub Release notes for the features and other notes about a specific release.
