#!/bin/bash
# Copyright 2019 Crunchy Data Solutions, Inc.
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


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

$DIR/cleanup.sh

$PGO_CMD --namespace=$PGO_NAMESPACE get clusterrole pgopclusterrole 2> /dev/null
if [ $? -ne 0 ]
then
	echo ERROR: pgopclusterrole was not found in $PGO_NAMESPACE namespace
	echo Verify you ran install-rbac.sh
	exit
fi

#
# credentials for pgbackrest sshd 
#
$PGO_CMD --namespace=$PGO_NAMESPACE create secret generic pgo-backrest-repo-config \
	--from-file=config=$PGOROOT/conf/pgo-backrest-repo/config \
	--from-file=ssh_host_rsa_key=$PGOROOT/conf/pgo-backrest-repo/ssh_host_rsa_key \
	--from-file=authorized_keys=$PGOROOT/conf/pgo-backrest-repo/authorized_keys \
	--from-file=sshd_config=$PGOROOT/conf/pgo-backrest-repo/sshd_config

#
# credentials for pgo-apiserver TLS REST API
#
$PGO_CMD --namespace=$PGO_NAMESPACE delete secret tls pgo.tls

$PGO_CMD --namespace=$PGO_NAMESPACE create secret tls pgo.tls --key=$PGOROOT/conf/postgres-operator/server.key --cert=$PGOROOT/conf/postgres-operator/server.crt

$PGO_CMD --namespace=$PGO_NAMESPACE create configmap pgo-config \
	--from-file=$PGOROOT/conf/postgres-operator

#
# create the postgres-operator Deployment and Service
#
expenv -f $DIR/deployment.json | $PGO_CMD --namespace=$PGO_NAMESPACE create -f -
$PGO_CMD --namespace=$PGO_NAMESPACE create -f $DIR/service.json
