---
title: "Ansible Roles for pgo-installer"
date:
draft: false
weight: 20
---

The ansible roles for the `pgo-installer` can be used to setup and run the
install jobs with the `pgo-installer` image. Ansible will perform the steps
oulined in the [Deploy with PostgreSQL Operator
Installer](#/installation/postgres-operator-install).

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

Please reference the [Configuring the Inventory File](/installation/install-with-ansible/prerequisites#configuring-the-inventory-file)
documentation as you update the inventory file. 

#### PGO-Installer Specific Inventory Options
The PostgreSQL Operator Installer has settings defined in the example inventory
file that are not referenced in the [Configuring the Inventory File](i/installation/install-with-ansible/prerequisites#configuring-the-inventory-file)
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

### Ansible Role Options
| Tag Name | Description |
|----------|--------------|
| `install-container` | Uses the `pgo-installer` image to install the PostgreSQL Operator. |
| `update-container` | Uses the `pgo-installer` image to update the PostgreSQL Operator. |
| `uninstall-container` | Uses the `pgo-installer` image to uninstall the PostgreSQL Operator. |
| `clean` | This option can be added to the `install-container`, `update-container`, and `uninstall-container` tags to delete the job after it completes. |
| `clean-all` | The `namespace` and `clusterrolebinding` will be deleted if they exist. |

