## Scaledown tests

This is a KUTTL version of a previous `TestScaleDown` test that was prone to flaky behavior; 
The KUTTL test captures the three test-cases enumerated in that test, and for ease of reading, 
all three tests exist in this folder, which necessitates a clean-up step after tests one and two. 
This tests makes extensive use of `status.instances` to make sure that the expected instances 
have the expected number of pods.

### From two sets to one set

* 00--create-cluster: create the cluster with two instance sets, one replica each
* 00-assert: check that the cluster exists with the expected status
* 01--update-cluster: update the cluster to remove one instance set
* 01-assert: check that the cluster exists with the expected status
* 02--delete-cluster

### From one set with multiple replicas to one set with one replica

* 10--create-cluster: create the cluster with one instance set with two replicas
* 10-assert: check that the cluster exists with the expected status
* 11-annotate: set the roles as labels on the pods
* 12--update-cluster: update the cluster to remove one replica
* 12-assert: check that the cluster exists with the expected status; and that the `master` pod that exists was the `master` before the scaledown
* 13--delete-cluster: delete the cluster

### From two sets with variable replicas to two set with one replica each

* 20--create-cluster: create the cluster with two instance sets, with two and one replica
* 20-assert: check that the cluster exists with the expected status
* 21--update-cluster: update the cluster to reduce the two-replica instance to one-replica
* 21-assert: check that the cluster exists with the expected status
