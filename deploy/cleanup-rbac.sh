#!/bin/bash

# Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

# delete existing PGO SCC (SCC commands require 'oc' in place of 'kubectl'
oc get scc pgo  > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
        oc delete scc pgo
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get serviceaccount postgres-operator  > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete serviceaccount postgres-operator
fi

$PGO_CMD get clusterrole pgo-cluster-role   > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD delete clusterrole pgo-cluster-role
fi

$PGO_CMD get clusterrolebinding pgo-cluster-role   > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD delete clusterrolebinding pgo-cluster-role  > /dev/null 2> /dev/null
fi

$PGO_CMD -n $PGO_OPERATOR_NAMESPACE get role pgo-role   > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD -n $PGO_OPERATOR_NAMESPACE delete role pgo-role
fi

$PGO_CMD -n $PGO_OPERATOR_NAMESPACE get rolebinding pgo-role   > /dev/null 2> /dev/null
if [ $? -eq 0 ]
then
	$PGO_CMD -n $PGO_OPERATOR_NAMESPACE delete rolebinding pgo-role  > /dev/null
fi


sleep 5
