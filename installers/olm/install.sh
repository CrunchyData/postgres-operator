#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

if command -v oc >/dev/null; then
	kubectl() { oc "$@"; }
	kubectl version
elif ! command -v kubectl >/dev/null; then
	# Use a version of `kubectl` that matches the Kubernetes server.
	eval "kubectl() { kubectl-$( kubectl-1.16 version --output=json |
		jq --raw-output '.serverVersion | .major + "." + .minor')"' "$@"; }'
	kubectl version --short
fi

catalog_source() (
	source_namespace="$1"
	source_name="$2"
	registry_namespace="$3"
	registry_name="$4"

	kc() { kubectl --namespace="$source_namespace" "$@"; }
	kc get namespace "$source_namespace" --output=jsonpath='{""}' 2>/dev/null ||
		kc create namespace "$source_namespace"

	# See https://godoc.org/github.com/operator-framework/api/pkg/operators/v1alpha1#CatalogSource
	source_json="$( jq <<< '{}' \
		--arg name "$source_name" \
		--arg registry "${registry_name}.${registry_namespace}" \
	'{
		apiVersion: "operators.coreos.com/v1alpha1", kind: "CatalogSource",
		metadata: { name: $name },
		spec: {
			displayName: "Test Registry",
			sourceType: "grpc", address: "\($registry):50051"
		}
	}' )"
	kc create --filename=- <<< "$source_json"
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

registry() (
	registry_namespace="$1"
	registry_name="$2"

	package_name="$( yq --raw-output '.packageName' ./package/*.package.yaml )"

	kc() { kubectl --namespace="$registry_namespace" "$@"; }
	kc get namespace "$registry_namespace" --output=jsonpath='{""}' 2>/dev/null ||
		kc create namespace "$registry_namespace"

	# Create a registry based on a ConfigMap containing the package with subdirectories encoded as dashes.
	#
	# There is a simpler `configmap-server` and CatalogSource.sourceType of `configmap`, but those only
	# support a subset of possible bundle files. Notably, Service files are not supported at this time.
	#
	# See https://godoc.org/github.com/operator-framework/operator-registry/pkg/sqlite#ConfigMapLoader
	# and https://godoc.org/github.com/operator-framework/operator-registry/pkg/sqlite#DirectoryLoader
	deployment_json="$( jq <<< '{}' \
		--arg name "$registry_name" \
		--arg package "$package_name" \
		--arg script '
			find -L /mnt/package -name ".*" -prune -o -type f -print | while IFS="" read s ; do
				t="${s#/mnt/package/}"; t="${t//--//}"
				install -D -m 644 "$s" "manifests/$PACKAGE_NAME/$t"
			done
			/usr/bin/initializer
			exec /usr/bin/registry-server
		' \
	'{
		apiVersion: "apps/v1", kind: "Deployment",
		metadata: { name: $name },
		spec: {
			selector: { matchLabels: { name: $name } },
			template: {
				metadata: { labels: { name: $name } },
				spec: {
					containers: [{
						name: "registry",
						image: "quay.io/openshift/origin-operator-registry:latest",
						imagePullPolicy: "IfNotPresent",
						command: ["bash", "-ec"], args: [ $script ],
						env: [{ name: "PACKAGE_NAME", value: $package }],
						volumeMounts: [{ mountPath: "/mnt/package", name: "package" }]
					}],
					volumes: [{ name: "package", configMap: { name: $name } }]
				}
			}
		}
	}' )"
	kc create configmap "$registry_name" $(
		find ./package -type f | while IFS='' read s ; do
			t="${s#./package/}"; t="${t//\//--}"
			echo "--from-file=$t=$s"
		done
	)
	kc create --filename=- <<< "$deployment_json"
	kc expose deploy "$registry_name" --port=50051

	if ! kc wait --for='condition=available' --timeout='90s' deploy "$registry_name"; then
		kc logs --selector="name=$registry_name" --tail='-1' --previous ||
		kc logs --selector="name=$registry_name" --tail='-1'
		exit 1
	fi
)

operator() (
	operator_namespace="$1"
	target_namespaces=("${@:2}")

	package_name="$( yq --raw-output '.packageName' ./package/*.package.yaml )"
	package_channel_name="$( yq --raw-output '.defaultChannel' ./package/*.package.yaml )"
	package_csv_name="$( yq \
		--raw-output --arg channel "$package_channel_name" \
		'.channels[] | select(.name == $channel).currentCSV' \
		./package/*.package.yaml )"

	kc() { kubectl --namespace="$operator_namespace" "$@"; }

	registry "$operator_namespace" olm-registry
	catalog_source "$operator_namespace" olm-catalog-source "$operator_namespace" olm-registry
	operator_group "$operator_namespace" olm-operator-group "${target_namespaces[@]}"

	# Create a Subscription to install the operator.
	# See https://godoc.org/github.com/operator-framework/api/pkg/operators/v1alpha1#Subscription
	subscription_json="$( jq <<< '{}' \
		--arg channel "$package_channel_name" \
		--arg namespace "$operator_namespace" \
		--arg package "$package_name" \
		--arg version "$package_csv_name" \
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
	}' )"
	kc create --filename=- <<< "$subscription_json"

	# Wait for the InstallPlan to exist and be healthy.
	for i in $(seq 10); do
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
	for i in $(seq 10); do
		[ '[]' != "$( kc get deploy --selector="olm.owner=$package_csv_name" --output=jsonpath='{.items}' )" ] &&
			break || sleep 1s
	done
	if ! kc wait --for='condition=available' --timeout='30s' deploy --selector="olm.owner=$package_csv_name"; then
		kc describe pod --selector="olm.owner=$package_csv_name"

		crashed_containers="$( kc get pod --selector="olm.owner=$package_csv_name" --output=json )"
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

	exit 0

	# Create a client Pod from which commands can be executed.
	client_image="$( kc get deploy --selector="olm.owner=$package_csv_name" --output=json |
		jq --raw-output '.items[0].spec.template.spec.containers[] | select(.name == "operator").image' )"
	client_image="${client_image/postgres-operator/pgo-client}"

	subscription_ownership="$( kc get "subscription.operators.coreos.com/$package_name" --output=json )"
	subscription_ownership="$( jq <<< "$subscription_ownership" '{
		apiVersion, kind, name: .metadata.name, uid: .metadata.uid
	}' )"

	role_secret_json="$( jq <<< '{}' \
		--arg rolename admin \
	'{
		apiVersion: "v1", kind: "Secret",
		metadata: {
			name: "pgorole-\($rolename)",
			labels: { "pgo-pgorole": "true", rolename: $rolename }
		},
		stringData: { permissions: "*", rolename: $rolename }
	}' )"
	user_secret_json="$( jq <<< '{}' \
		--arg password "${RANDOM}${RANDOM}${RANDOM}" \
		--arg rolename admin \
		--arg username admin \
	'{
		apiVersion: "v1", kind: "Secret",
		metadata: {
			name: "pgouser-\($username)",
			labels: { "pgo-pgouser": "true", username: $username }
		},
		stringData: { username: $username, password: $password, roles: $rolename }
	}' )"

	client_job_json="$( jq <<< '{}' \
		--arg image "$client_image" \
		--argjson subscription "$subscription_ownership" \
	'{
		apiVersion: "batch/v1", kind: "Job",
		metadata: { name: "pgo-client", ownerReferences: [ $subscription ] },
		spec: { template: { spec: {
			dnsPolicy: "ClusterFirst",
			restartPolicy: "OnFailure",
			containers: [{
				name: "client",
				image: $image,
				imagePullPolicy: "IfNotPresent",
				command: ["tail", "-f", "/dev/null"],
				env: [
					{ name: "PGO_APISERVER_URL", value: "https://postgres-operator:8443" },
					{ name: "PGOUSERNAME", valueFrom: { secretKeyRef: { name: "pgouser-admin", key: "username" } } },
					{ name: "PGOUSERPASS", valueFrom: { secretKeyRef: { name: "pgouser-admin", key: "password" } } },
					{ name: "PGO_CA_CERT",     value: "/etc/pgo/certificates/tls.crt" },
					{ name: "PGO_CLIENT_CERT", value: "/etc/pgo/certificates/tls.crt" },
					{ name: "PGO_CLIENT_KEY",  value: "/etc/pgo/certificates/tls.key" }
				],
				volumeMounts: [{ mountPath: "/etc/pgo/certificates", name: "certificates" }]
			}],
			volumes: [{ name: "certificates", secret: { secretName: "pgo.tls" } }]
		} } }
	}' )"
	kc expose deploy postgres-operator
	kc create --filename=- <<< "$role_secret_json"
	kc create --filename=- <<< "$user_secret_json"
	kc create --filename=- <<< "$client_job_json"
)

scorecard() (
	operator_namespace="$1"
	sdk_version="$2"

	kc() { kubectl --namespace="$operator_namespace" "$@"; }

	# Create a Secret that contains a `kubectl` configuration file to authenticate with `scorecard-proxy`.
	# See https://github.com/operator-framework/operator-sdk/blob/master/doc/test-framework/scorecard.md
	scorecard_username="$( jq <<< '{}' \
		--arg namespace "$operator_namespace" \
	'{ apiVersion: "", kind:"", "uid":"", name: "scorecard", Namespace: $namespace }' )"
	scorecard_kubeconfig="$( jq <<< '{}' \
		--arg namespace "$operator_namespace" \
		--arg username "$scorecard_username" \
	'{
		apiVersion: "v1", kind: "Config",
		clusters: [{
			name: "proxy-server",
			cluster: {
				server: "http://\($username | @base64)@localhost:8889",
				"insecure-skip-tls-verify": true
			}
		}],
		users: [{
			name: "admin/proxy-server",
			user: {
				username: ($username | @base64),
				password: "unused"
			}
		}],
		contexts: [{
			name: "\($namespace)/proxy-server",
			context: {
				cluster: "proxy-server",
				user: "admin/proxy-server"
			}
		}],
		"current-context": "\($namespace)/proxy-server",
		preferences: {}
	}' )"
	kc delete secret scorecard-kubeconfig --ignore-not-found
	kc create secret generic scorecard-kubeconfig --from-literal="kubeconfig=$scorecard_kubeconfig"

	# Inject a `scorecard-proxy` Container into the main Deployment and configure other containers
	# to make Kubernetes API calls through it.
	jq_filter='
		.spec.template.spec.volumes += [{
			name: "scorecard-kubeconfig",
			secret: {
				secretName: "scorecard-kubeconfig",
				items: [{ key: "kubeconfig", path: "config" }]
			}
		}] |
		.spec.template.spec.containers[].volumeMounts += [{
			name: "scorecard-kubeconfig",
			mountPath: "/scorecard-secret"
		}] |
		.spec.template.spec.containers[].env += [{
			name: "KUBECONFIG",
			value: "/scorecard-secret/config"
		}] |
		.spec.template.spec.containers += [{
			name: "scorecard-proxy",
			image: $image, imagePullPolicy: "Always",
			env: [{
				name: "WATCH_NAMESPACE",
				valueFrom: { fieldRef: { apiVersion: "v1", fieldPath: "metadata.namespace" } }
			}],
			ports: [{ name: "proxy", containerPort: 8889 }]
		}] |
	.'
	KUBE_EDITOR="yq --in-place --yaml-roundtrip \
		--arg image 'quay.io/operator-framework/scorecard-proxy:v$sdk_version' \
		'$jq_filter' \
	" kc edit deploy postgres-operator

	kc rollout status deploy postgres-operator --watch
)

"$@"
