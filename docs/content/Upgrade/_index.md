---
title: "Upgrade"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 80
---

## Upgrading the Operator
Various Operator releases will require action by the Operator administrator of your organization in order to upgrade to the next release of the Operator.  Some upgrade steps are automated within the Operator but not all are possible at this time.

This section of the documentation shows specific steps required to upgrade different versions of the Postgres Operator depending on your current environment.

[Upgrade Postgres Operator to 3.5] ( {{< relref "upgrade/upgradeto35.md" >}})

[Postgres Operator 3.5 Minor Version Upgrade] ( {{< relref "upgrade/upgrade35.md" >}})

[Upgrade Postgres Operator from 3.5 to 4.1] ( {{< relref "upgrade/upgrade35to4.md" >}})

[Upgrade Postgres Operator from 4.X to 4.3.0 (Bash)] ( {{< relref "upgrade/upgrade4xto42_bash.md" >}})

[Upgrade Postgres Operator from 4.X to 4.3.0 (Ansible)] ( {{< relref "upgrade/upgrade4xto42_ansible.md" >}})

[Upgrade Postgres Operator from 4.1.0 to a patch release] ( {{< relref "upgrade/upgrade41.md" >}})

## Upgrading A Postgres Cluster

Using the operator, it is possible to upgrade a postgres cluster in place.  When a pgo upgrade command is issued, and a --CCPImageTag is specified, the operator will upgrade each replica and the primary to the new CCPImageTag version. It is important to note that the postgres version of the new container should be compatible with the current running version. There is currently no version check done to ensure compatibility.

The upgrade is accomplished by updating the CCPImageTag version in the deployment, which causes the old pod to be terminated and a new pod created with the updated deployment specification.

When the upgrade starts, and if _autofail_ is enabled for the cluster, each replica is upgraded seqentially, waiting for the previous replica to go ready before updating the next. After the replicas complete, the primary is then upgraded to the new image. Please note that the upgrade process respects the _autofail_ setting as currently definied for the cluster being upgraded.  Therefore, if autofail is enabled when the primary deployment is updated, the cluster behaves as though the primary had failed and begins the failover process.  See _Automatic Failover_ in the _Overview_ section for more details about the PostgreSQL Operator failover process and expected behavior.

When the cluster is not in _autofail_ mode (i.e. autofail is disabled), the primary and all replicas are updated at the same time, after which they will remain in an "unready" status.  This is because when autofail is disabled, no attempt will be made to start the PostgreSQL databases contained within the primary or replica pods once the containers have been started following the update.  It is therefore necessary to re-enable autofail following a minor upgrade during which autofail was disabled in order to fully bring the cluster back online.

At this time, the backrest-repo container is not upgraded during this upgrade as it is part of the postgres operator release and is updated with the operator.

## Minor Upgrade Example

In this example, we are upgrading a cluster from PostgreSQL 11.6 to 11.7 using the `crunchy-postgres:centos7-11.7-4.3.0` container:

`pgo upgrade mycluster --ccp-image-tag=centos7-11.7-4.3.0`

For more information, please see the `pgo upgrade` documentation [here.] ( {{< relref "pgo-client/reference/pgo_upgrade.md" >}})
