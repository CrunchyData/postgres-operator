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

$CO_CMD --namespace=$CO_NAMESPACE get secret/pgo-backrest-repo-config 2> /dev/null

if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete secret/pgo-backrest-repo-config
fi

$CO_CMD --namespace=$CO_NAMESPACE get secret/pgo-auth-secret 2> /dev/null
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete secret/pgo-auth-secret
fi

$CO_CMD --namespace=$CO_NAMESPACE get configmap/pgo-config 2> /dev/null
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete configmap/pgo-config
fi

$CO_CMD --namespace=$CO_NAMESPACE get service/postgres-operator 2> /dev/null
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete service/postgres-operator
fi

$CO_CMD --namespace=$CO_NAMESPACE get deployment/postgres-operator 2> /dev/null
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete deployment/postgres-operator
fi

sleep 5
