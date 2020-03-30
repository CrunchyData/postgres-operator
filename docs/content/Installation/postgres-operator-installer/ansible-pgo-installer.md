---
title: "Ansible Roles for pgo-installer"
date:
draft: false
weight: 20
---

The ansible roles for the `pgo-installer` can be used to setup and run the
install jobs with the `pgo-installer` image. Ansible will perform the steps
oulined in the [Deploy with PostgreSQL Operator
Installer]({{< relref "/installation/postgres-operator-installer" >}}).

## Prerequisites
The following is required prior to installing Crunchy PostgreSQL Operator 
using Ansible:

* [postgres-operator  playbooks](https://github.com/CrunchyData/postgres-operator/) source code for the target version
* Ansible 2.8.0+

### Updating the Inventory file
The PostgreSQL Operator Installer requires an inventory file to be installed.
This inventory file must be created as a configmap that is mounted by the
`pgo-installer` image. Once mounted, the file can be used to configure how the
operator will function when deployed into Kubernetes. 

An example inventory file can be found here:
`$PGOROOT/installers/ansible/inventory`  

Please reference the [Configuring the Inventory File]({{< relref "/installation/install-with-ansible/prerequisites#configuring-the-inventory-file" >}})
documentation as you update the inventory file. 

#### PostgreSQL Operator Installer Specific Inventory Options
The PostgreSQL Operator Installer has settings defined in the example inventory
file that are not referenced in the [Configuring the Inventory File]({{< relref "/installation/install-with-ansible/prerequisites#configuring-the-inventory-file" >}})
section of the documentation. 

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `kubernetes_in_cluster` | false | **Required** | Set to true allow the installer to run from within a Kubernetes cluster. This must be true to use the `pgo-installer`. |
| `use_cluster_admin` | false | **Required** | Set to true allow the installer to use cluster-admin to setup cluster-wide resources. This must be true to use the `pgo-installer`|
| `pgo_installer_environment` | `kubernetes` | **Required** | Specifies if the Ansible Roles for PGO-Installer should use `kubectl` or `oc`. Options: `kubernetes`, `openshift` |

You have the option to manually create the resources needed to run the
PostgreSQL Operator Installer. If you manually create the resources you can
disable their creation and provide the name to your resource using the following
options.

| Name | Default | Required | Description |
|------|---------|----------|-------------|
| `pgo_installer_namespace` | `pgo-install` | | Defines the namespace in which the install job will run. |
| `pgo_installer_sa` | `pgo-installer-sa` | | Defines the name of the `serviceaccount` used by the `pgo-installer`. |
| `pgo_installer_crb` | `pgo_installer_crb` | | Defines the name of the `clusterrolebinding` that is given to the `pgo_installer_sa` service account. |
| `pgo_installer_configmap` | `pgo-installer-inventory` | | Defines the name of the `configmap` that is mounted by the `pgo-installer` and stores the inventory file for the PostgreSQL Operator install. |
| `create_pgo_installer_namespace` | false | | Enables creation of the `pgo_installer_namespace` |
| `create_pgo_installer_service_account` | false | | Enables the creation of the `pgo_installer_sa`. This `serviceaccount` is only created if `use_cluster_admin` is true. |
| `create_pgo_installer_clusterrolebinding` | false | |  Enables thecreation of the `pgo_installer_crb`. This `clusterrolebinding` is only created if `use_cluster_admin` is true. |
| `create_pgo_installer_configmap` | false | |Enables the creation of the `pgo_installer_configmap` |

### Setting Up Resources 
The `pgo-installer` jobs need the resources outlined in the 
[Deploy with PostgreSQL Operator Installer (pgo-installer)]({{< relref "/installation/postgres-operator-installer" >}}) 
section of the documentation to run. These resources can be setup using the ansible roles by
enabling their creation in the inventory file. If these options are not enabled,
you will need to manually create the resources.

### Running the Installer with Ansible
Once the inventory file and required resources are setup, you will be able to run
the ansible playbook to install PostgreSQL Operator with the `pgo-installer`
image. This can be done using the `ansible-playbook` command and passing in your
inventory file, the relevant tag from the list of options below, and the `main.yml` file.
For example, the following command will install the operator:

```
ansible-playbook -i $PGOROOT/installers/ansible/inventory --tags=install-container $PGOROOT/installers/pgo-installer/ansible/main.yml
```

You can also pass in multiple tags as you run the installer. This will allow you
to run a job and cleanup tasks at the same time. For example, the following
command will run the update job and cleanup tasks:

```
ansible-playbook -i $PGOROOT/installers/ansible/inventory --tags=update-container,clean $PGOROOT/installers/pgo-installer/ansible/main.yml
```

#### Ansible Role Options
| Tag Name | Description |
|----------|--------------|
| `install-container` | Uses the `pgo-installer` image to install the PostgreSQL Operator. |
| `update-container` | Uses the `pgo-installer` image to update the PostgreSQL Operator. |
| `uninstall-container` | Uses the `pgo-installer` image to uninstall the PostgreSQL Operator. |
| `setup-client` | Runs setup steps to use the `pgo` client locally |
| `clean` | This option can be added to the `install-container`, `update-container`, and `uninstall-container` tags to delete the job after it completes. |
| `clean-all` | The `namespace` and `clusterrolebinding` will be deleted if they exist. |

### PGO Client
#### Using the PGO Client Locally
Running the pgo client locally when using the pgo-installer image requires
access to the certs stored in the `pgo.tls` Kubernetes secret. The `client.crt`
and `client.key` need to be pulled from this secret and stored locally in a
location that is accessable to the `pgo` client. You will also need to setup a
`pgouser` file that contains the admin username and password that was set in your
inventory file. The Ansible installer for the pgo-installer image will setup
these resources in the `~/.pgo/<install-namespace>` directory. Please set the
following environment variables after the pgo-installer job has completed:

```
cat <<EOF >> ~/.bashrc
export PGOUSER="${HOME?}/.pgo/<pgo-installer-namespace>/pgouser"
export PGO_CA_CERT="${HOME?}/.pgo/<pgo-installer-namespace>/client.crt"
export PGO_CLIENT_CERT="${HOME?}/.pgo/<pgo-installer-namespace>/client.crt"
export PGO_CLIENT_KEY="${HOME?}/.pgo/<pgo-installer-namespace>/client.pem"
EOF
```

This will allow the the `pgo` client to have access to your postgres-operator
instance. Full install instructions for installing the pgo client can be found
in the [Install `pgo` client](/installation/install-pgo-client) section of the
docs.

#### Independent Setup of PGO Client
The steps to setup the `pgo` client are run on each update and install of the
operator. You also have the option of running the setup steps on an existing
postgres-operator without having to update or reinstall. The pgo-installer
ansible roles provide the `setup-client` tag that will download the `pgo` binary
and setup the required files on your local system.