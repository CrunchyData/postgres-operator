---
title: "FAQ"
date:
draft: false
weight: 105

aliases:
 - /contributing
---

## Project FAQ

### What is The PGO Project?

The PGO Project is the open source project associated with the development of [PGO](https://github.com/CrunchyData/postgres-operator), the [Postgres Operator](https://github.com/CrunchyData/postgres-operator) for Kubernetes from [Crunchy Data](https://www.crunchydata.com).

PGO is a [Kubernetes Operator](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/), providing a declarative solution for managing your PostgreSQL clusters.  Within a few moments, you can have a Postgres cluster complete with high availability, disaster recovery, and monitoring, all over secure TLS communications.

PGO is the upstream project from which [Crunchy PostgreSQL for Kubernetes](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes/) is derived. You can find more information on Crunchy PostgreSQL for Kubernetes [here](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes/).

### What’s the difference between PGO and Crunchy PostgreSQL for Kubernetes?

PGO is the Postgres Operator from Crunchy Data. It developed pursuant to the PGO Project and is designed to be a frequently released, fast-moving project where all new development happens.

[Crunchy PostgreSQL for Kubernetes](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes/) is produced by taking selected releases of PGO, combining them with Crunchy Certified PostgreSQL and PostgreSQL containers certified by Crunchy Data, maintained for commercial support, and made available to customers as the Crunchy PostgreSQL for Kubernetes offering.

### Where can I find support for PGO?

The community can help answer questions about PGO via the [PGO mailing list](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join).

Information regarding support for PGO is available in the [Support]({{< relref "support/_index.md" >}}) section of the PGO documentation, which you can find [here]({{< relref "support/_index.md" >}}).

For additional information regarding commercial support and Crunchy PostgreSQL for Kubernetes, you can [contact Crunchy Data](https://www.crunchydata.com/contact/).

### Under which open source license is PGO source code available?

The PGO source code is available under the [Apache License 2.0](https://github.com/CrunchyData/postgres-operator/blob/master/LICENSE.md).

### Where are the release tags for PGO v5?

With PGO v5, we've made some changes to our overall process. Instead of providing quarterly release
tags as we did with PGO v4, we're focused on ongoing active development in the v5 primary
development branch (`master`, which will become `main`).  Consistent with our practices in v4,
previews of stable releases with the release tags are made available in the
[Crunchy Data Developer Portal](https://www.crunchydata.com/developers).

These changes allow for more rapid feature development and releases in the upstream PGO project,
while providing
[Crunchy Postgres for Kubernetes](https://www.crunchydata.com/products/crunchy-postgresql-for-kubernetes/)
users with stable releases for production use.

To the extent you have constraints specific to your use, please feel free to reach out on
[info@crunchydata.com](mailto:info@crunchydata.com) to discuss how we can address those
specifically.

### How can I get involved with the PGO Project?

PGO is developed by the PGO Project. The PGO Project that welcomes community engagement and contribution.

The PGO source code and community issue trackers are hosted at [GitHub](https://github.com/CrunchyData/postgres-operator).

For community questions and support, please sign up for the [PGO mailing list](https://groups.google.com/a/crunchydata.com/forum/#!forum/postgres-operator/join).

For information regarding contribution, please review the contributor guide [here](https://github.com/CrunchyData/postgres-operator/blob/master/CONTRIBUTING.md).

Please register for the [Crunchy Data Developer Portal mailing list](https://www.crunchydata.com/developers/newsletter) to receive updates regarding Crunchy PostgreSQL for Kubernetes releases and the [Crunchy Data newsletter](https://www.crunchydata.com/newsletter/) for general updates from Crunchy Data.

### Where do I report a PGO bug?

The PGO Project uses GitHub for its [issue tracking](https://github.com/CrunchyData/postgres-operator/issues/new/choose). You can file your issue [here](https://github.com/CrunchyData/postgres-operator/issues/new/choose).

### How often is PGO released?

The PGO team currently plans to release new builds approximately every few weeks. The PGO team will flag certain builds as “stable” at their discretion. Note that the term “stable” does not imply fitness for production usage or any kind of warranty whatsoever.
