---
title: "User & Roles"
date:
draft: false
weight: 800
---

## User Roles in the PostgreSQL Operator

The PostgreSQL Operator, when used in conjunction with the associated PostgreSQL Containers and Kubernetes, provides you with the ability to host your own open source, Kubernetes
native PostgreSQL-as-a-Service infrastructure.  

In installing, configuring and operating the PostgreSQL Operator as a PostgreSQL-as-a-Service capability, the following user roles will be required:


|Role       |     Applicable Component  | Authorized Privileges and Functions Performed |
|-----------|---------------------------|-----------------------------------------------|
|Platform Admininistrator (Privileged User)| PostgreSQL Operator | The Platform Admininistrator is able to control all aspects of the PostgreSQL Operator functionality, including: provisioning and scaling clusters, adding PostgreSQL Administrators and PostgreSQL Users to clusters, setting PostgreSQL cluster security privileges, managing other PostgreSQL Operator users, and more. This user can have access to any database that is deployed and managed by the PostgreSQL Operator.|
|Platform User | PostgreSQL Operator | The Platform User  has access to a limited subset of PostgreSQL Operator functionality that is defined by specific RBAC rules. A Platform Administrator manages the specific permissions for an Platform User specific permissions. A Platform User only receives a permission if its is explicitly granted to them.|
|PostgreSQL Administrator(Privileged Account) | PostgreSQL Containers | The PostgreSQL Administrator is the equivalent of a PostgreSQL superuser (e.g. the "postgres" user) and can perform all the actions that a PostgreSQL superuser is permitted to do, which includes adding additional PostgreSQL Users, creating databases within the cluster.|
|PostgreSQL User|PostgreSQL Containers | The PostgreSQL User has access to a PostgreSQL Instance or Cluster but must be granted explicit permissions to perform actions in PostgreSQL based upon their role membership. |

As indicated in the above table, both the Operator Administrator and the PostgreSQL Administrators represent privilege users with components within the PostgreSQL Operator.

### Platform Administrator

For purposes of this User Guide, the "Platform Administrator" is a  Kubernetes system user with PostgreSQL Administrator privileges and has PostgreSQL Operator admin rights.  While
PostgreSQL Operator admin rights are not required, it is helpful to have admin rights to be able to verify that the installation completed successfully.  The Platform Administrator
will be responsible for managing the installation of the Crunchy PostgreSQL Operator service in Kubernetes. That installation can be on RedHat OpenShift 3.11+, Kubeadm, or even
Google’s Kubernetes Engine.

### Platform User

For purposes of this User Guide, a "Platform User" is a Kubernetes system user and has PostgreSQL Operator admin rights.  While admin rights are not required for a typical user,
testing out functiontionality will be easier, if you want to limit functionality to specific actions section 2.4.5 covers roles. The Platform User is anyone that is interacting with
the Crunchy PostgreSQL Operator service in Kubernetes via the PGO CLI tool.  Their rights to carry out operations using  the PGO CLI tool is governed by PGO Roles(discussed in more
detail later) configured by the Platform Administrator. If this is you, please skip to section 2.3.1 where we cover configuring and installing PGO.

### PostgreSQL User

In the context of the PostgreSQL Operator, the "PostgreSQL User" is any person interacting with the PostgreSQL database using database specific connections, such as a language
driver or a database management GUI.

The default PostgreSQL instance installation via the PostgreSQL Operator comes with the following users:

|Role name      |                         Attributes                             |
----------------|----------------------------------------------------------------|
|postgres       | Superuser, Create role, Create DB, Replication, Bypass RLS     |
|primaryuser    | Replication                                                    |
|testuser       |                                                                |

The postgres user will be the admin user for the database instance.  The primary user is used for replication between primary and replicas.  The testuser is a normal user that has
access to the database “userdb” that is created for testing purposes.
