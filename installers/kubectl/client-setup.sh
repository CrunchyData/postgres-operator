#!/bin/bash

#  Copyright 2020 Crunchy Data Solutions, Inc.
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#       http://www.apache.org/licenses/LICENSE-2.0
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# This script should be run after the operator has been deployed
export PGO_OPERATOR_NAMESPACE="${PGO_OPERATOR_NAMESPACE:-pgo}"
PGO_USER_ADMIN="${PGO_USER_ADMIN:-pgouser-admin}"
PGO_CLIENT_VERSION="${PGO_CLIENT_VERSION:-v4.3.0-beta.3}"
PGO_CLIENT_URL="https://github.com/CrunchyData/postgres-operator/releases/download/${PGO_CLIENT_VERSION}"

# Checks operating system and determines which binary to download
UNAME_RESULT=$(uname)
if [[ "${UNAME_RESULT}" == "Linux" ]]
then
    BIN_NAME="pgo"
elif [[ "${UNAME_RESULT}" == "Darwin" ]]
then
    BIN_NAME="pgo-mac"
else
    echo "${UNAME_RESULT} is not supported, valid operating systems are: Linux, Darwin"
    echo "Exiting..."
    exit 1
fi

# Creates the output directory for files
OUTPUT_DIR="${HOME}/.pgo/${PGO_OPERATOR_NAMESPACE}"
mkdir -p "${OUTPUT_DIR}"
# lock down the directory to just this user
chmod a-rwx,u+rwx "${OUTPUT_DIR}"

echo "Operating System found is ${UNAME_RESULT}. Downloading ${BIN_NAME} client binary..."

FULL_PATH="${PGO_CLIENT_URL}/${BIN_NAME}"
curl -L "${FULL_PATH}" -o "${OUTPUT_DIR}/${BIN_NAME}"
chmod +x "${OUTPUT_DIR}/${BIN_NAME}"


# Check that the pgouser-admin secret exists
if [ -z "$(kubectl get secret -n ${PGO_OPERATOR_NAMESPACE} ${PGO_USER_ADMIN})" ]
then
    echo "${PGO_USER_ADMIN} Secret not found in namespace: ${PGO_OPERATOR_NAMESPACE}"
    echo "Please ensure that the PostgreSQL Operator has been installed."
    echo "Exiting..."
    exit 1
fi

# Check that the pgo.tls secret exists
if [ -z "$(kubectl get secret -n ${PGO_OPERATOR_NAMESPACE} pgo.tls)" ]
then
    echo "pgo.tls Secret not found in namespace: ${PGO_OPERATOR_NAMESPACE}"
    echo "Please ensure that the PostgreSQL Operator has been installed."
    echo "Exiting..."
    exit 1
fi

# Use the pgouser-admin secret to generate pgouser file
kubectl get secret -n "${PGO_OPERATOR_NAMESPACE}" "${PGO_USER_ADMIN}" -o 'go-template={{.data.username | base64decode }}:{{.data.password | base64decode}}' > $OUTPUT_DIR/pgouser
# ensure this file is locked down to the specific user running this
chmod a-rwx,u+rw "${OUTPUT_DIR}/pgouser"


# Use the pgo.tls secret to generate the client cert files
kubectl get secret -n "${PGO_OPERATOR_NAMESPACE}" pgo.tls -o 'go-template={{ index .data "tls.crt" | base64decode }}' > $OUTPUT_DIR/client.crt
kubectl get secret -n "${PGO_OPERATOR_NAMESPACE}" pgo.tls -o 'go-template={{ index .data "tls.key" | base64decode }}' > $OUTPUT_DIR/client.key
# ensure the files are locked down to the specific user running this
chmod a-rwx,u+rw "${OUTPUT_DIR}/client.crt" "${OUTPUT_DIR}/client.key"


echo "pgo client files have been generated, please add the following to your bashrc"
echo "export PATH=${OUTPUT_DIR}/${BIN_NAME}:\$PATH"
echo "export PGOUSER=${OUTPUT_DIR}/pgouser"
echo "export PGO_CA_CERT=${OUTPUT_DIR}/client.crt"
echo "export PGO_CLIENT_CERT=${OUTPUT_DIR}/client.crt"
echo "export PGO_CLIENT_KEY=${OUTPUT_DIR}/client.key"
