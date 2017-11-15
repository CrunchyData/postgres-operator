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

$CO_CMD --namespace=$CO_NAMESPACE create configmap apiserver-conf \
	--from-file=$COROOT/conf/apiserver/pgouser \
	--from-file=$COROOT/conf/apiserver/pgo.yaml \
	--from-file=$COROOT/conf/apiserver/pgo.csvload-template.json \
	--from-file=$COROOT/conf/apiserver/pgo.lspvc-template.json 

$CO_CMD --namespace=$CO_NAMESPACE create configmap operator-conf \
	--from-file=$COROOT/conf/postgres-operator/backup-job.json \
	--from-file=$COROOT/conf/postgres-operator/pvc.json \
	--from-file=$COROOT/conf/postgres-operator/pvc-storageclass.json \
	--from-file=$COROOT/conf/postgres-operator/cluster/1

envsubst < $DIR/deployment.json | $CO_CMD --namespace=$CO_NAMESPACE create -f -

$CO_CMD --namespace=$CO_NAMESPACE create -f $DIR/service.json

