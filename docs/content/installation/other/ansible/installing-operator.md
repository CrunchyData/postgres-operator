---
title: "Installing PostgreSQL Operator"
date:
draft: false
weight: 21
---

# Installing

The following assumes the proper [prerequisites are satisfied][ansible-prerequisites]
we can now install the PostgreSQL Operator.

The commands should be run in the directory where the Crunchy PostgreSQL Operator
playbooks are stored.  See the `installers/ansible` directory in the Crunchy PostgreSQL Operator
project for the inventory file, values file, main playbook and ansible roles.

## Installing on Linux

On a Linux host with Ansible installed we can run the following command to install
the PostgreSQL Operator:

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=install --ask-become-pass main.yml
```

## Installing on macOS

On a macOS host with Ansible installed we can run the following command to install
the PostgreSQL Operator.

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=install --ask-become-pass main.yml
```

## Installing on Windows Ubuntu Subsystem

On a Windows host with an Ubuntu subsystem we can run the following commands to install
the PostgreSQL Operator.

```bash
ansible-playbook -i /path/to/inventory.yaml --tags=install --ask-become-pass main.yml
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

## Install the `pgo` Client

{{% notice info %}}
If TLS authentication was disabled during installation, please see the [TLS Configuration Page] ({{< relref "Configuration/tls.md" >}}) for additional configuration information.
{{% / notice %}}

During or after the installation of PGO: the Postgres Operator, download the `pgo` client set up script. This will help set up your local environment for using the Postgres Operator:

```
curl https://raw.githubusercontent.com/CrunchyData/postgres-operator/v{{< param operatorVersion >}}/installers/kubectl/client-setup.sh > client-setup.sh
chmod +x client-setup.sh
```

When the Postgres Operator is done installing, run the client setup script:

```
./client-setup.sh
```

This will download the `pgo` client and provide instructions for how to easily use it in your environment. It will prompt you to add some environmental variables for you to set up in your session, which you can do with the following commands:

```
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/pgo/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
export PGO_NAMESPACE=pgo
```

If you wish to permanently add these variables to your environment, you can run the following:

```
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/pgo/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/pgo/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/pgo/client.key"
export PGO_APISERVER_URL='https://127.0.0.1:8443'
export PGO_NAMESPACE=pgo
EOF

source ~/.bashrc
```

**NOTE**: For macOS users, you must use `~/.bash_profile` instead of `~/.bashrc`

## Verify `pgo` Connection

In a separate terminal we need to setup a port forward to the Crunchy PostgreSQL
Operator to ensure connection can be made outside of the cluster:

```bash
# If deployed to Kubernetes
kubectl port-forward -n pgo svc/postgres-operator 8443:8443

# If deployed to OpenShift
oc port-forward -n pgo svc/postgres-operator 8443:8443
```

You can subsitute `pgo` in the above examples with the namespace that you
deployed the PostgreSQL Operator into.

On a separate terminal verify the PostgreSQL client can communicate with the Crunchy PostgreSQL
Operator:

```bash
pgo version
```

If the above command outputs versions of both the client and API server, the Crunchy
PostgreSQL Operator has been installed successfully.

[ansible-prerequisites]: {{< relref "/installation/other/ansible/prerequisites.md" >}}
