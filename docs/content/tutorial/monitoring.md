---
title: "Monitoring"
date:
draft: false
weight: 90
---

While having [high availability]({{< relref "tutorial/high-availability.md" >}}) and [disaster recovery]({{< relref "tutorial/disaster-recovery.md" >}}) systems in place helps in the event of something going wrong with your PostgreSQL cluster, monitoring helps you anticipate problems before they happen. Additionally, monitoring can help you diagnose and resolve issues that may cause degraded performance rather than downtime.

Let's look at how PGO allows you to enable monitoring in your cluster.

## Adding the Exporter Sidecar

Let's look at how we can add the Crunchy PostgreSQL Exporter sidecar to your cluster using the `kustomize/postgres` example in the [Postgres Operator examples](https://github.com/CrunchyData/postgres-operator-examples/fork) repository.

Monitoring tools are added using the `spec.monitoring` section of the custom resource. Currently, the only monitoring tool supported is the Crunchy PostgreSQL Exporter configured with [pgMonitor].

The only required attribute for adding the Exporter sidecar is to set `spec.monitoring.pgmonitor.exporter.image`. In the `kustomize/postgres/postgres.yaml` file, add the following YAML to the spec:

```
monitoring:
  pgmonitor:
    exporter:
      image: {{< param imageCrunchyExporter >}}
```

Save your changes and run:

```
kubectl apply -k kustomize/postgres
```

PGO will detect the change and add the Exporter sidecar to all Postgres Pods that exist in your cluster. PGO will also do the work to allow the Exporter to connect to the database and gather metrics that can be accessed using the [PGO Monitoring] stack.

## Accessing the Metrics

Once the Crunchy PostgreSQL Exporter has been enabled in your cluster, follow the steps outlined in [PGO Monitoring] to install the monitoring stack. This will allow you to deploy a [pgMonitor] configuration of [Prometheus], [Grafana], and [Alertmanager] monitoring tools in Kubernetes. These tools will be set up by default to connect to the Exporter containers on your Postgres Pods.

## Next Steps

Now that we can monitor our cluster, let's explore how [connection pooling]({{< relref "connection-pooling.md" >}}) can be enabled using PGO and how it is helpful.

[pgMonitor]: https://github.com/CrunchyData/pgmonitor
[Grafana]: https://grafana.com/
[Prometheus]: https://prometheus.io/
[Alertmanager]: https://prometheus.io/docs/alerting/latest/alertmanager/
[PGO Monitoring]: {{< relref "installation/monitoring/_index.md" >}}
