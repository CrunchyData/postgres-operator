#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

# Simplify `push_trap_exit()` by always having something.
trap 'date' EXIT

push_trap_exit() {
	local -a array
	read -ra array <<< "$( trap -p EXIT )"
	eval "local previous=${array[@]:2:(${#array[@]}-3)}"
	trap "$1; $previous" EXIT
}

# Store anything in a single temporary directory that gets cleaned up.
export TMPDIR="$(mktemp --directory)"
push_trap_exit "rm -rf '$TMPDIR'"

# Use a version of `kubectl` that matches the Kubernetes server.
command -v kubectl >/dev/null || eval "kubectl() { kubectl-$( kubectl-1.16 version --output=json |
	jq --raw-output '.serverVersion | .major + "." + .minor')"' "$@"; }'

# Create a Namespace in which to deploy and test.
test_namespace="$( kubectl create --filename=- --output=jsonpath='{.metadata.name}' <<< '{
	"apiVersion": "v1", "kind": "Namespace",
	"metadata": { "generateName": "olm-test-" }
}' )"
echo 'namespace "'"$test_namespace"'" created'
push_trap_exit "kubectl delete namespace '$test_namespace'"

kc() { kubectl --namespace="$test_namespace" "$@"; }

# Clean up anything created by the Subscription, especially CustomResourceDefinitions.
push_trap_exit "kc delete --ignore-not-found --filename='./package/${PGO_VERSION}/'"

# Install the package and inject the scorecard proxy.
./install.sh operator "$test_namespace" "$test_namespace"
./install.sh scorecard "$test_namespace" "$OLM_VERSION"

# Restore the OLM operator that was disabled to inject the scorecard proxy.
push_trap_exit 'kubectl --namespace olm scale --replicas=1 deploy olm-operator'


# Run the OLM test suite against each example stored in CSV annotations.
examples_array="$( yq read \
	"./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml" \
	metadata.annotations.alm-examples )"

error=0
for index in $(seq 0 $(jq <<< "$examples_array" 'length - 1')); do
	jq > "${TMPDIR}/resource.json" <<< "$examples_array" ".[$index]"
	jq > "${TMPDIR}/scorecard.json" <<< '{}' \
		--arg resource "${TMPDIR}/resource.json" \
		--arg namespace "$test_namespace" \
		--arg version "$PGO_VERSION" \
	'{ scorecard: { plugins: [
		{ basic: {
			"cr-manifest": $resource,
			"crds-dir": "./package/\($version)",
			"csv-path": "./package/\($version)/postgresoperator.v\($version).clusterserviceversion.yaml",
			"namespace": $namespace,
			"olm-deployed": true
		} },
		{ olm: {
			"cr-manifest": $resource,
			"crds-dir": "./package/\($version)",
			"csv-path": "./package/\($version)/postgresoperator.v\($version).clusterserviceversion.yaml",
			"namespace": $namespace,
			"olm-deployed": true
		} }
	] } }'

	echo "Verifying metadata.annotations.alm-examples<[$index]>:"
	jq '{ apiVersion, kind, name: .metadata.name }' "${TMPDIR}/resource.json"

	start="$(date --utc +'%FT%TZ')"
	if ! operator-sdk scorecard --config "${TMPDIR}/scorecard.json" --verbose; then
		#kc logs --container='operator' --selector='name=postgres-operator' --since-time="$start"
		error=1
	fi
done

exit $error
