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

if command -v oc >/dev/null; then
	kubectl() { oc "$@"; }
elif ! command -v kubectl >/dev/null; then
	# Use a version of `kubectl` that matches the Kubernetes server.
	eval "kubectl() { kubectl-$( kubectl-1.19 version --output=json |
		jq --raw-output '.serverVersion | .major + "." + .minor')"' "$@"; }'
fi

# Find the OLM operator deployment.
olm_deployments="$( kubectl get deploy --all-namespaces --selector='app=olm-operator' --output=json )"
if [ '1' != "$( jq <<< "$olm_deployments" '.items | length' )" ] ||
	[ 'olm-operator' != "$( jq --raw-output <<< "$olm_deployments" '.items[0].metadata.name' )" ]
then
	>&2 echo Unable to find the OLM operator!
	exit 1
fi
olm_namespace="$( jq --raw-output <<< "$olm_deployments" '.items[0].metadata.namespace' )"

# Create a Namespace in which to deploy and test.
test_namespace="$( kubectl create --filename=- --output=jsonpath='{.metadata.name}' <<< '{
	"apiVersion": "v1", "kind": "Namespace",
	"metadata": { "generateName": "olm-test-" }
}' )"
echo 'namespace "'"$test_namespace"'" created'
push_trap_exit "kubectl delete namespace '$test_namespace'"

kc() { kubectl --namespace="$test_namespace" "$@"; }

# Clean up anything created by the Subscription, especially CustomResourceDefinitions.
push_trap_exit "kubectl delete clusterrole,clusterrolebinding --selector='olm.owner.namespace=$test_namespace'"
push_trap_exit "kubectl delete --ignore-not-found --filename='./package/${PGO_VERSION}/'"

# Install the package.
./install.sh operator "$test_namespace" "$test_namespace"

# Turn off OLM while we manipulate the operator deployment.
# OLM crashes when a running Deployment doesn't match the CSV.
>&2 echo $(tput bold)Turning off the OLM operator!$(tput sgr0)
kubectl --namespace="$olm_namespace" scale --replicas=0 deploy olm-operator
push_trap_exit "kubectl --namespace='$olm_namespace' scale --replicas=1 deploy olm-operator"
kubectl --namespace="$olm_namespace" rollout status deploy olm-operator --timeout=1m

# Inject the scorecard proxy.
./install.sh scorecard "$test_namespace" "$OLM_SDK_VERSION"


# Run the OLM test suite against each example stored in CSV annotations.
examples_array="$( yq --raw-output '.metadata.annotations["alm-examples"]' \
	"./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml" )"

error=0
for index in $(seq 0 $(jq <<< "$examples_array" 'length - 1')); do
	jq > "${TMPDIR}/resource.json" <<< "$examples_array" ".[$index]"
	jq > "${TMPDIR}/scorecard.json" <<< '{}' \
		--arg resource "${TMPDIR}/resource.json" \
		--arg namespace "$test_namespace" \
		--arg version "$PGO_VERSION" \
	'{ scorecard: { bundle: "./package", plugins: [
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
	if operator-sdk scorecard --config "${TMPDIR}/scorecard.json"; then
		: # no-op to preserve the exit code above
	else
		echo "Error: $?"
		#kc logs --container='operator' --selector='name=postgres-operator' --since-time="$start"
		error=1
	fi
done

exit $error
