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


if [ "$CO_CMD" = "kubectl" ]; then
	NS="--namespace=$CO_NAMESPACE"
fi

$CO_CMD $NS delete serviceaccount postgres-operator

$CO_CMD $NS delete clusterrole pgopclusterrole pgopclusterrolecrd
$CO_CMD $NS delete clusterrolebinding pgopclusterbinding pgopclusterbindingcrd

$CO_CMD $NS delete role pgo-role
$CO_CMD $NS delete rolebinding pgo-role-binding

sleep 5

