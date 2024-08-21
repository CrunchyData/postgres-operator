backups enabled
- backup spec present
    - should we warn if annotation present?
- backup spec absent, sts present, annotation absent
    - if spec absent and we consume spec = bad

spec | sts | annotation | backupsEnabled | backupReconciliationAllowed
Y    |     |            = Y              | Y

N    | N   |            = N              | Y
N    | Y   | Y          = N              | Y

N    | Y   | N          = ~Y             | N
