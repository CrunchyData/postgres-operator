#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

if command -v oc >/dev/null; then
	kubectl() { oc "$@"; }
	kubectl version
else
	kubectl version --short
fi

catalog_source() (
	source_namespace="$1"
	source_name="$2"
	index_image="$3"

	kc() { kubectl --namespace="$source_namespace" "$@"; }
	kc get namespace "$source_namespace" --output=jsonpath='{""}' 2>/dev/null ||
		kc create namespace "$source_namespace"

	# See https://godoc.org/github.com/operator-framework/api/pkg/operators/v1alpha1#CatalogSource
	source_json=$(jq --null-input \
		--arg name "${source_name}" \
		--arg image "${index_image}" \
	'{
		apiVersion: "operators.coreos.com/v1alpha1", kind: "CatalogSource",
		metadata: { name: $name },
		spec: {
			displayName: "Test Registry",
			sourceType: "grpc", image: $image
		}
	}')
	kc create --filename=- <<< "$source_json"

	# Wait for Pod to exist and be healthy.
	for _ in $(seq 10); do
		[ '[]' != "$( kc get pod --selector="olm.catalogSource=${source_name}" --output=jsonpath='{.items}' )" ] &&
			break || sleep 1s
	done
	if ! kc wait --for='condition=ready' --timeout='30s' pod --selector="olm.catalogSource=${source_name}"; then
		kc logs --previous --tail='-1' --selector="olm.catalogSource=${source_name}"
	fi
)

operator_group() (
	group_namespace="$1"
	group_name="$2"
	target_namespaces=("${@:3}")

	kc() { kubectl --namespace="$group_namespace" "$@"; }
	kc get namespace "$group_namespace" --output=jsonpath='{""}' 2>/dev/null ||
		kc create namespace "$group_namespace"

	group_json="$( jq <<< '{}' --arg name "$group_name" '{
		apiVersion: "operators.coreos.com/v1", kind: "OperatorGroup",
		metadata: { "name": $name },
		spec: { targetNamespaces: [] }
	}' )"

	for ns in "${target_namespaces[@]}"; do
		group_json="$( jq <<< "$group_json" --arg namespace "$ns" '.spec.targetNamespaces += [ $namespace ]' )"
	done

	kc create --filename=- <<< "$group_json"
)

operator() (
	bundle_directory="$1" index_image="$2"
	operator_namespace="$3"
	target_namespaces=("${@:4}")

	package_name=$(yq \
		--raw-output '.annotations["operators.operatorframework.io.bundle.package.v1"]' \
		"${bundle_directory}"/*/annotations.yaml)
	channel_name=$(yq \
		--raw-output '.annotations["operators.operatorframework.io.bundle.channels.v1"]' \
		"${bundle_directory}"/*/annotations.yaml)
	csv_name=$(yq --raw-output '.metadata.name' \
		"${bundle_directory}"/*/*.clusterserviceversion.yaml)

	kc() { kubectl --namespace="$operator_namespace" "$@"; }

	catalog_source "$operator_namespace" olm-catalog-source "${index_image}"
	operator_group "$operator_namespace" olm-operator-group "${target_namespaces[@]}"

	# Create a Subscription to install the operator.
	# See https://godoc.org/github.com/operator-framework/api/pkg/operators/v1alpha1#Subscription
	subscription_json=$(jq --null-input \
		--arg channel "$channel_name" \
		--arg namespace "$operator_namespace" \
		--arg package "$package_name" \
		--arg version "$csv_name" \
	'{
		apiVersion: "operators.coreos.com/v1alpha1", kind: "Subscription",
		metadata: { name: $package },
		spec: {
			name: $package,
			sourceNamespace: $namespace,
			source: "olm-catalog-source",
			startingCSV: $version,
			channel: $channel
		}
	}')
	kc create --filename=- <<< "$subscription_json"

	# Wait for the InstallPlan to exist and be healthy.
	for _ in $(seq 10); do
		[ '[]' != "$( kc get installplan --output=jsonpath="{.items}" )" ] &&
			break || sleep 1s
	done
	if ! kc wait --for='condition=installed' --timeout='30s' installplan --all; then
		subscription_uid="$( kc get subscription "$package_name" --output=jsonpath='{.metadata.uid}' )"
		installplan_json="$( kc get installplan --output=json )"

		jq <<< "$installplan_json" --arg uid "$subscription_uid" \
			'.items[] | select(.metadata.ownerReferences[] | select(.uid == $uid)).status.conditions'
		exit 1
	fi

	# Wait for Deployment to exist and be healthy.
	for _ in $(seq 10); do
		[ '[]' != "$( kc get deploy --selector="olm.owner=$csv_name" --output=jsonpath='{.items}' )" ] &&
			break || sleep 1s
	done
	if ! kc wait --for='condition=available' --timeout='30s' deploy --selector="olm.owner=$csv_name"; then
		kc describe pod --selector="olm.owner=$csv_name"

		crashed_containers="$( kc get pod --selector="olm.owner=$csv_name" --output=json )"
		crashed_containers="$( jq <<< "$crashed_containers" --raw-output \
			'.items[] | {
				pod: .metadata.name,
				container: .status.containerStatuses[] | select(.restartCount > 0).name
			} | [.pod, .container] | @tsv' )"

		test -z "$crashed_containers" || while IFS=$'\t' read -r pod container; do
			echo; echo "$pod/$container" restarted:
			kc logs --container="$container" --previous --tail='-1' "pod/$pod"
		done <<< "$crashed_containers"

		exit 1
	fi
)

"$@"
