## Optional backups

### Steps

00-02. Create cluster without backups, check that expected K8s objects do/don't exist, e.g., repo-host sts doesn't exist; check that the archive command is `true`

03-06. Add data and a replica; check that the data successfully replicates to the replica.

10-11. Update cluster to add backups, check that expected K8s objects do/don't exist, e.g., repo-host sts exists; check that the archive command is set to the usual

20-21. Update cluster to remove backups but without annotation, check that no changes were made, including to the archive command

22-25. Annotate cluster to remove existing backups, check that expected K8s objects do/don't exist, e.g., repo-host sts doesn't exist; check that the archive command is `true`
