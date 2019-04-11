#!/bin/bash

# Copyright 2017 - 2019 Crunchy Data Solutions, Inc.
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

RED="\033[0;31m"
GREEN="\033[0;32m"
RESET="\033[0m"

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
CONTAINER_NAME='pgo-custom-ssl-container'

echo $DIR

function echo_err() {
    echo -e "${RED?}$(date) ERROR: ${1?}${RESET?}"
}

function echo_info() {
    echo -e "${GREEN?}$(date) INFO: ${1?}${RESET?}"
}

#Error if PGO_CMD not set
if [[ -z ${PGO_CMD} ]]
then
        echo_err "PGO_CMD is not set."
fi

#Error is PGO_NAMESPACE not set
if [[ -z ${PGO_NAMESPACE} ]]
then
        echo_err "PGO_NAMESPACE is not set."
fi

# If both PGO_CMD and PGO_NAMESPACE are set, config map can be created.
if [[ ! -z ${PGO_CMD} ]] && [[ ! -z ${PGO_NAMESPACE} ]]
then

# Cleanup old certs

rm -rf ${DIR?}/certs
rm -rf ${DIR?}/out
rm -f ${DIR?}/configs/ca.* ${DIR?}/configs/server.*

#Generate test certs

${DIR?}/ssl-creator.sh "testuser@crunchydata.com" "${CONTAINER_NAME?}" "${DIR}"
if [[ $? -ne 0 ]]
then
    echo_err "Failed to create certs, exiting.."
    exit 1
fi

cp ${DIR?}/certs/server.* ${DIR?}/configs
cp ${DIR?}/certs/ca.* ${DIR?}/configs



echo_info "PGO_NAMESPACE=${PGO_NAMESPACE}"

${PGO_CMD?} delete configmap pgo-custom-ssl-config -n ${PGO_NAMESPACE}

${PGO_CMD?} create --namespace=${PGO_NAMESPACE?} configmap pgo-custom-ssl-config \
    --from-file=${DIR?}/configs/ca.crt \
    --from-file=${DIR?}/configs/ca.crl \
    --from-file=${DIR?}/configs/server.crt \
    --from-file=${DIR?}/configs/server.key \
    --from-file=${DIR?}/configs/pg_hba.conf \
    --from-file=${DIR?}/configs/pg_ident.conf \
    --from-file=${DIR?}/configs/postgresql.conf

echo ""
echo "To connect via SSL, run the following once the DB is ready: "
echo "psql \"postgresql://testuser@${CONTAINER_NAME?}:5432/userdb?\
sslmode=verify-full&\
sslrootcert=$PGOROOT/examples/custom-config-ssl/certs/ca.crt&\
sslcrl=$PGOROOT/examples/custom-config-ssl/certs/ca.crl&\
sslcert=$PGOROOT/examples/custom-config-ssl/certs/client.crt&\
sslkey=$PGOROOT/examples/custom-config-ssl/certs/client.key\""
echo ""

fi
