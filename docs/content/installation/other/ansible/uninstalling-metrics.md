---
title: "Uninstalling Metrics Stack"
date:
draft: false
weight: 41
---

# Uninstalling the Metrics Stack

The following assumes the proper [prerequisites are satisfied][ansible-prerequisites]
we can now uninstall the PostgreSQL Operator Metrics Infrastructure.

First, it is recommended to use the playbooks tagged with the same version
of the Metrics stack currently deployed.

With the correct playbooks acquired and prerequisites satisfied, simply run
the following command:

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=uninstall-metrics main.yml
```

[ansible-prerequisites]: {{< relref "/installation/other/ansible/prerequisites.md" >}}
