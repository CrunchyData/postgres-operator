#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

kc() { kubectl --namespace="$NAMESPACE" "$@"; }

application_ownership="$( kc get "applications.app.k8s.io/$NAME" --output=json )"
application_ownership="$( jq <<< "$application_ownership" '{ metadata: {
	labels: { "app.kubernetes.io/name": .metadata.name },
	ownerReferences: [{
		apiVersion, kind, name: .metadata.name, uid: .metadata.uid
	}]
} }' )"

existing="$( kc get clusterrole/pgo-cluster-role --output=json 2> /dev/null || true )"

if [ -n "$existing" ]; then
	>&2 echo ERROR: Crunchy PostgreSQL Operator is already installed in another namespace
	exit 1
fi

application_icon="$( base64 --wrap=0 /opt/postgres-operator/favicon.png )"
application_metadata="$( jq <<< '{}' --arg icon "$application_icon" '{ metadata: {
	annotations: { "kubernetes-engine.cloud.google.com/icon": "data:image/png;base64,\($icon)" }
} }' )"

kc patch "applications.app.k8s.io/$NAME" --type=merge --patch="$application_metadata"

/usr/bin/ansible-playbook \
	--extra-vars 'kubernetes_in_cluster=true' \
	--extra-vars 'config_path=/etc/ansible/values.yaml' \
	--inventory    /opt/postgres-operator/ansible/inventory.yaml \
	--tags=install /opt/postgres-operator/ansible/main.yml

resources=(
	clusterrole/pgo-cluster-role
	clusterrolebinding/pgo-cluster-role
	configmap/pgo-config
	deployment/postgres-operator
	role/pgo-backrest-role
	role/pgo-pg-role
	role/pgo-role
	role/pgo-target-role
	rolebinding/pgo-backrest-role-binding
	rolebinding/pgo-pg-role-binding
	rolebinding/pgo-role
	rolebinding/pgo-target-role-binding
	secret/pgo.tls
	secret/pgorole-pgoadmin
	secret/pgouser-admin
	service/postgres-operator
	serviceaccount/pgo-backrest
	serviceaccount/pgo-default
	serviceaccount/pgo-pg
	serviceaccount/pgo-target
	serviceaccount/postgres-operator
)

for resource in "${resources[@]}"; do
	kind="${resource%/*}"
	name="${resource#*/}"

	for _ in $(seq 5); do
		if [ "$( kc get "$kind" --field-selector="metadata.name=$name" --output=name )" ]
		then break
		else sleep 1s
		fi
	done

	kc patch "$resource" --type=strategic --patch="$application_ownership"
done
