---
title: "Deploy with Kubectl"
date:
draft: false
weight: 20
---

The `pgo-installer` image can be deployed using the `kubectl` or `oc` clients.
The resources needed to run the installer are described in the
[pgo-installer]({{< relref "/installation/postgres-operator-installer/_index.md" >}})
section of the documentation.

### Resources

The `pgo-installer` requires a serviceaccount and clusterrolebinding to run the
job. Both of these resources are defined `deploy.yml` file and will be created
with the install job. These resources can be updated based on your specific
needs but we provide sane defaults so that you can easily deploy.

The installer will run in the `pgo` namespace by default but this can be
updated in the `deploy.yml` file. Please ensure that the namespace exists before
the job is run.

#### ServiceAccount and ClusterRoleBinding

The installer image needs a service account that can access the Kubernetes
cluster where the PostgreSQL Operator will be installed. The job yaml defines a
service account and clusterrolebinding that gives the service account the
cluster-admin role. This is required for the installer to run correctly. If you
have a preconfigured service account with the cluster-admin role, you can remove
this section of the yaml and update the service account name in the job spec.

##### Image Pull Secrets

If you are pulling the PostgreSQL Operator images from a private registry you
will need to setup an
[imagePullSecret](https://kubernetes.io/docs/tasks/configure-pod-container/pull-image-private-registry/)
with access to the registry. The image pull secret will need to be added to the
installer service account to have access. The secret will need to be created in
each namespace that the PostgreSQL Operator will be using.

After you have configured your image pull secret in the installer namespace,
add the name of the secret to the job yaml that you are using. You can update
the existing section like this:

```
apiVersion: v1
kind: ServiceAccount
metadata:
    name: pgo-installer-sa
    namespace: pgo
imagePullSecrets:
  - name: <image_pull_secret_name>
```

If the service account is configured without using the job yaml file, you
can link the secret to the service account with the `kubectl` or `oc`
clients.

```
# kubectl
kubectl patch serviceaccount <installer-sa> -p '{"imagePullSecrets": [{"name": "myregistrykey"}]}' -n <install-namespace>

# oc
oc secrets link <registry-secret> <installer-sa> --for=pull --namespace=<install-namespace>
```

#### Job

Once the resources have been configured the job spec will be used to deploy the
PostgreSQL Operator in your Kubernetes environment. The job spec includes sane
defaults that can be used to deploy a specific version of the PostgreSQL Operator
based on the version of the pgo-installer image that is used. Each version will
install the corresponding version of the PostgreSQL Operator.

##### Deployment Options

The installer image uses environment variables to specify deployment options for
the PostgreSQL Operator. The environment variables that you can define are the
same as the options in the inventory file for the ansible installer. These
options can be found in the
[Configuring the Inventory File]({{< relref "/installation/install-with-ansible/prerequisites.md" >}})
section of the docs. The environment variables will be the same as the inventory
options but in all capital letters. A full list of available environment
variables can be found in the `$PGOROOT/installers/method/kubectl/full_options`
file. The deployment options that are included in the default job spec are
required.

### Deploying

The deploy job can be used to perform different deployment actions for the
PostgreSQL Operator. If you run the job it will install the operator but you can
change the deployment action by updating the `DEPLOY_ACTION` environment
variable in the `deploy.yml` file. This variable can be set to `install`,
`update`, `uninstall`, `install-metrics`, and `uninstall-metrics`. Each time a
job is run you will need to cleanup the job using the command below.

### pgo Client Binary

Running the pgo client locally when using the pgo-installer image requires
access to the certs stored in the `pgo.tls` Kubernetes secret. The `client.crt`
and `client.key` need to be pulled from this secret and stored locally in a
location that is accessable to the `pgo` client. You will also need to setup a
`pgouser` file that contains the admin username and password that was set in your
inventory file. This username and password will be pulled from the
`pgouser-admin` secret. The the `client-setup.sh` script will setup these
resources in the `~/.pgo/$PGO_OPERATOR_NAMESPACE` directory. Please set the
following environment variables after the pgo-installer job has completed:

```
cat <<EOF >> ~/.bashrc
export PGOUSER="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/pgouser"
export PGO_CA_CERT="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.crt"
export PGO_CLIENT_CERT="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.crt"
export PGO_CLIENT_KEY="$HOME/.pgo/$PGO_OPERATOR_NAMESPACE/client.key"
EOF
```

This will allow the the `pgo` client to have access to your postgres-operator
instance. Full install instructions for installing the pgo client can be found
in the [Install `pgo` client]({{< relref "/installation/install-pgo-client" >}})
section of the docs.

### Cleanup

The job resources can be cleaned up by running a delete on the `deploy.yml`
file. The resources can also be delete manually through the kubectl
client.

```
kubectl delete -f deploy.yml
```