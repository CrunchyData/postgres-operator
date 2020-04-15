---
title: "Deploy with PostgreSQL Operator Installer (pgo-deployer)"
date:
draft: false
weight: 100
---

## Install the Postgres Operator with Installer Image

The following job can be run to `install`, `update`, and `uninstall` the Crunchy
PostgreSQL Operator in your Kubernetes or OpenShift cluster using the
pgo-deployer image. Examples of these job can be found in
`$PGOROOT/installers/method/ansible-playbook/roles/pgo-deployer/templates`. The
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

### Running the pgo Client Binary

The `pgo-client` allows you to work with the PostgreSQL Operator using the
convenient command-line utility, similar to how you would use `kubectl`. You can
use the `pgo-client` from a container image, or install it to your local
environment. as a binary. The following steps will show you how to set up and
use both methods.

#### pgo-client container

The pgo-client image can be installed along side of the PostgreSQL Operator by
enabling the `pgo_client_container_install` option in the inventory file.
This image contains the pgo binary and is setup to have access to the
apiserver and can be accessed by running the following:

```
kubectl exec -it -n <operator-namespace> pgo-client-<id> bash
```

Once you `exec` into the container you can run `pgo` commands without having to
do any more setup. More information about the pgo-client container can be found
[here]({{< relref "installation/install-pgo-client/_index.md" >}}) in the docs.

#### pgo Client Binary

The pgo binary has required resources that are needed to connect to the
apiserver. These resources are defined under the [Install the Postgres Operator
(pgo) Client]({{< relref "installation/install-pgo-client/_index.md" >}}) section of the documentation. By
following these steps you will be able to install the `pgo` client and setup the
necessary resources.

##### Configuring Client TLS

The `client.pem` and `client.crt` files can be found in the `pgo.tls` secret in
the `<operator-namespace>` namespace. You can use `kubectl` to access the secret
and store it locally as instructed in the [client install]({{< relref "installation/install-pgo-client/_index.md" >}}) docs.

```
kubectl get secret -n pgo pgo.tls -o jsonpath="{.data.tls\.crt}" | base64 --decode > client.crt
kubectl get secret -n pgo pgo.tls -o jsonpath="{.data.tls\.key}" | base64 --decode > client.pem
```

##### Configuring pgouser

The pgouser file contains the username and password used for authentication with
the Crunchy PostgreSQL Operator. You will need to create this file with the
correct username and password in order for the `pgo` binary to access the
PostgreSQL Operator. More information about this file can be found in the
[Install pgo Client]({{< relref "installation/install-pgo-client/_index.md" >}})
documentation. The username and password are specified in the inventory file
when the operator is installed. The variables in the inventory file are
`pgo_admin_username` and `pgo_admin_password`.
