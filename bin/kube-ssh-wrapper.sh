#!/bin/bash

# Determine if connecting to the backrest repo or primary db
if [[ "${PGO_BACKREST_REPO}" == "true" ]]
then
    source "/tmp/pod_env.sh"
    TARGET_LABEL="role=master"
else
    TARGET_LABEL="pgo-backrest-repo=true"
fi

CLUSTER_LABEL="pg-cluster=${CLUSTER_NAME}"
CONTAINER_NAME="database"

opts=$(echo "$@" | grep -o "\-\-c.*")
pod=$(kubectl get pods --selector=${CLUSTER_LABEL},${TARGET_LABEL} --field-selector=status.phase=Running -o name)

exec kubectl exec -i "${pod}" -c ${CONTAINER_NAME} -- bash -c "pgbackrest ${opts}"
