---
title: "Manual Upgrades"
date:
draft: false
weight: 100
---

## Manually Upgrading the Operator and PostgreSQL Clusters

In the event that the automated upgrade cannot be used, below are manual upgrade procedures for both PostgreSQL Operator 3.5 and 4.0 releases. These procedures will require action by the Operator administrator of your organization in order to upgrade to the current release of the Operator. Some upgrade steps are still automated within the Operator, but not all are possible with this upgrade method. As such, the pages below show the specific steps required to upgrade different versions of the PostgreSQL Operator depending on your current environment.

NOTE: If you are upgrading from Crunchy PostgreSQL Operator version 4.1.0 or later, the [Automated Upgrade Procedure](/upgrade/automatedupgrade) is recommended. If you are upgrading PostgreSQL 12 clusters, you MUST use the [Automated Upgrade Procedure](/upgrade/automatedupgrade). 

When performing a manual upgrade, it is recommended to upgrade to the latest PostgreSQL Operator available.

[Manual Upgrade - PostgreSQL Operator 3.5]( {{< relref "upgrade/manual/upgrade35.md" >}})

[Manual Upgrade - PostgreSQL Operator 4]( {{< relref "upgrade/manual/upgrade4.md" >}})

