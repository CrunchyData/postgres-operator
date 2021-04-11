#!/bin/bash
# Copyright 2017 - 2021 Crunchy Data Solutions, Inc.
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

test="${PGO_CONF_DIR:?Need to set PGO_CONF_DIR env variable}"

# awsKeySecret is borrowed from the legacy way to pull out the AWS s3
# credentials in an environmental variable. This is only here while we
# transition away from whatever this was
awsKeySecret() {
    val=$(grep "$1" -m 1 "${PGOROOT}/conf/pgo-backrest-repo/aws-s3-credentials.yaml" | sed "s/^.*:\s*//")
    # remove leading and trailing whitespace
    val=$(echo -e "${val}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')
    if [[ "$val" != "" ]]
		then
        echo "${val}"
    fi
}

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

$DIR/cleanup.sh

if [[ "${PGO_NAMESPACE_MODE:-dynamic}" != "disabled" ]]
then
	$PGO_CMD get clusterrole pgo-cluster-role 2> /dev/null > /dev/null
	if [ $? -ne 0 ]
	then
		echo ERROR: pgo-cluster-role was not found
		echo Verify you ran install-rbac.sh
		exit
	fi
fi

# credentials for pgbackrest sshd
pgbackrest_aws_s3_key=$(awsKeySecret "aws-s3-key")
pgbackrest_aws_s3_key_secret=$(awsKeySecret "aws-s3-key-secret")
pgbackrest_gcs_key=$(awsKeySecret "gcs-key")

if [[ ! -z $pgbackrest_aws_s3_key ]] || [[ ! -z $pgbackrest_aws_s3_key_secret ]] || [[ ! -z $pgbackrest_gcs_key ]]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create secret generic pgo-backrest-repo-config \
		--from-literal=aws-s3-key="${pgbackrest_aws_s3_key}" \
		--from-literal=aws-s3-key-secret="${pgbackrest_aws_s3_key_secret}" \
		--from-literal=gcs-key="${pgbackrest_gcs_key}"
fi

#
# credentials for pgo-apiserver TLS REST API
#
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get secret pgo.tls > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete secret pgo.tls
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create secret tls pgo.tls --key=${PGOROOT}/conf/postgres-operator/server.key --cert=${PGOROOT}/conf/postgres-operator/server.crt

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create configmap pgo-config \
	--from-file=${PGOROOT}/conf/postgres-operator/pgo.yaml \
	--from-file=${PGO_CONF_DIR}/pgo-configs


#
# check if custom port value is set, otherwise set default values
#

if [[ -z ${PGO_APISERVER_PORT} ]]
then
        echo "PGO_APISERVER_PORT is not set. Setting to default port value of 8443."
		export PGO_APISERVER_PORT=8443
fi

export PGO_APISERVER_SCHEME="HTTPS"

# check if TLS is disabled. If it is, both ensure that the probes occur over
# HTTP, and then also set TLS_NO_VERIFY to true as well, given TLS is disabled
if [[ "${DISABLE_TLS}" == "true" ]]
then
  export PGO_APISERVER_SCHEME="HTTP"
  export TLS_NO_VERIFY="true"
fi

#
# create the postgres-operator Deployment and Service
#
envsubst < $DIR/deployment.json | $PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create -f -
envsubst < $DIR/service.json | $PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create -f -
