This directory contains the files that are used to install [Crunchy PostgreSQL for Kubernetes][hub-listing],
which includes PGO, the Postgres Operator from [Crunchy Data][], using [Operator Lifecycle Manager][OLM].

The integration centers around a [ClusterServiceVersion][olm-csv] [manifest](./bundle.csv.yaml)
that gets packaged for OperatorHub. Changes there are accepted only if they pass all the [scorecard][]
tests. Consult the [technical requirements][hub-contrib] when making changes.

<!-- Requirements might have changed with https://github.com/operator-framework/community-operators/issues/4159 -->

[Crunchy Data]: https://www.crunchydata.com
[hub-contrib]: https://operator-framework.github.io/community-operators/packaging-operator/
[hub-listing]: https://operatorhub.io/operator/postgresql
[OLM]: https://github.com/operator-framework/operator-lifecycle-manager
[olm-csv]: https://github.com/operator-framework/operator-lifecycle-manager/blob/master/doc/design/building-your-csv.md
[scorecard]: https://sdk.operatorframework.io/docs/advanced-topics/scorecard/

[Red Hat Container Certification]: https://redhat-connect.gitbook.io/partner-guide-for-red-hat-openshift-and-container/
[Red Hat Operator Certification]: https://redhat-connect.gitbook.io/certified-operator-guide/

<!-- registry.connect.redhat.com/crunchydata/postgres-operator-bundle -->


## Testing

### Setup

```sh
make tools
```

### Testing

```sh
make bundles validate-bundles
```

```sh
BUNDLE_DIRECTORY='bundles/community'
BUNDLE_IMAGE='gcr.io/.../postgres-operator-bundle:latest'
INDEX_IMAGE='gcr.io/.../postgres-operator-bundle-index:latest'
NAMESPACE='pgo'

docker build --tag "$BUNDLE_IMAGE" "$BUNDLE_DIRECTORY"
docker push "$BUNDLE_IMAGE"

opm index add --bundles "$BUNDLE_IMAGE" --tag "$INDEX_IMAGE" --container-tool=docker
docker push "$INDEX_IMAGE"

./install.sh operator "$BUNDLE_DIRECTORY" "$INDEX_IMAGE" "$NAMESPACE" "$NAMESPACE"

# Cleanup
operator-sdk cleanup postgresql --namespace="$NAMESPACE"
kubectl -n "$NAMESPACE" delete operatorgroup olm-operator-group
```
