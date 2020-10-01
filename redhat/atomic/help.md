= postgres-operator (1)
Crunchy Data
December 23, 2019

== NAME
postgres-operator - Trusted open-source PostgreSQL-as-a-Service

== DESCRIPTION
The Crunchy PostgreSQL Operator automates and simplifies deploying and managing
open source PostgreSQL clusters on Kubernetes and other Kubernetes-enabled
platforms by providing the essential features you need to keep your PostgreSQL
clusters up and running, including:

- PostgreSQL Cluster Provisioning
- High-Availability
- Disaster Recovery
- Monitoring
- PostgreSQL User Management
- Upgrade Management
- Advanced Replication Support
- Clone
- Connection Pooling
- Node Affinity
- Scheduled Backups
- Multi-Namespace Support

and more.

== USAGE
For more information on the PostgreSQL Operator, see the official
[PostgreSQL Operator Documentation](https://access.crunchydata.com/documentation/postgres-operator/)

== LABELS
The starter container includes the following LABEL settings:

That atomic command runs the Docker command set in this label:

`Name=`

The registry location and name of the image. For example, Name="registry.developers.crunchydata.com/crunchydata/postgres-operator".

`Version=`

The Red Hat Enterprise Linux version from which the container was built. For example, Version="7.7"

`Release=`

The specific release number of the container. For example, Release="4.5.0"
