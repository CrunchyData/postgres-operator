#!/bin/bash

# Determine if connecting to the pgBackRest dedicated repository host or the primary database.  Specifically,
# set the proper label to target one or the other when executing the kubectl command in place of SSH.
if [[ "${PGO_BACKREST_REPO}" == "true" ]]
then
    # Add required env vars to environment. Specifically sets the CLUSTER_NAME when running on a dedicated
    # pgBakcRest reposotiory host, as required to set variable CLUSTER_LABEL label below
    source "/tmp/pod_env.sh"
    TARGET_LABEL="role=master"
else
    TARGET_LABEL="pgo-backrest-repo=true"
fi

# Set the name of the cluster the pgBackRest command is being executed for, as well as the name of the
# container being targeted for the command
CLUSTER_LABEL="pg-cluster=${CLUSTER_NAME}"
CONTAINER_NAME="database"

# Drop any SSH options to obtain the pgBackRest command only
backrest_cmd=$(sed -n "s/^.*\(pgbackrest \)/\1/p" <<< "$@")

# Find the pod to exec into using the labels defined above, and ensure the pod is running.  There might
# still be a terminating pod with the same labels in certain situations, e.g. when an old pgBackRest repo 
# pod is still terminating, and a new pgBackRest repo pod is up and running during a pgBackRest restore
pod=$(kubectl get pods --selector="${CLUSTER_LABEL}","${TARGET_LABEL}" --no-headers | awk '/Running/ {print $1}')

# Execute the kubectl command in place of SSH
exec kubectl exec -i "${pod}" -c "${CONTAINER_NAME}" -- bash -c "${backrest_cmd}"
