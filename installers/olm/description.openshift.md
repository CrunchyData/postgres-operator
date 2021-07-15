Crunchy PostgreSQL for OpenShift lets you run your own production-grade PostgreSQL-as-a-Service on OpenShift!

Powered by the Crunchy [Postgres Operator](https://github.com/CrunchyData/postgres-operator), Crunchy PostgreSQL
for OpenShift automates and simplifies deploying and managing open source PostgreSQL clusters on OpenShift by
providing the essential features you need to keep your PostgreSQL clusters up and running, including:

- **PostgreSQL Cluster Provisioning**: [Create, Scale, & Delete PostgreSQL clusters with ease][provisioning],
  while fully customizing your Pods and PostgreSQL configuration!
- **High-Availability**: Safe, automated failover backed by a [distributed consensus based high-availability solution][high-availability].
  Uses [Pod Anti-Affinity][k8s-anti-affinity] to help resiliency; you can configure how aggressive this can be!
  Failed primaries automatically heal, allowing for faster recovery time. You can even create regularly scheduled
  backups as well and set your backup retention policy
- **Disaster Recovery**: [Backups][backups] and [restores][disaster-recovery] leverage the open source [pgBackRest][] utility and
  [includes support for full, incremental, and differential backups as well as efficient delta restores][backups].
  Set how long you want your backups retained for. Works great with very large databases!
- **Monitoring**: [Track the health of your PostgreSQL clusters][monitoring] using the open source [pgMonitor][] library.
- **Clone**: [Create new clusters from your existing clusters or backups][clone] with efficient data cloning.
- **TLS**: All connections are over [TLS][tls]. You can also [bring your own TLS infrastructure][tls] if you do not want to use the provided defaults.
- **Connection Pooling**: Advanced [connection pooling][pool] support using [pgBouncer][].
- **Affinity and Tolerations**: Have your PostgreSQL clusters deployed to [Kubernetes Nodes][k8s-nodes] of your preference.
  Set your [pod anti-affinity][k8s-anti-affinity], node affinity, Pod tolerations and more rules to customize your deployment topology!
- **Full Customizability**: Crunchy PostgreSQL for Kubernetes makes it easy to get your own PostgreSQL-as-a-Service up and running
  and fully customize your deployments, including:
    - Choose the resources for your Postgres cluster: [container resources and storage size][resize-cluster]. [Resize at any time][resize-cluster] with minimal disruption.
    - Use your own container image repository, including support `imagePullSecrets` and private repositories
    - [Customize your PostgreSQL configuration][customize-cluster]

and much more!

[backups]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/backups/
[clone]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/disaster-recovery/#clone-a-postgres-cluster
[customize-cluster]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/customize-cluster/
[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/disaster-recovery/
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/high-availability/
[monitoring]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/monitoring/
[pool]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/connection-pooling/
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/create-cluster/
[resize-cluster]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/resize-cluster/
[tls]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial/customize-cluster/#customize-tls

[k8s-anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity
[k8s-nodes]: https://kubernetes.io/docs/concepts/architecture/nodes/

[pgBackRest]: https://www.pgbackrest.org
[pgBouncer]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorial/pgbouncer/
[pgMonitor]: https://github.com/CrunchyData/pgmonitor


## Post-Installation

### Tutorial

Want to [learn more about the PostgreSQL Operator][tutorial]? Browse through the [tutorial][] to learn more about what you can do!

[tutorial]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorial

