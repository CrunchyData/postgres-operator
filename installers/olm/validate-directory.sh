#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

if command -v oc > /dev/null; then
	kubectl() { oc "$@"; }
	kubectl version
else
	kubectl version --short
fi

push_trap_exit() {
	local -a array
	eval "array=($(trap -p EXIT))"
	# shellcheck disable=SC2064
	trap "$1;${array[2]-}" EXIT
}

validate_bundle_directory() {
	local directory="$1"
	local namespace

	namespace=$(kubectl create --filename=- --output='go-template={{.metadata.name}}' <<< '{
		"apiVersion": "v1", "kind": "Namespace",
		"metadata": {
			"generateName": "olm-test-",
			"labels": { "olm-test": "bundle-directory" }
		}
	}')
	echo 'namespace "'"${namespace}"'" created'
	push_trap_exit "kubectl delete namespace '${namespace}'"

	# https://olm.operatorframework.io/docs/best-practices/common/
	# https://sdk.operatorframework.io/docs/advanced-topics/scorecard/scorecard/
	operator-sdk scorecard --namespace="${namespace}" "${directory}"
}

validate_bundle_directory "$@"
