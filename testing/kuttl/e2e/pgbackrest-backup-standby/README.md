### pgBackRest backup-standby test

* 00: Create a cluster with 'backup-standby' set to 'y' but with only one replica.
* 01: Check the backup Job Pod logs for the expected error.
* 02: Update the cluster to have 2 replicas and verify that the cluster can initialize successfully and the backup job can complete.
