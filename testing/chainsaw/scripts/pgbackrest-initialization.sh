#!/bin/bash

EXPECTED_STATUS=$1
EXPECTED_NUM_BACKUPS=$2

CLUSTER=${CLUSTER:-default}

INFO=$(kubectl -n "${NAMESPACE}" exec "statefulset.apps/${CLUSTER}-repo-host" -c pgbackrest -- pgbackrest info)

# Grab the `status` line from `pgbackrest info`, remove whitespace with `xargs`,
# and trim the string to only include the status in order to 
# validate the status matches the expected status.
STATUS=$(grep "status" <<< "$INFO" | xargs | cut -d' ' -f 2)
if [[ "$STATUS" != "$EXPECTED_STATUS" ]]; then
    echo "Expected ${EXPECTED_STATUS} but got ${STATUS}"
    exit 1
fi

# Count the lines with `full backup` to validate that the expected number of backups are found.
NUM_BACKUPS=$(grep -c "full backup:" <<< "$INFO")
if [[ "$NUM_BACKUPS" != "$EXPECTED_NUM_BACKUPS" ]]; then
    echo "Expected ${EXPECTED_NUM_BACKUPS} but got ${NUM_BACKUPS}"
    exit 1
fi
