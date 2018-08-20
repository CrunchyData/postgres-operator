#!/bin/bash
# Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

if [ "$CO_CMD" = "kubectl" ]; then
	NS="--namespace=$CO_NAMESPACE"
fi

$CO_CMD $NS create secret generic apiserver-conf-secret \
        --from-file=server.crt=$COROOT/conf/apiserver/server.crt \
        --from-file=server.key=$COROOT/conf/apiserver/server.key \
        --from-file=pgouser=$COROOT/conf/apiserver/pgouser \
        --from-file=pgorole=$COROOT/conf/apiserver/pgorole \
        --from-file=pgo.yaml=$COROOT/conf/apiserver/pgo.yaml \
        --from-file=pgo.load-template.json=$COROOT/conf/apiserver/pgo.load-template.json \
        --from-file=pgo.lspvc-template.json=$COROOT/conf/apiserver/pgo.lspvc-template.json

$CO_CMD $NS create configmap operator-conf \
	--from-file=$COROOT/conf/postgres-operator/backup-job.json \
	--from-file=$COROOT/conf/postgres-operator/pgo-ingest-watch-job.json \
	--from-file=$COROOT/conf/postgres-operator/rmdata-job.json \
	--from-file=$COROOT/conf/postgres-operator/pvc.json \
	--from-file=$COROOT/conf/postgres-operator/pvc-storageclass.json \
	--from-file=$COROOT/conf/postgres-operator/pvc-matchlabels.json \
	--from-file=$COROOT/conf/postgres-operator/backrest-job.json \
	--from-file=$COROOT/conf/postgres-operator/backrest-restore-configmap.json \
	--from-file=$COROOT/conf/postgres-operator/backrest-restore-job.json \
	--from-file=$COROOT/conf/postgres-operator/cluster/1

if [ "$CO_UI" = "true" ]; then
$CO_CMD $NS create configmap pgo-ui-conf \
	--from-file=$COROOT/conf/pgo-ui/config.json \
        --from-file=$COROOT/conf/apiserver/server.crt \
        --from-file=$COROOT/conf/apiserver/server.key 

	expenv -f $DIR/deployment-with-ui.json | $CO_CMD $NS create -f -
	$CO_CMD $NS create -f $DIR/service-with-ui.json
else
	expenv -f $DIR/deployment.json | $CO_CMD $NS create -f -
	$CO_CMD $NS create -f $DIR/service.json
fi

