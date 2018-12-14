


# Design 

The *postgres-operator* design incorporates the following concepts:

 * adds Custom Resource Definitions for PostgreSQL to Kubernetes
 * adds controller logic that watches events on PostgreSQL resources
 * provides a command line client (*pgo*) and REST API for interfacing with the postgres-operator
 * provides for very customized deployments including container resources, storage configurations, and PostgreSQL custom configurations

<!--stackedit_data:
eyJoaXN0b3J5IjpbMTQ2MzQ4NTA1OSwtMTgyMzk5NV19
-->