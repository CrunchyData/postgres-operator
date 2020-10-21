# postgres-operator

The directories in here are intended to be used via [kustomize](https://kubernetes-sigs.github.io/kustomize/).


## Usage

### Apply directly

```shell
kustomize build postgres-operator/installers/kustomize/example | kubectl apply -f -
```


### Use as a kustomize base

In your own repository, you might write a `kustomization.yaml` that contains:

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
namespace: postgres-operator
resources:
  - github.com/CrunchyData/postgres-operator/installers/kustomize/base
components:
  - github.com/CrunchyData/postgres-operator/installers/kustomize/cluster-rbac-readonly
```
