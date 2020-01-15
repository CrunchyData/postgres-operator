---
title: "Upgrade PostgreSQL Operator 4.1 Minor Versions"
Latest Release: 4.3.0 {docdate}
draft: false
weight: 8
---
## Upgrading Postgres Operator from 4.1.0 to a patch release

This procedure will give instructions on how to upgrade Postgres Operator 4.1
patch releases.

{{% notice info %}}

As with any upgrade, please ensure you have taken recent backups of all relevant
data!

{{% / notice %}}

##### Prerequisites

You will need the following items to complete the upgrade:

* The latest 4.1.X code for the Postgres Operator available
* The latest 4.1.X PGO client binary
* Finally, these instructions assume you are executing from $COROOT in a
terminal window and that you are using the same user from your previous
installation. This user must also still have admin privileges in your Kubernetes
or Openshift environment.

##### Step 1

Run `pgo show config` and save this output to compare at the end to ensure you
do not miss any of your current configuration changes.

##### Step 2

Update environment variables in the `.bashrc` file:

```bash
export CO_VERSION=4.1.X
```

If you are pulling your images from the same registry as before this should be
the only update to the 4.1.X environment variables.

source the updated bash file:

```bash
source ~/.bashrc
```

Check to make sure that the correct CO_IMAGE_TAG image tag is being used. With
a centos7 base image and version 4.1.X of the operator your image tag will be in
the format of `centos7-4.1.1`. Verify this by running echo `$CO_IMAGE_TAG`.


##### Step 3

Update the pgo.yaml file in `$COROOT/conf/postgres-operator/pgo.yaml`. Use the
config that you saved in Step 1. to make sure that you have updated the settings
to match the old config. Confirm that the yaml file includes the correct images
for the updated version.

For example, to update to versions 4.1.1:

```yaml
CCPImageTag: centos7-11.6-4.1.1
COImageTag: centos7-4.1.1
```

##### Step 4

Install the 4.1.X Operator:

```bash
make deployoperator
```

Verify the Operator is running:

```bash
kubectl get pod -n <operator namespace>
```


##### Step 5

Update the `pgo` client binary to 4.1.x by replacing the binary file with the
new one.

Run `which pgo` to ensure you are replacing the current binary.

##### Step 6

Make sure that any and all configuration changes have been updated. Run:

```bash
pgo show config
```

This will print out the current configuration that the operator is using.
Ensure you made any configuration changes required, you can compare this output
with Step 1 to ensure no settings are missed.  If you happened to miss a
setting, update the `pgo.yaml` file and rerun `make deployoperator`.


##### Step 7

The Postgres Operator is now upgraded to 4.1.X.

Verify this by running:

```bash
pgo version
```

## Postgres Operator Container Upgrade Procedure

At this point, the Operator should be running the latest minor version of 4.1,
and new clusters will be built using the appropriate specifications defined in
your pgo.yaml file. For the existing clusters, upgrades can be performed with
the following steps.

{{% notice info %}}

Before beginning your upgrade procedure, be sure to consult the
[Compatibility Requirements Page]( {{< relref "configuration/compatibility.md" >}})
for container dependency information.

{{% / notice %}}

You can upgrade each cluster using the following command:

```bash
pgo upgrade -n <clusternamespace> --ccp-image-tag=centos7-11.6-4.1.1 <clustername>
```

This process takes a few momnets to complete.

To check everything is now working as expected, execute:

```bash
pgo test yourcluster
```

To check the various cluster elements are listed as expected:

```bash
pgo show cluster -n <clusternamespace> <clustername>
```

You've now completed the upgrade and are running Crunchy PostgreSQL Operator
v4.1.X!  For this minor upgrade, most existing settings and related services
(such as pgBouncer, backup schedules and existing policies) are expected to
work, but should be tested for functionality and adjusted or recreated as
necessary.
