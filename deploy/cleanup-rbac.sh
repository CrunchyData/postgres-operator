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

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get serviceaccount postgres-operator 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete serviceaccount postgres-operator
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get serviceaccount pgo-backrest 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete serviceaccount pgo-backrest
fi

IFS=', ' read -r -a array <<< "$NAMESPACE"

echo ""
echo "delete pgo-backrest ServiceAccount from each target namespace"

for ns in "${array[@]}"
do
	$PGO_CMD get sa pgo-backrest --namespace=$ns > /dev/null 2> /dev/null
	if [ $? -eq 0 ]
	then
		$PGO_CMD delete sa  pgo-backrest --namespace=$ns > /dev/null 2> /dev/null
	fi
done

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get clusterrole pgopclusterrole  
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete clusterrole pgopclusterrole 
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get clusterrole pgopclusterrolesecret
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete clusterrole pgopclusterrolesecret 
fi


$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get clusterrole pgopclusterrolecrd 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete clusterrole pgopclusterrolecrd
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get clusterrolebinding pgopclusterbinding-$PGO_OPERATOR_NAMESPACE  
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete clusterrolebinding pgopclusterbinding-$PGO_OPERATOR_NAMESPACE 
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get clusterrolebinding pgopclusterbindingcrd-$PGO_OPERATOR_NAMESPACE
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete clusterrolebinding pgopclusterbindingcrd-$PGO_OPERATOR_NAMESPACE
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get clusterrolebinding pgopclusterbindingsecret-$PGO_OPERATOR_NAMESPACE
if [ $? -eq 0 ]
then
    $PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete clusterrolebinding pgopclusterbindingsecret-$PGO_OPERATOR_NAMESPACE
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get role pgo-role 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete role pgo-role
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get rolebinding pgo-role-binding 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete rolebinding pgo-role-binding
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get role pgo-backrest-role 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete role pgo-backrest-role
fi

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE get rolebinding pgo-backrest-role-binding 
if [ $? -eq 0 ]
then
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete rolebinding pgo-backrest-role-binding
fi

sleep 5

