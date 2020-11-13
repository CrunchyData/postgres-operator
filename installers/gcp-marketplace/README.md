
This directory contains the files that are used to install [Crunchy PostgreSQL for GKE][gcp-details],
which uses the PostgreSQL Operator, from the Google Cloud Marketplace.

The integration centers around a container [image](./Dockerfile) that contains an installation
[schema](./schema.yaml) and an [Application][k8s-app] [manifest](./application.yaml).
Consult the [technical requirements][gcp-k8s-requirements] when making changes.

[k8s-app]: https://github.com/kubernetes-sigs/application/
[gcp-k8s]: https://cloud.google.com/marketplace/docs/kubernetes-apps/
[gcp-k8s-requirements]: https://cloud.google.com/marketplace/docs/partners/kubernetes-solutions/create-app-package
[gcp-k8s-tool-images]: https://console.cloud.google.com/gcr/images/cloud-marketplace-tools
[gcp-k8s-tool-repository]: https://github.com/GoogleCloudPlatform/marketplace-k8s-app-tools
[gcp-details]: https://console.cloud.google.com/marketplace/details/crunchydata/crunchy-postgresql-operator


# Installation

## Quick install with Google Cloud Marketplace

Install [Crunchy PostgreSQL for GKE][gcp-details] to a Google Kubernetes Engine cluster using
Google Cloud Marketplace.

## Command line instructions

### Prepare

1. You'll need the following tools in your development environment. If you are using Cloud Shell,
   everything is already installed.

   - envsubst
   - [git](https://git-scm.com/book/en/v2/Getting-Started-Installing-Git)
   - [kubectl](https://kubernetes.io/docs/reference/kubectl/overview/)

2. Clone this repository.

   ```shell
   git clone https://github.com/CrunchyData/postgres-operator.git
   ```

3. Install the [Application][k8s-app] Custom Resource Definition.

   ```shell
   kubectl apply -f 'https://raw.githubusercontent.com/GoogleCloudPlatform/marketplace-k8s-app-tools/master/crd/app-crd.yaml'
   ```

4. At least one Storage Class is required. Google Kubernetes Engine is preconfigured with a default.

   ```shell
   kubectl get storageclasses
   ```

### Install the PostgreSQL Operator

1. Configure the installation by setting environment variables.

   1. Choose a version to install.

      ```shell
      IMAGE_REPOSITORY=gcr.io/crunchydata-public/postgres-operator

      export PGO_VERSION=4.5.0
      export INSTALLER_IMAGE=${IMAGE_REPOSITORY}/deployer:${PGO_VERSION}
      export OPERATOR_IMAGE=${IMAGE_REPOSITORY}:${PGO_VERSION}
      export OPERATOR_IMAGE_API=${IMAGE_REPOSITORY}/pgo-apiserver:${PGO_VERSION}
      ```

   2. Choose a namespace and name for the application.

      ```shell
      export OPERATOR_NAMESPACE=pgo OPERATOR_NAME=pgo
      ```

   2. Choose a password for the application admin.

      ```shell
      export OPERATOR_ADMIN_PASSWORD=changethis
      ```

   4. Choose default values for new PostgreSQL clusters.

      ```shell
      export POSTGRES_METRICS=false
      export POSTGRES_SERVICE_TYPE=ClusterIP
      export POSTGRES_CPU=1000 # mCPU
      export POSTGRES_MEM=2 # GiB
      export POSTGRES_STORAGE_CAPACITY=1 # GiB
      export POSTGRES_STORAGE_CLASS=ssd
      export PGBACKREST_STORAGE_CAPACITY=2 # GiB
      export PGBACKREST_STORAGE_CLASS=ssd
      export BACKUP_STORAGE_CAPACITY=1 # GiB
      export BACKUP_STORAGE_CLASS=ssd
      ```

2. Prepare the Kubernetes namespace.

   ```shell
   export INSTALLER_SERVICE_ACCOUNT=postgres-operator-installer

   kubectl create namespace "$OPERATOR_NAMESPACE"
   kubectl create serviceaccount -n "$OPERATOR_NAMESPACE" "$INSTALLER_SERVICE_ACCOUNT"
   kubectl create clusterrolebinding \
     "$OPERATOR_NAMESPACE:$INSTALLER_SERVICE_ACCOUNT:cluster-admin" \
     --serviceaccount="$OPERATOR_NAMESPACE:$INSTALLER_SERVICE_ACCOUNT" \
     --clusterrole=cluster-admin
   ```

3. Generate and apply Kubernetes manifests.

   ```shell
   envsubst < application.yaml > "${OPERATOR_NAME}_application.yaml"
   envsubst < install-job.yaml > "${OPERATOR_NAME}_install-job.yaml"
   envsubst < inventory.ini > "${OPERATOR_NAME}_inventory.ini"

   kubectl create -n "$OPERATOR_NAMESPACE" secret generic install-postgres-operator \
     --from-file=inventory="${OPERATOR_NAME}_inventory.ini"

   kubectl create -n "$OPERATOR_NAMESPACE" -f "${OPERATOR_NAME}_application.yaml"
   kubectl create -n "$OPERATOR_NAMESPACE" -f "${OPERATOR_NAME}_install-job.yaml"
   ```

The application can be seen in Google Cloud Platform Console at [Kubernetes Applications][].

[Kubernetes Applications]: https://console.cloud.google.com/kubernetes/application


# Uninstallation

## Using Google Cloud Platform Console

1. In the Console, open [Kubernetes Applications][].
2. From the list of applications, select _Crunchy PostgreSQL Operator_ then click _Delete_.

## Command line instructions

Delete the Kubernetes resources created during install.

```shell
export OPERATOR_NAMESPACE=pgo OPERATOR_NAME=pgo

kubectl delete -n "$OPERATOR_NAMESPACE" job install-postgres-operator
kubectl delete -n "$OPERATOR_NAMESPACE" secret install-postgres-operator
kubectl delete -n "$OPERATOR_NAMESPACE" application "$OPERATOR_NAME"
```
