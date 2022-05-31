# Streaming Standby Tests

The streaming standby test will deploy two clusters, one primary and one standby.
Both clusters are created in the same namespace to allow for easy connections
over the network.

This test scenario can be run without any specific Kubernetes environment
requirements. More standby tests can be added that will require access to a
cloud storage.
