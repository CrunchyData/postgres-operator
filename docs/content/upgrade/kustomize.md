---
title: "Upgrading PGO v5 Using Kustomize"
date:
draft: false
weight: 50
---

## Upgrading from PGO v5.0.0 Using Kustomize

Starting with PGO v5.0.1, both the Deployment and ServiceAccount created when installing PGO via
the installers in the
[Postgres Operator examples repository](https://github.com/CrunchyData/postgres-operator-examples)
have been renamed from `postgres-operator` to `pgo`.  As a result of this change, if using
Kustomize to install PGO and upgrading from PGO v5.0.0, the following step must be completed prior
to upgrading.  This will ensure multiple versions of PGO are not installed and running concurrently
within your Kubernetes environment.

Prior to upgrading PGO, first manually delete the PGO v5.0.0 `postgres-operator` Deployment and
ServiceAccount:

```bash
kubectl -n postgres-operator delete deployment,serviceaccount postgres-operator
```

Then, once both the Deployment and ServiceAccount have been deleted, proceed with upgrading PGO
by applying the new version of the Kustomize installer:

```bash
kubectl apply --server-side -k kustomize/install/default
```

## Upgrading from PGO v5.0.2 and Below

As a result of changes to pgBackRest dedicated repository host deployments in PGO v5.0.3
(please see the [PGO v5.0.3 release notes]({{< relref "../releases/5.0.3.md" >}}) for more details),
reconciliation of a pgBackRest dedicated repository host might become stuck with the following
error (as shown in the PGO logs) following an upgrade from PGO versions v5.0.0 through v5.0.2:

```bash
StatefulSet.apps \"hippo-repo-host\" is invalid: spec: Forbidden: updates to statefulset spec for fields other than 'replicas', 'template', 'updateStrategy' and 'minReadySeconds' are forbidden
```

If this is the case, proceed with deleting the pgBackRest dedicated repository host StatefulSet,
and PGO will then proceed with recreating and reconciling the dedicated repository host normally:

```bash
kubectl delete sts hippo-repo-host
```

Additionally, please be sure to update and apply all PostgresCluster custom resources in accordance
with any applicable spec changes described in the
[PGO v5.0.3 release notes]({{< relref "../releases/5.0.3.md" >}}).

## Upgrading from PGO v5.0.5 and Below

Starting in PGO v5.1, new pgBackRest features available in version 2.38 are used
that impact both the `crunchy-postgres` and `crunchy-pgbackrest` images. For any
clusters created before v5.0.6, you will need to update these image values
BEFORE upgrading to PGO {{< param operatorVersion >}}. These changes will need
to be made in one of two places, depending on your desired configuration.

If you are setting the image values on your `PostgresCluster` manifest,
you would update the images value as shown (updating the `image` values as
appropriate for your environment):

```yaml
apiVersion: postgres-operator.crunchydata.com/v1beta1
kind: PostgresCluster
metadata:
  name: hippo
spec:
  image: {{< param imageCrunchyPostgres >}}
  postgresVersion: {{< param postgresVersion >}}
...
  backups:
    pgbackrest:
      image: {{< param imageCrunchyPGBackrest >}}
...
```

After updating these values, you will apply these changes to your PostgresCluster
custom resources. After these changes are completed and the new images are in place,
you may update PGO to {{< param operatorVersion >}}.

Relatedly, if you are instead using the `RELATED_IMAGE` environment variables to
set the image values, you would instead check and update these as needed before
redeploying PGO.

For Kustomize installations, these can be found in the `manager` directory and
`manager.yaml` file. Here you will note various key/value pairs, these will need
to be updated before deploying PGO {{< param operatorVersion >}}. Besides updating the
`RELATED_IMAGE_PGBACKREST` value, you will also need to update the relevant
Postgres image for your environment. For example, if you are using PostgreSQL 14,
you would update the value for `RELATED_IMAGE_POSTGRES_14`. If instead you are
using the PostGIS 3.1 enabled PostgreSQL 13 image, you would update the value
for `RELATED_IMAGE_POSTGRES_13_GIS_3.1`.

For Helm deployments, you would instead need to similarly update your `values.yaml`
file, found in the `install` directory. There you will note a `relatedImages`
section, followed by similar values as mentioned above. Again, be sure to update
`pgbackrest` as well as the appropriate `postgres` value for your clusters.

Once there values have been properly verified, you may deploy PGO {{< param operatorVersion >}}.