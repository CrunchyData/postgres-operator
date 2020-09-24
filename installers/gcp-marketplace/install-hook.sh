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

existing="$( kc get deployment/postgres-operator --output=json 2> /dev/null || true )"

if [ -n "$existing" ]; then
	>&2 echo ERROR: Crunchy PostgreSQL Operator is already installed in this namespace
	exit 1
fi

install_values="$( /bin/config_env.py envsubst < /opt/postgres-operator/values.yaml )"
installer="$( /bin/config_env.py envsubst < /opt/postgres-operator/install-job.yaml )"

kc create --filename=/dev/stdin <<< "$installer"
kc patch job/install-postgres-operator --type=strategic --patch="$application_ownership"

job_ownership="$( kc get job/install-postgres-operator --output=json )"
job_ownership="$( jq <<< "$job_ownership" '{ metadata: {
	labels: { "app.kubernetes.io/name": .metadata.labels["app.kubernetes.io/name"] },
	ownerReferences: [{
		apiVersion, kind, name: .metadata.name, uid: .metadata.uid
	}]
} }' )"

kc create secret generic install-postgres-operator --from-file=values.yaml=/dev/stdin <<< "$install_values"
kc patch secret/install-postgres-operator --type=strategic --patch="$job_ownership"

# Wait for either status condition then terminate the other.
kc wait --for=condition=complete --timeout=5m job/install-postgres-operator &
kc wait --for=condition=failed --timeout=5m job/install-postgres-operator &
wait -n
kill -s INT %% 2> /dev/null || true

kc logs --selector=job-name=install-postgres-operator --tail=-1
test 'Complete' = "$( kc get job/install-postgres-operator --output=jsonpath='{.status.conditions[*].type}' )"

exec /opt/postgres-operator/cloud-marketplace-tools/bin/create_manifests.sh "$@"
