---
title: "Uninstalling the Monitoring Infrastructure"
date:
draft: false
weight: 41
---

# Uninstalling the Monitoring Infrastructure

The following assumes the proper [prerequisites are satisfied][ansible-prerequisites]
we can now uninstall the PostgreSQL Operator Monitoring infrastructure.

First, it is recommended to use the playbooks tagged with the same version
of the Metrics infratructure currently deployed.

With the correct playbooks acquired and prerequisites satisfied, simply run
the following command:

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=uninstall-metrics main.yml
```

[ansible-prerequisites]: {{< relref "/installation/metrics/other/ansible/metrics-prerequisites.md" >}}
