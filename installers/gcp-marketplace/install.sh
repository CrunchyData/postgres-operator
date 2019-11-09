#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

kc() { kubectl --namespace="$NAMESPACE" "$@"; }

application_metadata="$( kc get "applications.app.k8s.io/$NAME" --output=json )"
application_metadata="$( jq <<< "$application_metadata" '{ metadata: {
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

/usr/bin/ansible-playbook --tags=install /opt/postgres-operator/ansible/main.yml

resources=(
	clusterrole/pgo-cluster-role
	clusterrolebinding/pgo-cluster-role
	configmap/pgo-config
	deployment/postgres-operator
	role/pgo-role
	rolebinding/pgo-role
	secret/pgo.tls
	secret/pgo-backrest-repo-config
	secret/pgorole-pgoadmin
	secret/pgouser-admin
	service/postgres-operator
	serviceaccount/postgres-operator
)

for resource in "${resources[@]}"; do
	kc patch "$resource" --type=strategic --patch="$application_metadata"
done
