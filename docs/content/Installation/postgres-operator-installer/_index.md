---
title: "Deploy with PostgreSQL Operator Installer (pgo-installer)"
date:
draft: false
weight: 100
---

## Install the Postgres Operator with Installer Image

The following jobs can be run to install, update, and uninstall the Crunchy
PostgreSQL Operator in your Kubernetes or OpenShift cluster using the
pgo-installer image. Examples of these job can be found in
`$PGOROOT/installers/pgo-installer/ansible/roles/pgo-installer/templates`. The
job templates will need to updated with the following variables to run correctly.

### Installer Namespace
The job template allows you to specify the namespace in
which to run the install job. This does not specify which namespace the
postgres-operator will be installed but they can both be in the same namespace.
The namespace should be defined in place of the `{{ pgo_installer_namespace }}`
variable.

### Cluster Resources
#### Service Account
The postgres-operator-installer
requires a service account with cluster-admin privileges. You can create a
service account manually and assign it to the job by updating the `{{
pgo_installer_sa }}` variable.

#### Config Map
The ansible installer used by the `pgo-installer` image requires
an inventory file to be created as a configmap in your environment. This
configmap will be used to install the  PostgreSQL Operator and should meet all
of the requirements outlined in the ansible install instructions.

### Job Varibles
#### Command
The command defined in the installer job uses
the `pgo-install.sh` script to pass in the ansible tag to be run. The command
can use any of the tags supported by the ansible installer.

#### Image Prefix and Tag
The install job uses the `pgo-installer` image that is
built using each version of the ansible installer. You will need to update the
`{{ pgo_image_prefix }}` and `{{ pgo_image_tag }}` for the version of the
installer that you are using. The `pgo-installer` tag must match the version of
the Crunchy PostgreSQL Operator that you are installing.

#### Image Pull Policy
The image pull policy needs to be defined for your job.
In most cases this should be updated to `IfNotPresent`.


