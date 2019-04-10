---
title: "Uninstalling PostgreSQL Operator"
date:
draft: false
weight: 50
---

# Uninstalling PostgreSQL Operator

The following assumes the proper [prerequisites are satisfied](/getting-started/prerequisites)
we can now deprovision the PostgreSQL Operator.

First, it is recommended to use the playbooks tagged with the same version
of the PostgreSQL Operator currently deployed.

With the correct playbooks acquired and prerequisites satisfied, simply run
the following command:

```bash
ansible-playbook -i /path/to/inventory main.yml --tags=deprovision
```

## Deleting `pgo` Client

To remove the `pgo` client, simply run the following command:

```
rm /usr/local/bin/pgo
```
