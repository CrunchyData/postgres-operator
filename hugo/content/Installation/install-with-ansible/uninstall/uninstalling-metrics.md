---
title: "Uninstalling Metrics Stack"
date:
draft: false
weight: 50
---

# Uninstalling the Metrics Stack

The following assumes the proper [prerequisites are satisfied](/getting-started/prerequisites)
we can now deprovision the PostgreSQL Operator.

First, it is recommended to use the playbooks tagged with the same version
of the Metrics stack currently deployed.

With the correct playbooks acquired and prerequisites satisfied, simply run
the following command:

```bash
ansible-playbook -i /path/to/inventory main.yml --tags=deprovision-metrics
```
