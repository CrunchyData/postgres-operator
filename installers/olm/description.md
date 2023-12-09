[Crunchy Postgres for Kubernetes](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes), is the leading Kubernetes native
Postgres solution. Built on PGO, the Postgres Operator from Crunchy Data, Crunchy Postgres for Kubernetes gives you a declarative Postgres
solution that automatically manages your PostgreSQL clusters.

Designed for your GitOps workflows, it is [easy to get started](https://access.crunchydata.com/documentation/postgres-operator/latest/quickstart)
with Crunchy Postgres for Kubernetes. Within a few moments, you can have a production grade Postgres cluster complete with high availability, disaster
recovery, and monitoring, all over secure TLS communications. Even better, Crunchy Postgres for Kubernetes lets you easily customize your Postgres
cluster to tailor it to your workload!

With conveniences like cloning Postgres clusters to using rolling updates to roll out disruptive changes with minimal downtime, Crunchy Postgres
for Kubernetes is ready to support your Postgres data at every stage of your release pipeline. Built for resiliency and uptime, Crunchy Postgres
for Kubernetes will keep your Postgres cluster in a desired state so you do not need to worry about it.

Crunchy Postgres for Kubernetes is developed with many years of production experience in automating Postgres management on Kubernetes, providing
a seamless cloud native Postgres solution to keep your data always available.

Crunchy Postgres for Kubernetes is made available to users without an active Crunchy Data subscription in connection with Crunchy Data's
[Developer Program](https://www.crunchydata.com/developers/terms-of-use).
For more information, please contact us at [info@crunchydata.com](mailto:info@crunchydata.com).

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
- **PostgreSQL Major Version Upgrades**: Perform a [PostgreSQL major version upgrade][major-version-upgrade] declaratively.
- **Database Administration**: Easily deploy [pgAdmin4][pgadmin] to administer your PostgresClusters' databases.
  The automatic discovery of PostgresClusters ensures that you are able to seamlessly access any databases within your environment from the pgAdmin4 GUI.
- **Full Customizability**: Crunchy PostgreSQL for Kubernetes makes it easy to get your own PostgreSQL-as-a-Service up and running
  and fully customize your deployments, including:
    - Choose the resources for your Postgres cluster: [container resources and storage size][resize-cluster]. [Resize at any time][resize-cluster] with minimal disruption.
    - Use your own container image repository, including support `imagePullSecrets` and private repositories
    - [Customize your PostgreSQL configuration][customize-cluster]

and much more!

[backups]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/backups-disaster-recovery
[clone]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorials/backups-disaster-recovery/disaster-recovery
[customize-cluster]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorials/day-two/customize-cluster
[disaster-recovery]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/backups-disaster-recovery/disaster-recovery
[high-availability]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/day-two/high-availability
[major-version-upgrade]: https://access.crunchydata.com/documentation/postgres-operator/v5/guides/major-postgres-version-upgrade/
[monitoring]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/day-two/monitoring
[pool]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/basic-setup/connection-pooling
[provisioning]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/basic-setup/create-cluster
[resize-cluster]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorials/cluster-management/resize-cluster
[tls]: https://access.crunchydata.com/documentation/postgres-operator/latest/tutorials/day-two/customize-cluster#customize-tls

[k8s-anti-affinity]: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/#inter-pod-affinity-and-anti-affinity
[k8s-nodes]: https://kubernetes.io/docs/concepts/architecture/nodes/

[pgAdmin]: https://www.pgadmin.org/
[pgBackRest]: https://www.pgbackrest.org
[pgBouncer]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials/basic-setup/connection-pooling
[pgMonitor]: https://github.com/CrunchyData/pgmonitor

## Post-Installation

### Tutorial

Want to [learn more about the PostgreSQL Operator][tutorial]? Browse through the [tutorial][] to learn more about what you can do, [join the Discord server][discord] for community support, or check out the [PGO GitHub repo][ghrepo] to learn more about the open source Postgres Operator project that powers Crunchy Postgres for Kubernetes.

[tutorial]: https://access.crunchydata.com/documentation/postgres-operator/v5/tutorials
[discord]: https://discord.gg/a7vWKG8Ec9
[ghrepo]: https://github.com/CrunchyData/postgres-operator
