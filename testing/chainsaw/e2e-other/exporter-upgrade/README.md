The exporter-upgrade test makes sure that PGO updates an extension used for monitoring. This
avoids an error where a user might update to a new PG image with a newer extension, but with an
older extension operative.

Note: This test relies on two `crunchy-postgres` images with known, different `pgnodemx` extensions:
the image created in 00--cluster.yaml has `pgnodemx` 1.1; the image we update the cluster to in 
02--update-cluster.yaml has `pgnodemx` 1.3. 

00-01
This starts up a cluster with a purposely outdated `pgnodemx` extension. Because we want a specific
extension, the image used here is hard-coded (and so outdated it's not publicly available).

(This image is so outdated that it doesn't finish creating a backup with the current PGO, which is 
why the 00-assert.yaml only checks that the pod is ready; and why 01--check-exporter.yaml wraps the 
call in a retry loop.)

02-03
The cluster is updated with a newer (and hardcoded) image with a newer version of `pgnodemx`. Due
to the change made in https://github.com/CrunchyData/postgres-operator/pull/3400, this should no 
longer produce multiple errors.

Note: a few errors may be logged after the `exporter` container attempts to run the `pgnodemx`
functions but before the extension is updated. So this checks that there are no more than 2 errors,
since that was the observed maximum number of printed errors during manual tests of the check.

For instance, using these hardcoded images (with `pgnodemx` versions 1.1 and 1.3), those errors were: 

```
Error running query on database \"localhost:5432\": ccp_nodemx_disk_activity pq: query-specified return tuple and function return type are not compatible" 
Error running query on database \"localhost:5432\": ccp_nodemx_data_disk pq: query-specified return tuple and function return type are not compatible
```
