### Root Certificate Ownership Test

This Kuttl routine runs through the following steps:

#### Create two clusters and verify the root certificate secret ownership

- 00: Creates the two clusters and verifies they and the root cert secret exist
- 01: Check that the secret shows both clusters as owners

#### Delete the first cluster and verify the root certificate secret ownership

- 02: Delete the first cluster, assert that the second cluster and the root cert
secret are still present and that the first cluster is not present
- 03: Check that the secret shows the second cluster as an owner but does not show
the first cluster as an owner

#### Delete the second cluster and verify the root certificate secret ownership

- 04: Delete the second cluster, assert that both clusters are not present
- 05: Check the number of clusters in the namespace. If there are any remaining
clusters, ensure that the secret shows neither the first nor second cluster as an
owner. If there are no clusters remaining in the namespace, ensure the root cert
secret has been deleted.
