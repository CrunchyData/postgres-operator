---
title: "Upgrade"
Latest Release: 4.0.1 {docdate}
draft: false
weight: 8
---

## Upgrading the Operator
Various Operator releases will require action by the Operator administrator of your organization in order to upgrade to the next release of the Operator.  Some upgrade steps are automated within the Operator but not all are possible at this time.

This section of the documentation shows specific steps required to the
latest version from the previous version.

### Upgrading to Version 3.5.0 From Previous Versions
 * For clusters created in prior versions that used pgbackrest, you
will be required to first create a pgbasebackup for those clusters,
and after upgrading to Operator 3.5, you will need to restore those clusters
from the pgbasebackup into a new cluster with *--pgbackrest* enabled, this
is due to the new pgbackrest shared repository being implemented in 3.5.  This
is a breaking change for anyone that used pgbackrest in Operator versions
prior to 3.5.
 * The pgingest CRD is removed. You will need to manually remove it from any deployments of the operator after upgrading to this version. This includes removing ingest related permissions from the pgorole file. Additionally, the API server now
removes the ingest related API endpoints.
 * Primary and replica labels are only applicable at cluster creation and are not updated after a cluster has executed a failover. A new *service-name* label is applied to PG cluster components to indicate whether a deployment/pod is a primary or replica. *service-name* is also the label now used by the cluster services to route with. This scheme allows for an almost immediate failover promotion and avoids the pod having to be bounced as part of a failover.  Any existing PostgreSQL clusters will need to be updated to specify them as a primary or replica using the new *service-name* labeling scheme.  
 * The autofail label was moved from deployments and pods to just the pgcluster CRD to support autofail toggling.
 * The storage configurations in *pgo.yaml* support the MatchLabels attribute for NFS storage. This will allow users to have more than a single NFS backend,. When set, this label (key=value) will be used to match the labels on PVs when a PVC is created.
 * The UpdateCluster permission was added to the sample pgorole file to support the new pgo update CLI command. It was also added to the pgoperm file.
 * The pgo.yaml adds the PreferredFailoverNode setting. This is a Kubernetes selector string (e.g. key=value).  This value if set, will cause fail-over targets to be preferred based on the node they run on if that node is in the set of *preferred*.
 * The ability to select nodes based on a selector string was added.  For this to feature to be used, multiple replicas have to be in a ready state, and also at the same replication status.  If those conditions are not met, the default fail-over target selection is used.
 * The pgo.yaml file now includes a new storage configuration, XlogStorage, which when set will cause the xlog volume to be allocated using this storage configuration. If not set, the PrimaryStorage configuration will be used.
 * The pgo.yaml file now includes a new storage configuration, BackrestStorage, will cause the pgbackrest shared repository volume to be allocated using this storage configuration. 
 * The pgo.yaml file now includes a setting, AutofailReplaceReplica, which will enable or disable whether a new replica is created as part of a fail-over. This is turned off by default.

See the GitHub Release notes for the features and other notes about a specific release.

### Upgrading a Cluster from Version 3.5.x to 4.0

This section will outline the procedure to upgrade a given cluster created using Postgres Operator 3.5.x to 4.0.

1) Create a new Centos/Redhat user with the same permissions are the existing user used to install the Crunchy Postgres Operator. This is necessary to avoid any issues with environment variable differences between 3.5 and 4.0.

2) For the cluster(s) you wish to upgrade, scale down any replicas, if necessary, then delete the cluster

	pgo delete cluster <clustername>

IMPORTANT NOTES:
Please note the name of each cluster, 
the namespace used, 
and be sure not to delete the associated PVCs or CRDs!

3) Delete the 3.5.x version of the operator by executing:

	$COROOT/deploy/cleanup.sh
	$COROOT/deploy/remove-crd.sh

4) Log in as your new Linux user and install the 4.0 Postgres Operator. Be sure to add the existing namespace to the list of watched namespaces (see the 'Getting Started->Design->Namespace' section of this document for more information)  and make sure to avoid overwriting any existing data storage.

5) Once the Operator is installed and functional, create a new 4.0 cluster with the same name as was used previously. This will allow the new cluster to utilize the existing PVCs.

	pgo create cluster <clustername> -n <namespace>

6) Manually update the old leftover Secrets to use the new label as defined in 4.0:

	kubectl label secret/<clustername>-postgres-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-primaryuser-secret pg-cluster=<clustername> -n <namespace>
	kubectl label secret/<clustername>-testuser-secret pg-cluster=<clustername> -n <namespace>

7) To verify cluster status, run 
	pgo test <clustername> -n <namespace>
Output should be similar to:
```
psql -p 5432 -h 10.104.74.189 -U postgres postgres is Working
psql -p 5432 -h 10.104.74.189 -U postgres userdb is Working
psql -p 5432 -h 10.104.74.189 -U primaryuser postgres is Working
psql -p 5432 -h 10.104.74.189 -U primaryuser userdb is Working
psql -p 5432 -h 10.104.74.189 -U testuser postgres is Working
psql -p 5432 -h 10.104.74.189 -U testuser userdb is Working
``` 
8) Scale up to the required number of replicas, as needed.
