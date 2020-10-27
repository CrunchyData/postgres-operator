---
title: "Uninstalling PostgreSQL Operator"
date:
draft: false
weight: 40
---

# Uninstalling PostgreSQL Operator

The following assumes the proper [prerequisites are satisfied][ansible-prerequisites]
we can now uninstall the PostgreSQL Operator.

First, it is recommended to use the playbooks tagged with the same version
of the PostgreSQL Operator currently deployed.

With the correct playbooks acquired and prerequisites satisfied, simply run
the following command:

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=uninstall --ask-become-pass main.yml
```

## Deleting `pgo` Client

If variable `pgo_client_install` is set to `true` in the `values.yaml` file, the `pgo` client will also be removed when uninstalling.

Otherwise, the `pgo` client can be manually uninstalled by running the following command:

```
rm /usr/local/bin/pgo
```

[ansible-prerequisites]: {{< relref "/installation/other/ansible/prerequisites.md" >}}
