#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

render() { envsubst '$CCP_IMAGE_PREFIX $CCP_IMAGE_TAG $PACKAGE_NAME $PGO_IMAGE_PREFIX $PGO_IMAGE_TAG $PGO_VERSION'; }

mkdir -p "./package/${PGO_VERSION}"

# PackageManifest filename must end with '.package.yaml'
render < postgresql.package.yaml > "./package/${PACKAGE_NAME}.package.yaml"

# ClusterServiceVersion filenames must end with '.clusterserviceversion.yaml'
render < postgresoperator.csv.yaml > "./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"

crd_array="$( yq read --doc='*' --tojson postgresoperator.crd.yaml )"
crd_names="$( jq <<< "$crd_array" --raw-output 'to_entries[] | [.key, .value.metadata.name] | @tsv' )"

# `operator-courier verify` expects only one CustomResourceDefinition per file.
while IFS=$'\t' read index name; do
	yq read --doc="$index" postgresoperator.crd.yaml > "./package/${PGO_VERSION}/${name}.crd.yaml"
done <<< "$crd_names"

yq_script="$( yq read --tojson "./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml" | jq \
	--argjson images "$( yq read --tojson postgresoperator.csv.images.yaml | render )" \
	--argjson crds "$( yq read --tojson postgresoperator.crd.descriptions.yaml | render )" \
	--arg examples "$( yq read --tojson postgresoperator.crd.examples.yaml --doc='*' | render | jq . )" \
	--arg description "$( render < postgresoperator.csv.description.md )" \
	--arg icon "$( base64 --wrap=0 ../seal.svg )" \
'{
	"metadata.annotations.alm-examples": $examples,
	"spec.customresourcedefinitions.owned": $crds,
	"spec.description": $description,
	"spec.icon": [{ mediatype: "image/svg+xml", base64data: $icon }],

	"spec.install.spec.deployments[0].spec.template.spec.containers[0].env": (
	.spec.install.spec.deployments[0].spec.template.spec.containers[0].env + $images),

	"spec.install.spec.deployments[0].spec.template.spec.containers[1].env": (
	.spec.install.spec.deployments[0].spec.template.spec.containers[1].env + $images)
}' )"
yq write --inplace --script=- <<< "$yq_script" "./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"

if [ "${K8S_DISTRIBUTION:-}" = 'openshift' ]; then
	yq_script="$( jq <<< '{}' \
		--arg description "$( render < openshift.description.md )" \
	'{
		"spec.description": $description,
		"spec.displayName": "Crunchy PostgreSQL for OpenShift",
	}' )"
	yq write --inplace --script=- <<< "$yq_script" "./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"
fi

if > /dev/null command -v tree; then tree -C './package'; fi
