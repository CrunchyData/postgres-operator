### pgBackRest Init test

* 00: Create a cluster with two PVC repos and set up for manual backups to go to the second; verify that the PVCs exist and that the backup job completed successfully
* 01: Run pgbackrest-initialization.sh, which checks that the status matches the expected status of `mixed` (because the second repo in the repo list has not yet been pushed to) and that there is only one full backup
* 02: Use `kubectl` to annotate the cluster to initiate a manual backup; verify that the job completed successfully
* 03: Rerun pgbackrest-initialization.sh, now expecting the status to be `ok` since both repos have been pushed to and there to be two full backups
