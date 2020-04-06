---
title: "Deploy with Kubectl"
date:
draft: false
weight: 20
---

The `pgo-installer` image can be deployed using the `kubectl` client. The resources
needed to run the installer are described in the
[pgo-installer]({{< relref "/installation/postgres-operator-installer/_index.md" >}})
section of the documentation. The following yaml files include these resources
that are needed to deploy the operator.

### Resources

Each of these resources is defined in the yaml file that is used to deploy the
operator. These resources can be updated to customize your expirence with the
Postgres-Operator but we provide sane defaults so that you can easily deploy.
By default the installer will create each resource based on the resource
definition in the job yaml.

#### Namespace

The namespace definiton will be used to create a namespace for the installer to 
run. By default the installer will run in a separate namespace from where the
Postgres-Operator will be deployed. If you want the jobs to run in a
preconfigured namespace you can easily update this file to remove the
namespace definition. The other resources will need to be updated to reference
this preconfigured namespace.

#### ServiceAccount and ClusterRoleBinding

The installer image needs a service account that can access the Kubernetes
cluster where the Postgres-Operator will be installed. The job yaml defines a
service account and clusterrolebinding that gives the service account the
cluster-admin role. This is required for the installer to run correctly. If you
have a preconfigured service account with the cluster-admin role, you can remove
this section of the yaml and update the service account name in the job spec.

#### Job

Once the resources have been configured the job spec will be used to deploy the
Postgres-Operator in your Kubernetes environment. The job spec includes sane
defaults that can be used to deploy a specific version of the Postgres-Operator
based on the version of the pgo-installer image that is used. Each version will
install the corresponding version of the Postgres-Operator.

##### Deployment Options

The installer image uses environment variables to specify deployment options for
the Postgres-Operator. The environment variables that you can define are the
same as the options in the inventory file for the ansible installer. These
options can be found in the
[Configuring the Inventory File]({{< relref "/installation/install-with-ansible/prerequisites.md" >}})
section of the docs. The environment variables will be the same as the inventory
options but in all capital letters. A full list of available environment
variables can be found in the `$PGOROOT/installers/method/kubectl/full_options`
file. The deployment options that are included in the default job spec are
required.

### Installing

```
kubectl apply -f $PGOROOT/installers/method/kubectl/install.yml
```

### Uninstalling

```
kubectl apply -f $PGOROOT/installers/method/kubectl/uninstall.yml
```

### Updating

```
kubectl apply -f $PGOROOT/installers/method/kubectl/update.yml
```

### Cleanup

The job resources can be cleaned up by running a delete on the specific yaml
file for each job. *Please not that this will delete the namespace where the
installer ran. If this is the same as the Postgres-Operator namespace that will
also be deleted.* The resources can also be delete manually through the kubectl
client.

```
kubectl delete -f job.yml
```