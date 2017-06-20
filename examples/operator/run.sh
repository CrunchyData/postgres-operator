#!/bin/bash
# Copyright 2016 Crunchy Data Solutions, Inc.
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


if [ -z "$CO_NAMESPACE" ]; then
	echo "CO_NAMESPACE not set, using default"
	export CO_NAMESPACE=default
fi
if [ -z "$CO_CMD" ]; then
	echo "CO_CMD not set, using kubectl"
	export CO_CMD=kubectl
fi

$DIR/cleanup.sh

$CO_CMD --namespace=$CO_NAMESPACE get pvc crunchy-pvc
rc=$?

if [ ! $rc -eq 0 ]; then
	echo "crunchy-pvc does not exist...creating crunchy-pvc "
	$CO_CMD --namespace=$CO_NAMESPACE create -f $DIR/crunchy-pvc.json
	$DIR/create-pv.sh
else
	echo "crunchy-pvc already exists..."
fi

if [ ! -d /data ]; then
	echo "create the HostPath directory"
	sudo mkdir /data
	sudo chmod 777 /data
fi

$CO_CMD create configmap operator-conf \
	--from-file=$COROOT/conf/postgres-operator/backup-job.json \
	--from-file=$COROOT/conf/postgres-operator/pvc.json \
	--from-file=$COROOT/conf/postgres-operator/cluster/1

envsubst < $DIR/deployment.json | $CO_CMD --namespace=$CO_NAMESPACE create -f -

sleep 3
$CO_CMD get pod --selector=name=postgres-operator
