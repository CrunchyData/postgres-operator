# Installing the Monitoring Infrastructure

This Helm chart installs the metrics deployment for Crunchy PostgreSQL Operator
by using its “pgo-deployer” container. Helm will setup the ServiceAccount, RBAC,
and ConfigMap needed to run the container as a Kubernetes Job. Then a job will
be created based on `helm` `install`, or `uninstall` to install or uninstall
metrics. After the job has completed the RBAC will be cleaned up.

## Prerequisites

- Helm v3
- Kubernetes 1.14+

## Getting the chart

Clone the `postgres-operator` repo:
```
git clone https://github.com/CrunchyData/postgres-operator.git
```

## Installing

```
cd postgres-operator/installers/metrics/helm
helm install metrics . -n pgo
```

## Uninstalling

```
cd postgres-operator/installers/metrics/helm
helm uninstall metrics -n pgo
```

## Configuraiton 

The following shows the configurable parameters that are relevant to the Helm
Chart. A full list of all Crunchy PostgreSQL Operator configuration options can
be found in the [documentation](https://access.crunchydata.com/documentation/postgres-operator/latest/installation/configuration/).

| Name | Default | Description |
| ---- | ------- | ----------- |
| fullnameOverride | "" |  |
| rbac.create | true | If false RBAC will not be created. RBAC resources will need to be created manually and bound to `serviceAccount.name` |
| rbac.useClusterAdmin | false | If enabled the ServiceAccount will be given cluster-admin privileges. |
| serviceAccount.create | true | If false a ServiceAccount will not be created. A ServiceAccount must be created manually. |
| serviceAccount.name | "" | Use to override the default ServiceAccount name. If serviceAccount.create is false this ServiceAccount will be used. |

{{% notice tip %}}
If installing into an OpenShift 3.11 or Kubernetes 1.11 cluster `rbac.useClusterAdmin` must be enabled.
{{% /notice %}}
