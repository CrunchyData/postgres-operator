# Install

This is a working example of how to install the postgres-operator using Helm.

## Helm
Helm will need to be installed for this example to run.

## Documentation
Please see the documentation for more guidance using the postgres-operator

https://access.crunchydata.com/documentation/postgres-operator/latest

## CRD
With Helm v3, CRDs created by this chart are not updated by default and should be manually updated. Consult also the [Helm Documentation on CRDs](https://helm.sh/docs/chart_best_practices/custom_resource_definitions).

## Installing
For this exmple we will deploy the operator into the postgres-operator namespace. Return to the helm directory:
```
cd postgres-operator/examples/helm
```

The following commands will deploy the operator into the postgres-operator namespace. The `--dry-run` flag will allow you to verify that your configuration is set correctly. Then run the install command without the flag.
```
helm install --dry-run -n postgres-operator postgres-operator install

helm install -n postgres-operator postgres-operator install
```

## Verify
Now you can verify your postgres-operator has deploy into the postgres-operator namespace by running these commands:

```
helm list
```

You will see the postgres-operator pod and deployment:
```
kubectl get pods,deployments -n postgres-operator
```

You can also verify that the clusterrole and clusterrolebinding have been created:
```
kubectl get clusterrole postgres-operator
kubectl get clusterrolebinding postgres-operator
```

If you are running with `singleNamespace: true`, you should check that the role and rolebinding have been created:
```
kubectl get rolebinding -n postgres-operator
kubectl get role -n postgres-operator
```

## Uninstall
Run the following helm command to uninstall the postgres-operator:

```
helm uninstall -n postgres-operator postgres-operator
```