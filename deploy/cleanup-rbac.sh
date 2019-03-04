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


$CO_CMD --namespace=$CO_NAMESPACE delete serviceaccount postgres-operator

$CO_CMD --namespace=$CO_NAMESPACE delete clusterrole pgopclusterrole pgopclusterrolecrd
$CO_CMD --namespace=$CO_NAMESPACE delete clusterrolebinding pgopclusterbinding pgopclusterbindingcrd

$CO_CMD --namespace=$CO_NAMESPACE delete role pgo-role
$CO_CMD --namespace=$CO_NAMESPACE delete rolebinding pgo-role-binding

$CO_CMD --namespace=$CO_NAMESPACE delete serviceaccount pgo-backrest
$CO_CMD --namespace=$CO_NAMESPACE delete role pgo-backrest-role
$CO_CMD --namespace=$CO_NAMESPACE delete rolebinding pgo-backrest-role-binding

sleep 5

