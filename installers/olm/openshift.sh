#!/usr/bin/env bash
# vim: set noexpandtab :
set -eu

yq_script="$( jq <<< '{}' \
	--arg description "$(< openshift.description.md )" \
'{
	"spec.description": $description,
	"spec.displayName": "Crunchy PostgreSQL for OpenShift",
}' )"
yq write --inplace --script=- <<< "$yq_script" "./package/${PGO_VERSION}/postgresoperator.v${PGO_VERSION}.clusterserviceversion.yaml"
