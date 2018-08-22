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

if [ "$CO_UI" = "true" ]; then
$CO_CMD $NS delete configmap pgo-ui-conf 
fi

$CO_CMD $NS delete configmap operator-conf 
$CO_CMD $NS delete secret apiserver-conf-secret

$CO_CMD $NS delete service postgres-operator

$CO_CMD $NS delete deployment postgres-operator

sleep 5

