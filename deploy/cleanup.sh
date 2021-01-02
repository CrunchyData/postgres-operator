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

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get secret/pgo-backrest-repo-config 2> /dev/null

if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete secret/pgo-backrest-repo-config
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get secret pgo.tls 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete secret pgo.tls
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get configmap/pgo-config 2> /dev/null > /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete configmap/pgo-config
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get service/postgres-operator 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete service/postgres-operator
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get deployment/postgres-operator 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete deployment/postgres-operator
	for (( ; ; ))
	do
		echo "checking for postgres-operator pod..."
		lines=`$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get pod --selector=name=postgres-operator --ignore-not-found=true --no-headers | wc -l`

		if [ $lines -eq 0 ]
		then
			echo postgres-operator pod is gone
			break
		elif [ $lines -eq 1 ]
		then
			echo postgres-operator is out there
		fi
		sleep 3
	done
fi

