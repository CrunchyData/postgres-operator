
---
title: "Compatibility Requirements"
Latest Release: 4.0.1 {docdate}
draft: false
weight: 1
---

## Container Dependencies

The Operator depends on the Crunchy Containers and there are 
version dependencies between the two projects.

| Operator Release   |      Container Release      |
|:----------|:-------------|
| 4.0.1 | 2.4.1  |
| 3.5.2 | 2.3.1  |

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

