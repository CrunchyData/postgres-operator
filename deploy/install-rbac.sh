#!/bin/bash
# Copyright 2018 Crunchy Data Solutions, Inc.
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

$DIR/cleanup-rbac.sh

# see if CRDs need to be created
$CO_CMD get crd pgclusters.cr.client-go.k8s.io
if [ $? -eq 1 ]; then
	$CO_CMD create -f $DIR/crd.yaml
fi

if [ "$CO_CMD" = "kubectl" ]; then
	NS="--namespace=$CO_NAMESPACE"
fi

# create the cluster role and add to the service account
expenv -f $DIR/cluster-rbac.yaml | $CO_CMD create -f -

# create the service account, role, role-binding and add to the service account
expenv -f $DIR/rbac.yaml | $CO_CMD create -f -

