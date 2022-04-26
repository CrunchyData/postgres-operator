### Delete test

#### Regular cluster delete

* Start a regular cluster
* Delete it
* Check that nothing remains.

#### Delete cluster after switchover

* Start a regular cluster with 2 replicas
* Trigger a switchover
* Delete it
* Check that primary pod terminated last
* Check that nothing remains

#### Delete a cluster that never started

* Start a cluster with a bad image
* Delete it
* Check that nothing remains
