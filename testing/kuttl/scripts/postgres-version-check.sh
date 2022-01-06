#!/bin/bash

EXPECTED_VERSION=$1

CLUSTER=${CLUSTER:-default}

POSTGRES_POD=$(kubectl -n ${NAMESPACE} get pod --selector=postgres-operator.crunchydata.com/cluster=${CLUSTER},postgres-operator.crunchydata.com/instance-set=instance1 --no-headers -o custom-columns=":metadata.name")

# Grab the Postgres major version from `version()` output to compare to the provided value
VERSION=$(kubectl -n ${NAMESPACE} exec -i ${POSTGRES_POD} -c database -- psql -tc "SELECT version();" |
    xargs |
    cut -d' ' -f 2 |
    cut -d. -f 1)

if [[ "$VERSION" != "$EXPECTED_VERSION" ]]; then
    echo "Expected ${EXPECTED_VERSION} but got ${VERSION}"
    exit 1
fi
