---
title: "Uninstalling PostgreSQL Operator"
date:
draft: false
weight: 40
---

# Uninstalling PostgreSQL Operator

The following assumes the proper [prerequisites are satisfied](/installation/install-with-ansible/prereq/prerequisites/)
we can now deprovision the PostgreSQL Operator.

First, it is recommended to use the playbooks tagged with the same version
of the PostgreSQL Operator currently deployed.

With the correct playbooks acquired and prerequisites satisfied, simply run
the following command:

```bash
ansible-playbook -i /path/to/inventory --tags=deprovision --ask-become-pass main.yml
```

If the Crunchy PostgreSQL Operator playbooks were installed using `yum`, use the
following commands:

```bash
export ANSIBLE_ROLES_PATH=/usr/share/ansible/roles/crunchydata

ansible-playbook -i /path/to/inventory --tags=deprovision --ask-become-pass \
    /usr/share/ansible/postgres-operator/playbooks/main.yml
```

## Deleting `pgo` Client

If variable `pgo_client_install` is set to `true` in the `inventory` file, the `pgo` client will also be uninstalled when deprovisioning.

Otherwise, the `pgo` client can be manually uninstalled by running the following command:

```
rm /usr/local/bin/pgo
```
