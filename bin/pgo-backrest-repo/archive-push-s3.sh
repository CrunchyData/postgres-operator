#!/bin/bash

awsKeySecret() {
    val=$(grep "$1" -m 1 /sshd/aws-s3-credentials.yaml | sed "s/^.*:\s*//")
    # remove leading and trailing whitespace
    val=$(echo -e "${val}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
    if [[ "$val" == "" ]]
    then
        echo "empty-$1"
    else
        echo "${val}"
    fi
}

PGBACKREST_REPO1_S3_KEY=$(awsKeySecret "aws-s3-key")
export PGBACKREST_REPO1_S3_KEY
PGBACKREST_REPO1_S3_KEY_SECRET=$(awsKeySecret "aws-s3-key-secret")
export PGBACKREST_REPO1_S3_KEY_SECRET

pgbackrest "$@"
