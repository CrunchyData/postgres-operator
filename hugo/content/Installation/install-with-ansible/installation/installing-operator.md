---
title: "Installing PostgreSQL Operator"
date:
draft: false
weight: 40
---

# Installing

The following assumes the proper [prerequisites are satisfied](/getting-started/prerequisites)
we can now install the PostgreSQL Operator.

## Installing on Linux

On a Linux host with Ansible installed we can run the following command to install 
the PostgreSQL Operator:

The following command should be run from the directory where the
`postgres-operator-playbooks` project is located:

```bash
ansible-playbook -i /path/to/inventory main.yml --tags=install --ask-become-pass
```

## Installing on MacOS

On a MacOS host with Ansible installed we can run the following command to install
the PostgreSQL Operator.

The following command should be run from the directory where the
`postgres-operator-playbooks` project is located:

```bash
ansible-playbook -i /path/to/inventory main.yml --tags=install --ask-become-pass
```

## Installing on Windows Ubuntu Subsystem

On a Windows host with an Ubuntu subsystem we can run the following commands to install 
the PostgreSQL Operator.

The following command should be run from the directory where the
`postgres-operator-playbooks` project is located:

```bash
ansible-playbook -i /path/to/inventory main.yml --tags=install --ask-become-pass
```

## Verifying the Installation

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

After the Crunchy PostgreSQL Operator has successfully been installed we will need 
to configure local environment variables before using the `pgo` client.

To configure the environment variables used by `pgo` run the following command:

Note: `<PGO_NAMESPACE>` should be replaced with the namespace the Crunchy PostgreSQL
Operator was deployed to.

```bash
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/<PGO_NAMESPACE>/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/<PGO_NAMESPACE>/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/<PGO_NAMESPACE>/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/<PGO_NAMESPACE>/client.pem"
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
kubectl port-forward <OPERATOR_POD_NAME> -n <OPERATOR_NAMESPACE> 8443:8443

# If deployed to OpenShift
oc port-forward <OPERATOR_POD_NAME> -n <OPERATOR_NAMESPACE> 8443:8443
```

On a separate terminal verify the `pgo` can communicate with the Crunchy PostgreSQL 
Operator:

```bash
pgo version
```

If the above command outputs versions of both the client and API server, the Crunchy 
PostgreSQL Operator has been installed successfully.
