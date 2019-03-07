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

$CO_CMD --namespace=$CO_NAMESPACE get serviceaccount postgres-operator 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete serviceaccount postgres-operator
fi

$CO_CMD --namespace=$CO_NAMESPACE get serviceaccount pgo-backrest 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete serviceaccount pgo-backrest
fi

$CO_CMD --namespace=$CO_NAMESPACE get clusterrole pgopclusterrole  
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete clusterrole pgopclusterrole 
fi

$CO_CMD --namespace=$CO_NAMESPACE get clusterrole pgopclusterrolecrd 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete clusterrole pgopclusterrolecrd
fi

$CO_CMD --namespace=$CO_NAMESPACE get clusterrolebinding pgopclusterbinding  
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete clusterrolebinding pgopclusterbinding 
fi

$CO_CMD --namespace=$CO_NAMESPACE get clusterrolebinding pgopclusterbindingcrd 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete clusterrolebinding pgopclusterbindingcrd
fi

$CO_CMD --namespace=$CO_NAMESPACE get role pgo-role 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete role pgo-role
fi

$CO_CMD --namespace=$CO_NAMESPACE get rolebinding pgo-role-binding 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete rolebinding pgo-role-binding
fi

$CO_CMD --namespace=$CO_NAMESPACE get role pgo-backrest-role 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete role pgo-backrest-role
fi

$CO_CMD --namespace=$CO_NAMESPACE get rolebinding pgo-backrest-role-binding 
if [ $? -eq 0 ]
then
	$CO_CMD --namespace=$CO_NAMESPACE delete rolebinding pgo-backrest-role-binding
fi

sleep 5

