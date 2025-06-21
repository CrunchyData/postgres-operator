# Exporter connection on standby cluster

The exporter standby test will deploy two clusters, one primary and one standby.
Both clusters have monitoring enabled and are created in the same namespace to
allow for easy connections over the network.

The `ccp_monitoring` password for both clusters are updated to match allowing
the exporter on the standby cluster to query postgres using the proper `ccp_monitoring`
password.
