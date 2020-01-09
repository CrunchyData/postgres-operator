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


# Install the package and inject the scorecard proxy.
./install.sh operator "$test_namespace" "$test_namespace"
./install.sh scorecard "$test_namespace" "$OLM_VERSION"

# Restore the OLM operator that was disabled to inject the scorecard proxy.
push_trap_exit 'kubectl --namespace olm scale --replicas=1 deploy olm-operator'

# Clean up anything created by the Subscription, especially CustomResourceDefinitions.
push_trap_exit "kc delete --ignore-not-found --filename='./package/${PGO_VERSION}/'"


# Create some configuration referenced in the examples.
kc delete secret example-postgresuser --ignore-not-found
kc create secret generic example-postgresuser \
	--from-literal='username=postgres' \
	--from-literal='password=password'

kc delete secret example-primaryuser --ignore-not-found
kc create secret generic example-primaryuser \
	--from-literal='username=primaryuser' \
	--from-literal='password=password'

kc delete secret example-backrest-repo-config --ignore-not-found
kc create secret generic example-backrest-repo-config

scorecard_config="$(mktemp).json"

# Run the OLM test suite against the first example stored in CSV annotations.
jq > "$scorecard_config" <<< '{}' \
	--arg namespace "$test_namespace" \
	--arg version "$PGO_VERSION" \
'{ scorecard: { plugins: [
	{ olm: {
		"crds-dir": "./package/\($version)",
		"csv-path": "./package/\($version)/postgresoperator.v\($version).clusterserviceversion.yaml",
		"namespace": $namespace,
		"olm-deployed": true
	} }
] } }'
operator-sdk scorecard --config "$scorecard_config" --verbose

# TODO repeat above with the Basic test suite after cleaning up.
# TODO repeat above with any CR specified in `cr-manifest`.
