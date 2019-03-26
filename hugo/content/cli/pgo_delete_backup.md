---
title: "pgo_delete_backup"
---
## pgo delete backup

Delete a backup

### Synopsis

Delete a backup. For example:
    
    pgo delete backup mydatabase

```
pgo delete backup [flags]
```

### Options

```
  -h, --help   help for backup
```

### Options inherited from parent commands

```
      --apiserver-url string     The URL for the PostgreSQL Operator apiserver.
      --debug                    Enable debugging when true.
  -n, --namespace string         The namespace to use for pgo requests.
      --pgo-ca-cert string       The CA Certificate file path for authenticating to the PostgreSQL Operator apiserver.
      --pgo-client-cert string   The Client Certificate file path for authenticating to the PostgreSQL Operator apiserver.
      --pgo-client-key string    The Client Key file path for authenticating to the PostgreSQL Operator apiserver.
```

### SEE ALSO

* [pgo delete](/cli/pgo_delete/)	 - Delete a backup,   cluster, pgbouncer, pgpool, label, policy, upgrade, or user

