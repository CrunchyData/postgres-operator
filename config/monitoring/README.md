To deploy monitoring:

> **_NOTE:_** For more detailed instructions on deploying, see the [documentation on installing Monitoring](https://access.crunchydata.com/documentation/postgres-operator/latest/tutorials/day-two/monitoring).

1. verify the namespace is correct in kustomization.yaml 
2. If you are deploying in openshift, comment out the fsGroup line under securityContext in the following files:
  - `alertmanager/deployment.yaml`
  - `grafana/deployment.yaml`
  - `prometheus/deployment.yaml`
3. kubectl apply -k .
