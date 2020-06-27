# Crunchy PostgreSQL Operator

This Helm chart uses the `pgo-deployer` container to install the Crunchy PostgreSQL
Operator. Helm will setup the ServiceAccount, RBAC, and ConfigMap needed to
run the container as a Kubernetes Job. Then a job will be created based on `helm`
`install`, `upgrade`, or `uninstall`. After the job has completed the RBAC will
be cleaned up.

## Prerequisites

- Helm v3.2.3
- Kubernetes 1.14+

## Getting the chart

Clone the `postgres-operator` repo:
```
git clone https://github.com/CrunchyData/postgres-operator.git
```

## Installing

```
cd postgres-operator/installers/helm
helm install postgres-operator . -n pgo
```

## Upgrading

```
cd postgres-operator/installers/helm
helm upgrade postgres-operator . -n pgo
```

## Uninstalling

```
cd postgres-operator/installers/helm
helm uninstall postgres-operator . -n pgo
```

## Configuraiton 

The following shows the configurable parameters that are relevant to the Helm
Chart. A full list of all Crunchy PostgreSQL Operator configuration options can
be found in the [documentation](https://crunchydata.github.io/postgres-operator/latest/installation/configuration/).

| Name | Default | Description |
| ---- | ------- | ----------- |
| fullnameOverride | "" |  |
| rbac.create | true | If false rbac will not be created. RBAC resources will need to be created manually and bound to `serviceAccount.name` |
| serviceAccount.create | true | If false a service account will not be created. ServiceAccount will need to be created manually |
| serviceAccount.name | "" | Use to override the default service account name. If serviceAccount.create is false this service account will be used. |