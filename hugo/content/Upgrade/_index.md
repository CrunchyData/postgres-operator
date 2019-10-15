---
title: "Upgrade"
Latest Release: 4.1.0 {docdate}
draft: false
weight: 8
---

## Upgrading the Operator
Various Operator releases will require action by the Operator administrator of your organization in order to upgrade to the next release of the Operator.  Some upgrade steps are automated within the Operator but not all are possible at this time.

This section of the documentation shows specific steps required to upgrade different versions of the Postgres Operator depending on your current environment.

[Upgrade Postgres Operator to 3.5] ( {{< relref "upgrade/upgradeto35.md" >}})

[Postgres Operator 3.5 Minor Version Upgrade] ( {{< relref "upgrade/upgrade35.md" >}})

[Upgrade Postgres Operator from 3.5 to 4.1] ( {{< relref "upgrade/upgrade35to4.md" >}})

[Upgrade Postgres Operator from 4.0.1 to 4.1.0 (Bash)] ( {{< relref "upgrade/upgrade40to41_bash.md" >}})

[Upgrade Postgres Operator from 4.0.1 to 4.1.0 (Ansible)] ( {{< relref "upgrade/upgrade40to41_ansible.md" >}})

## Upgrading A Postgres Cluster

Using the operator, it is possible to upgrade a postgres cluster in place.  When a pgo upgrade command is issued, and a --CCPImageTag is specified, the operator will upgrade each replica and the primary to the new CCPImageTag version. It is important to note that the postgres version of the new container should be compatible with the current running version. There is currently no version check done to ensure compatibility.

The upgrade is accomplished by updating the CCPImageTag version in the deployment, which causes the old pod to be terminated and a new pod created with the updated deployment specification.

When the upgrade starts, each replica is upgraded seqentially, waiting for the previous replica to go ready before updating the next. After the replicas complete, the primary is then upgraded to the new image. The upgrade process respects the _autofail_ and the _AutofailReplaceReplica_ settings as provided in the pgo.yaml or as a command line flag, if applicable.

When the cluster is not in _autofail_ mode, the deployments simply create a new pod when updated, terminating the old one. When autofail is enabled and the primary deployment is updated, the cluster behaves as though the primary had failed and begins the failover process. See _Automatic Failover_ in the _Overview_ section for more details on this and replica replacement.

At this time, the backrest-repo container is not upgraded during this upgrade as it is part of the postgres operator release and is updated with the operator.

## Minor Upgrade Example

In this example, we are upgrading a cluster from PostgreSQL 11.4 to 11.5 using the `crunchy-postgres:centos7-11.5-2.4.2` container:

`pgo upgrade mycluster --ccp-image-tag=centos7-11.5-2.4.2`

For more information, please see the `pgo upgrade` documentation [here.] ( {{< relref "operatorcli/cli/pgo_upgrade.md" >}})
