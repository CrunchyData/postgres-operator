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
[scorecard]: https://sdk.operatorframework.io/docs/testing-operators/scorecard/

[Red Hat Container Certification]: https://redhat-connect.gitbook.io/partner-guide-for-red-hat-openshift-and-container/
[Red Hat Operator Certification]: https://redhat-connect.gitbook.io/certified-operator-guide/

<!-- registry.connect.redhat.com/crunchydata/postgres-operator-bundle -->

## Notes

### v5 Versions per Repository

Community: https://github.com/k8s-operatorhub/community-operators/tree/main/operators/postgresql

5.0.2
5.0.3
5.0.4
5.0.5
5.1.0

Community Prod: https://github.com/redhat-openshift-ecosystem/community-operators-prod/tree/main/operators/postgresql

5.0.2
5.0.3
5.0.4
5.0.5
5.1.0

Certified: https://github.com/redhat-openshift-ecosystem/certified-operators/tree/main/operators/crunchy-postgres-operator

5.0.4
5.0.5
5.1.0

Marketplace: https://github.com/redhat-openshift-ecosystem/redhat-marketplace-operators/tree/main/operators/crunchy-postgres-operator-rhmp

5.0.4
5.0.5
5.1.0

### Issues Encountered

We hit various issues with 5.1.0 where the 'replaces' name, set in the clusterserviceversion.yaml, didn't match the
expected names found for all indexes. Previously, we set the 'com.redhat.openshift.versions' annotation to "v4.6-v4.9".
The goal for this setting was to limit the upper bound of supported versions for a particularly PGO release.
The problem with this was, at the time of the 5.1.0 release, OCP 4.10 had been just been released. This meant that the
5.0.5 bundle did not exist in the OCP 4.10 index. The solution presented by Red Hat was to use the 'skips' clause for
the 5.1.0 release to remedy the immediate problem, but then go back to using an unbounded setting for subsequent
releases.

For the certified, marketplace and community repositories, this strategy of using 'skips' instead of replaces worked as
expected. However, for the production community operator bundle, we were seeing a failure that required adding an
additional 'replaces' value of 5.0.4 in addition to the 5.0.5 'skips' value. While this allowed the PR to merge, it
seems at odds with the behavior at the other repos.

For more information on the use of 'skips' and 'replaces', please see:
https://olm.operatorframework.io/docs/concepts/olm-architecture/operator-catalog/creating-an-update-graph/


Another version issue encountered was related to our attempt to both support OCP v4.6 (which is an Extended Update
Support (EUS) release) while also limiting Kubernetes to 1.20+. The issue with this is that OCP 4.6 utilizes k8s 1.19
and the kube minversion validation was in fact limiting the OCP version as well. Our hope was that those setting would
be treated independently, but that was unfortunately not the case. The fix for this was to move this kube version to the
1.19, despite its being released 3rd quarter of 2020 with 1 year of patch support.

Following the lessons learned above, when bumping the Openshift supported version from v4.6 to v4.8, we will similarly
keep the matching minimum Kubernetes version, i.e. 1.21.
https://access.redhat.com/solutions/4870701

## Testing

### Setup

```sh
make tools
```

### Testing

```sh
make bundles validate-bundles
```

Previously, the 'validate_bundle_image' function in validate-bundles.sh ended
with the following command:

```sh
	# Create an index database from the bundle image.
	"${opm[@]}" index add --bundles="${image}" --generate

	# drwxr-xr-x. 2 user user     22 database
	# -rw-r--r--. 1 user user 286720 database/index.db
	# -rw-r--r--. 1 user user    267 index.Dockerfile
```

this command was used to generate the updated registry database, but this step
is no longer required when validating the OLM bundles.
- https://github.com/operator-framework/operator-registry/blob/master/docs/design/opm-tooling.md#add-1

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

### Post Bundle Generation

After generating and testing the OLM bundles, there are two manual steps.

1. Update the image SHA values (denoted with '<update_(imagetype)_SHA_value>', required for both the Red Hat 'Certified' and
'Marketplace' bundles)
2. Update the 'description.md' file to indicate which OCP versions this release of PGO was tested against.

### Troubleshooting

If, when running `make validate-bundles` you encounter an error similar  to

`cannot find Containerfile or Dockerfile in context directory: stat /mnt/Dockerfile: permission denied`

the target command is likely being blocked by SELinux and you will need to adjust
your settings accordingly.
