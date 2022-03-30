#!/usr/bin/env bash
# shellcheck disable=SC2016
# vim: set noexpandtab :
set -eu

DISTRIBUTION="$1"

cd "${BASH_SOURCE[0]%/*}"

bundle_directory="bundles/${DISTRIBUTION}"
project_directory="projects/${DISTRIBUTION}"
go_api_directory=$(cd ../../pkg/apis && pwd)

# TODO(tjmoore4): package_name and project_name are kept separate to maintain
# expected names in all projects. This could be consolidated in the future.
package_name='postgresql'
# Per OLM guidance, the filename for the clusterserviceversion.yaml must be prefixed
# with the Operator's package name for the 'redhat' and 'marketplace' bundles.
# https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/troubleshooting.md#get-supported-versions
project_name='postgresoperator'
case "${DISTRIBUTION}" in
	# https://redhat-connect.gitbook.io/certified-operator-guide/appendix/what-if-ive-already-published-a-community-operator
	'redhat') package_name='crunchy-postgres-operator' ;;
	# https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#bundle-structure
	'marketplace') package_name='crunchy-postgres-operator-rhmp' ;;
esac

operator_yamls=$(kubectl kustomize "config/${DISTRIBUTION}")
operator_crds=$(yq <<< "${operator_yamls}" --slurp --yaml-roundtrip 'map(select(.kind == "CustomResourceDefinition"))')
operator_deployments=$(yq <<< "${operator_yamls}" --slurp --yaml-roundtrip 'map(select(.kind == "Deployment"))')
operator_accounts=$(yq <<< "${operator_yamls}" --slurp --yaml-roundtrip 'map(select(.kind == "ServiceAccount"))')
operator_roles=$(yq <<< "${operator_yamls}" --slurp --yaml-roundtrip 'map(select(.kind == "ClusterRole"))')

# Recreate the Operator SDK project.
[ ! -d "${project_directory}" ] || rm -r "${project_directory}"
install -d "${project_directory}"
(
	cd "${project_directory}"
	operator-sdk init --fetch-deps='false' --project-name=${project_name}
	rm ./*.go go.*

	# Generate CRD descriptions from Go markers.
	# https://sdk.operatorframework.io/docs/building-operators/golang/references/markers/
	crd_gvks=$(yq <<< "${operator_crds}" 'map({
		group: .spec.group, kind: .spec.names.kind, version: .spec.versions[].name
	})')
	yq --in-place --yaml-roundtrip --argjson resources "${crd_gvks}" \
		'.multigroup = true | .resources = $resources | .' ./PROJECT

	ln -s "${go_api_directory}" .
	operator-sdk generate kustomize manifests --interactive='false'
)

# Recreate the OLM bundle.
[ ! -d "${bundle_directory}" ] || rm -r "${bundle_directory}"
install -d \
	"${bundle_directory}/manifests" \
	"${bundle_directory}/metadata" \
	"${bundle_directory}/tests/scorecard" \

# `echo "${operator_yamls}" | operator-sdk generate bundle` includes the ServiceAccount which cannot
# be upgraded: https://github.com/operator-framework/operator-lifecycle-manager/issues/2193

# Include Operator SDK scorecard tests.
# https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/
kubectl kustomize "${project_directory}/config/scorecard" \
	> "${bundle_directory}/tests/scorecard/config.yaml"

# Render bundle annotations and strip comments.
# Per Red Hat we should not include the org.opencontainers annotations in the
# 'redhat' & 'marketplace' annotations.yaml file, so only add them for 'community'.
# - https://coreos.slack.com/team/UP1LZCC1Y
if [ ${DISTRIBUTION} == 'community' ]; then
yq --yaml-roundtrip < bundle.annotations.yaml > "${bundle_directory}/metadata/annotations.yaml" \
	--arg package "${package_name}" \
'
	.annotations["operators.operatorframework.io.bundle.package.v1"] = $package |
	.annotations["org.opencontainers.image.authors"] = "info@crunchydata.com" |
	.annotations["org.opencontainers.image.url"] = "https://crunchydata.com" |
	.annotations["org.opencontainers.image.vendor"] = "Crunchy Data" |
.'
else
yq --yaml-roundtrip < bundle.annotations.yaml > "${bundle_directory}/metadata/annotations.yaml" \
	--arg package "${package_name}" \
'
	.annotations["operators.operatorframework.io.bundle.package.v1"] = $package |
.'
fi

# Copy annotations into Dockerfile LABELs.
labels=$(yq --raw-output < "${bundle_directory}/metadata/annotations.yaml" \
	'.annotations | to_entries | map(.key +"="+ (.value | tojson)) | join(" \\\n\t")')
ANNOTATIONS="${labels}" envsubst '$ANNOTATIONS' < bundle.Dockerfile > "${bundle_directory}/Dockerfile"

# Include CRDs as manifests.
crd_names=$(yq --raw-output <<< "${operator_crds}" 'to_entries[] | [.key, .value.metadata.name] | @tsv')
while IFS=$'\t' read -r index name; do
	yq --yaml-roundtrip <<< "${operator_crds}" ".[${index}]" > "${bundle_directory}/manifests/${name}.crd.yaml"
done <<< "${crd_names}"


abort() { echo >&2 "$@"; exit 1; }
dump() { yq --color-output; }

yq > /dev/null <<< "${operator_deployments}" --exit-status 'length == 1' ||
	abort "too many deployments!" $'\n'"$(dump <<< "${operator_deployments}")"

yq > /dev/null <<< "${operator_accounts}" --exit-status 'length == 1' ||
	abort "too many service accounts!" $'\n'"$(dump <<< "${operator_accounts}")"

yq > /dev/null <<< "${operator_roles}" --exit-status 'length == 1' ||
	abort "too many roles!" $'\n'"$(dump <<< "${operator_roles}")"

# Render bundle CSV and strip comments.

csv_stem=$(yq --raw-output '.projectName' "${project_directory}/PROJECT")

# marketplace and redhat require different naming patters than community
if [ ${DISTRIBUTION} == 'marketplace' ] || [ ${DISTRIBUTION} == 'redhat' ]; then
	mv "${project_directory}/config/manifests/bases/${project_name}.clusterserviceversion.yaml" \
	"${project_directory}/config/manifests/bases/${package_name}.clusterserviceversion.yaml" 
	
	csv_stem=${package_name}
fi

crd_descriptions=$(yq '.spec.customresourcedefinitions.owned' \
"${project_directory}/config/manifests/bases/${csv_stem}.clusterserviceversion.yaml")

crd_gvks=$(yq <<< "${operator_crds}" 'map({
	group: .spec.group, kind: .spec.names.kind, version: .spec.versions[].name
} | {
	apiVersion: "\(.group)/\(.version)", kind
})')
crd_examples=$(yq <<< "${operator_yamls}" --slurp --argjson gvks "${crd_gvks}" 'map(select(
	IN({ apiVersion, kind }; $gvks | .[])
))')

yq --yaml-roundtrip < bundle.csv.yaml > "${bundle_directory}/manifests/${csv_stem}.clusterserviceversion.yaml" \
	--argjson deployment "$(yq <<< "${operator_deployments}" 'first')" \
	--argjson account "$(yq <<< "${operator_accounts}" 'first | .metadata.name')" \
	--argjson rules "$(yq <<< "${operator_roles}" 'first | .rules')" \
	--argjson crds "${crd_descriptions}" \
	--arg examples "${crd_examples}" \
	--arg version "${PGO_VERSION}" \
	--arg replaces "${REPLACES_VERSION}" \
	--arg description "$(< description.md)" \
	--arg icon "$(base64 ../seal.svg | tr -d '\n')" \
	--arg stem "${csv_stem}" \
'
	.metadata.annotations["alm-examples"] = $examples |
	.metadata.annotations["containerImage"] = ($deployment.spec.template.spec.containers[0].image) |

	.metadata.name = "\($stem).v\($version)" |
	.spec.version = $version |
	.spec.replaces = "\($stem).v\($replaces)" |

	.spec.customresourcedefinitions.owned = $crds |
	.spec.description = $description |
	.spec.icon = [{ mediatype: "image/svg+xml", base64data: $icon }] |

	.spec.install.spec.permissions = [{ serviceAccountName: $account, rules: $rules }] |
	.spec.install.spec.deployments = [( $deployment | { name: .metadata.name, spec } )] |
.'

case "${DISTRIBUTION}" in
	'redhat')
		# https://redhat-connect.gitbook.io/certified-operator-guide/appendix/what-if-ive-already-published-a-community-operator
		yq --in-place --yaml-roundtrip \
		'
			.metadata.annotations.certified = "true" |
		.' \
			"${bundle_directory}/manifests/${csv_stem}.clusterserviceversion.yaml"

  		# Finally, add related images. NOTE: SHA values will need to be updated
		# -https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/troubleshooting.md#digest-pinning
		cat bundle.relatedImages.yaml >> "${bundle_directory}/manifests/${csv_stem}.clusterserviceversion.yaml"
		;;
	'marketplace')
		# Annotations needed when targeting Red Hat Marketplace
		# https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/ci-pipeline.md#bundle-structure
		yq --in-place --yaml-roundtrip \
				--arg package_url "https://marketplace.redhat.com/en-us/operators/${package_name}" \
		'
				.metadata.annotations["marketplace.openshift.io/remote-workflow"] =
						"\($package_url)/pricing?utm_source=openshift_console" |
				.metadata.annotations["marketplace.openshift.io/support-workflow"] =
						"\($package_url)/support?utm_source=openshift_console" |
		.' \
			"${bundle_directory}/manifests/${csv_stem}.clusterserviceversion.yaml"

  		# Finally, add related images. NOTE: SHA values will need to be updated
		# -https://github.com/redhat-openshift-ecosystem/certification-releases/blob/main/4.9/ga/troubleshooting.md#digest-pinning
		cat bundle.relatedImages.yaml >> "${bundle_directory}/manifests/${csv_stem}.clusterserviceversion.yaml"
		;;
esac

if > /dev/null command -v tree; then tree -C "${bundle_directory}"; fi
