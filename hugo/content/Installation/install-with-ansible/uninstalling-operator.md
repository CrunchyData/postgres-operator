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
ansible-playbook -i /path/to/inventory --tags=deprovision main.yml
```

If the Crunchy PostgreSQL Operator playbooks were installed using `yum`, use the
following commands:

```bash
export ANSIBLE_ROLES_PATH=/usr/share/ansible/roles/crunchydata

ansible-playbook -i /path/to/inventory --tags=deprovision \
    /usr/share/ansible/postgres-operator/playbooks/main.yml
```

## Deleting `pgo` Client

To remove the `pgo` client, simply run the following command:

```
rm /usr/local/bin/pgo
```
