---
title: "PostgreSQL Operator Namespace Considerations Overview"
date:
draft: false
weight: 6
---

## Kubernetes Namespaces and the PostgreSQL Operator

In Kubernetes, namespaces provide the user the ability to divide cluster resources between multiple users (via resource quota).  

The PostgreSQL Operator makes use of the Kubernetes Namespace support in order to define the Namespace to which the PostgreSQL Operator will deploy PostgreSQL clusters, enabling users to more easily allocate Kubernetes resources to specific areas within their business (users, projects, departments).  

#### Namespaces Applied to Organizational Requirements

Prior to version PostgreSQL Operator 4.0, the PostgreSQL Operator could only be deployed with a Namespace deployment pattern where both the PostgreSQL Operator and the PostgreSQL Clusters it deployed existed within a single Kubernetes namespace. 

With the PostgreSQL Operator 4.0 release, the operator now supports a variety of Namespace deployment patterns, including:

* **OwnNamespace** Operator and PostgreSQL clusters deployed to the same Kubernetes Namespace

* **SingleNamespace and MultiNamespace** Operator and PostgreSQL clusters deployed to a predefined set of Kubernetes Namespaces

* **AllNamespaces** Operator deployed into a single Kubernetes Namespace but watching all Namespaces on a Kubernetes cluster

#### Configuration of the Namespace to which PostgreSQL Operator is Deployed

In order to configure the Kubernetes Namespace within which the PostgreSQL Operator will run, it is necessary to configure the PGO_OPERATOR_NAMESPACE environment variable.  Both the Ansible and Bash installation method enable you to modify this PGO_OPERATOR_NAMESPACE environment variable in connection with the PostgreSQL Operator installation. 

#### Configuration of the Namespaces to which PostgreSQL Operator will Deploy PostgreSQL Clusters 

At startup time, the PostgreSQL Operator determines the Kubernetes Namespaces to which it will be able to deploy and administer PostgreSQL databases and clusters. The Kubernetes Namespace that the PostgreSQL Operator will be able to service is determined at startup time by the NAMESPACE environment variable.  The NAMESPACE variable is set as part of the PostgreSQL Operator installation process.  The format of the NAMESPACE value in the PostgreSQL Operator is modeled after the Operator Lifecycle Manager project. 



