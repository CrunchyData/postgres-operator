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

$DIR/cleanup.sh

if [ -z "$NAMESPACE" ]; then
	echo "NAMESPACE not set, using default"
	export NAMESPACE=default
fi

if [ ! -d /data ]; then
	echo "create the HostPath directory"
	sudo mkdir /data
	sudo chmod 777 /data
	echo "create the test PV and PVC using the HostPath dir"
	$DIR/create-pv.sh
	sleep 3
	kubectl --namespace=$NAMESPACE create -f $DIR/crunchy-pvc.json
	sleep 3
fi

kubectl create configmap operator-conf \
	--from-file=$COROOT/conf/postgres-operator/backup-job.json \
	--from-file=$COROOT/conf/postgres-operator/pvc.json \
	--from-file=$COROOT/conf/postgres-operator/cluster/1 \
	--from-file=$COROOT/conf/postgres-operator/database/1 

envsubst < $DIR/deployment.json | kubectl --namespace=$NAMESPACE create -f -
