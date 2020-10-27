#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

render() { envsubst '$CCP_IMAGE_PREFIX $CCP_IMAGE_TAG $CCP_POSTGIS_IMAGE_TAG $PACKAGE_NAME $PGO_IMAGE_PREFIX $PGO_IMAGE_TAG $PGO_VERSION'; }

mkdir -p "./package/${PGO_VERSION}"

# PackageManifest filename must end with '.package.yaml'
render < postgresql.package.yaml > "./package/${PACKAGE_NAME}.package.yaml"

# ClusterServiceVersion filenames must end with '.clusterserviceversion.yaml'
render < postgresoperator.csv.yaml > "./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"

crd_names="$( yq --raw-output --slurp 'to_entries[] | [.key, .value.metadata.name] | @tsv' postgresoperator.crd.yaml )"

# `operator-courier verify` expects only one CustomResourceDefinition per file.
while IFS=$'\t' read index name; do
	yq --slurp --yaml-roundtrip ".[$index]" postgresoperator.crd.yaml > "./package/${PGO_VERSION}/${name}.crd.yaml"
done <<< "$crd_names"

yq --in-place --yaml-roundtrip \
	--argjson images "$( yq '.' postgresoperator.csv.images.yaml | render )" \
	--argjson crds "$( yq '.' postgresoperator.crd.descriptions.yaml | render )" \
	--arg examples "$( yq --slurp '.' postgresoperator.crd.examples.yaml | render )" \
	--arg description "$( render < description.upstream.md )" \
	--arg icon "$( base64 ../seal.svg | tr -d '\n' )" \
'
	.metadata.annotations["alm-examples"] = $examples |
	.spec.customresourcedefinitions.owned = $crds |
	.spec.description = $description |
	.spec.icon = [{ mediatype: "image/svg+xml", base64data: $icon }] |

	.spec.install.spec.deployments[0].spec.template.spec.containers[0].env += $images |
	.spec.install.spec.deployments[0].spec.template.spec.containers[1].env += $images |
.' \
	"./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"

if [ "${K8S_DISTRIBUTION:-}" = 'openshift' ]; then
	yq --in-place --yaml-roundtrip \
		--arg description "$( render < description.openshift.md )" \
	'
		.spec.description = $description |
		.spec.displayName = "Crunchy PostgreSQL for OpenShift" |
	.' \
		"./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"
fi

if > /dev/null command -v tree; then tree -C './package'; fi
