# Create a Postgres Cluster

This is a working example of how to create a PostgreSQL cluster [Helm](https://helm.sh/) chart.

## Prerequisites

### Postgres Operator

This example assumes you have the [Crunchy PostgreSQL Operator installed](https://access.crunchydata.com/documentation/postgres-operator/latest/quickstart/) in a namespace called `pgo`.  

### Helm

To execute a Helm chart, [Helm](https://helm.sh/) needs to be installed in your local environment.

## Setup

If you are running Postgres Operator 4.5.1 or later, you can skip the step below.

### Before 4.5.1

```
cd postgres-operator/examples/helm/create-cluster

mkdir certs
cd certs

# this variable is the name of the cluster being created
export pgo_cluster_name=hippo

# generate a SSH public/private keypair for use by pgBackRest
ssh-keygen -t ed25519 -N '' -f "${pgo_cluster_name}-key"
```

## Running the Example

### Download the Helm Chart

For this example we will deploy the cluster into the `pgo` namespace where the Postgres Operator is installed and running.

You will need to download this Helm chart. One way to do this is by cloning the Postgres Operator project into your local environment:

```
git clone --depth 1 https://github.com/CrunchyData/postgres-operator.git
```

Go into the directory that contains the Helm chart for creating a PostgreSQL cluster:

```
cd postgres-operator/examples/helm
```

### Set Values

There are only three required values to run the Helm chart:

- `name`: The name of your PostgreSQL cluster.
- `namespace`: The namespace for where the PostgreSQL cluster should be deployed.
- `password`: A password for the user that will be allowed to connect to the database.

The following values can also be set:

- `cpu`: The CPU limit for the PostgreSQL cluster. Follows standard Kubernetes formatting.
- `diskSize`: The size of the PVC for the PostgreSQL cluster. Follows standard Kubernetes formatting.
- `ha`: Whether or not to deploy a high availability PostgreSQL cluster. Can be either `true` or `false`, defaults to `false`.
- `imagePrefix`: The prefix of the container images to use for this PostgreSQL cluster. Default to `registry.developers.crunchydata.com/crunchydata`.
- `image`: The name of the container image to use for the PostgreSQL cluster. Defaults to `crunchy-postgres-ha`.
- `imageTag`: The container image tag to use. Defaults to `centos8-13.3-4.7.0`.
- `memory`: The memory limit for the PostgreSQL cluster. Follows standard Kubernetes formatting.
- `monitoring`: Whether or not to enable monitoring / metrics collection for this PostgreSQL instance. Can either be `true` or `false`, defaults to `false`.

### Execute the Chart

The following commands will allow you to execute a dry run first with debug
if you want to verify everything is set correctly. Then after everything looks
good run the install command with out the flags:

```
helm install -n pgo --dry-run --debug postgres-cluster postgres
helm install -n pgo postgres-cluster postgres
```

This will deploy a PostgreSQL cluster with the specified name into the specified namespace.

## Verify

You can verify that your PostgreSQL cluster is deployed into the `pgo` namespace by running the following commands:

```
kubectl get all -n pgo
```

Once your PostgreSQL cluster is provisioned, you can connect to it. Assuming you are using the default value of `hippo` for the name of the cluster, in a new terminal window, set up a port forward to the PostgreSQL cluster:

```
kubectl -n pgo port-forward svc/hippo 5432:5432
```

Still assuming your are using the default values for this Helm chart, you can connect to the Postgres cluster with the following command:

```
PGPASSWORD="W4tch0ut4hippo$" psql -h localhost -U hippo hippo
```

## Notes

Prior to PostgreSQL Operator 4.7.0, you will have to manually clean up some of the artifacts when running `helm uninstall`.

## Additional Resources

Please see the documentation for more guidance using custom resources:

[https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/](https://access.crunchydata.com/documentation/postgres-operator/latest/custom-resources/)
