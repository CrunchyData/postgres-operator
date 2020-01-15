
---
title: "Compatibility Requirements"
Latest Release: 4.2.1 {docdate}
draft: false
weight: 1
---

## Container Dependencies

The Operator depends on the Crunchy Containers and there are
version dependencies between the two projects. Below are the operator releases and their dependent container release. For reference, the Postgres and PgBackrest versions for each container release are also listed.

| Operator Release   |      Container Release      | Postgres | PgBackrest Version
|:----------|:-------------|:------------|:--------------
| 4.2.1 | 4.2.1  | 12.1 | 2.20 |
|||11.6|2.20|
|||10.11|2.20|
|||9.6.16|2.20|
|||9.5.20|2.20|
||||
| 4.2.0 | 4.2.0  | 12.1 | 2.20 |
|||11.6|2.20|
|||10.11|2.20|
|||9.6.16|2.20|
|||9.5.20|2.20|
||||
| 4.1.1 | 4.1.1  | 12.1 | 2.18 |
|||11.6|2.18|
|||10.11|2.18|
|||9.6.16|2.18|
|||9.5.20|2.18|
||||
| 4.1.0 | 2.4.2  | 11.5 | 2.17 |
|||10.10| 2.17|
|||9.6.15|2.17|
|||9.5.19|2.17|
||||
| 4.0.1 | 2.4.1  | 11.4 | 2.13 |
|||10.9| 2.13|
|||9.6.14|2.13|
|||9.5.18|2.13|
||||
| 4.0.0 | 2.4.0  | 11.3 | 2.13 |
|||10.8| 2.13|
|||9.6.13|2.13|
|||9.5.17|2.13|
||||
| 3.5.4 | 2.3.3 | 11.4| 2.13 |
|||10.9| 2.13|
|||9.6.14|2.13|
|||9.5.18|2.13|
||||
| 3.5.3 | 2.3.2 | 11.3| 2.13 |
|||10.8| 2.13|
|||9.6.13|2.13|
|||9.5.17|2.13|
||||
| 3.5.2 | 2.3.1  | 11.2| 2.10 |
|||10.7| 2.10|
|||9.6.12|2.10|
|||9.5.16|2.10|

Features sometimes are added into the underlying Crunchy Containers
to support upstream features in the Operator thus dictating a
dependency between the two projects at a specific version level.

## Operating Systems

The Operator is developed on both Centos 7 and RHEL 7 operating systems.  The
underlying containers are designed to use either Centos 7 or RHEL 7 as the base
container image.

Other Linux variants are possible but are not supported at this time.

## Kubernetes Distributions

The Operator is designed and tested on Kubernetes and Openshift Container Platform.

## Storage

The Operator is designed to support HostPath, NFS, and Storage Classes for
persistence.  The Operator does not currently include code specific to
a particular storage vendor.

## Releases

The Operator is released on a quarterly basis often to coincide with Postgres releases.

There are pre-release and or minor bug fix releases created on an as-needed basis.
