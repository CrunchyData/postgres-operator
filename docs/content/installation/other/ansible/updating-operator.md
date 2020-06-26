---
title: "Updating PostgreSQL Operator"
date:
draft: false
weight: 30
---

# Updating

Updating the Crunchy PostgreSQL Operator is essential to the lifecycle management
of the service.  Using the `update` flag will:

* Update and redeploy the operator deployment
* Recreate configuration maps used by operator
* Remove any deprecated objects
* Allow administrators to change settings configured in the `values.yaml`
* Reinstall the `pgo` client if a new version is specified

The following assumes the proper [prerequisites are satisfied][ansible-prerequisites]
we can now update the PostgreSQL Operator.

The commands should be run in the directory where the Crunchy PostgreSQL Operator
playbooks is stored.  See the `ansible` directory in the Crunchy PostgreSQL Operator
project for the inventory file, values file, main playbook and ansible roles.

## Updating on Linux

On a Linux host with Ansible installed we can run the following command to update  
the PostgreSQL Operator:

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=update --ask-become-pass main.yml
```

## Updating on macOS

On a macOS host with Ansible installed we can run the following command to update  
the PostgreSQL Operator.

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=update --ask-become-pass main.yml
```

## Updating on Windows Ubuntu Subsystem

On a Windows host with an Ubuntu subsystem we can run the following commands to update  
the PostgreSQL Operator.

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=update --ask-become-pass main.yml
```

## Verifying the Update

This may take a few minutes to deploy.  To check the status of the deployment run
the following:

```bash
# Kubernetes
kubectl get deployments -n <NAMESPACE_NAME>
kubectl get pods -n <NAMESPACE_NAME>

# OpenShift
oc get deployments -n <NAMESPACE_NAME>
oc get pods -n <NAMESPACE_NAME>
```

## Configure Environment Variables

After the Crunchy PostgreSQL Operator has successfully been updated we will need
to configure local environment variables before using the `pgo` client.

To configure the environment variables used by `pgo` run the following command:

Note: `<PGO_NAMESPACE>` should be replaced with the namespace the Crunchy PostgreSQL
Operator was deployed to.
Also, if TLS was disabled, or if the port was changed, update PGO_APISERVER_URL accordingly.

```bash
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/<PGO_NAMESPACE>/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/<PGO_NAMESPACE>/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/<PGO_NAMESPACE>/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/<PGO_NAMESPACE>/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
EOF
```

Apply those changes to the current session by running:

```bash
source ~/.bashrc
```

## Verify `pgo` Connection

In a separate terminal we need to setup a port forward to the Crunchy PostgreSQL
Operator to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n pgo svc/postgres-operator 8443:8443

# If deployed to OpenShift
oc port-forward -n pgo svc/postgres-operator 8443:8443
```
In the above examples, you can substitute `pgo` for the namespace that you
deployed the PostgreSQL Operator into.

On a separate terminal verify the PostgreSQL Operator client can communicate
with the PostgreSQL Operator:

```bash
pgo version
```

If the above command outputs versions of both the client and API server, the Crunchy
PostgreSQL Operator has been updated successfully.

[ansible-prerequisites]: {{< relref "/installation/other/ansible/prerequisites.md" >}}
