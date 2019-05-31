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

# parameter #1 passed into this script should be a namespace
#   into which the rbac role and rolebinding will be created
# parameter #2 passed into this script should be the namespace
#   into which the operator is deployed


echo ""
echo "creating target rbac role and rolebinding in namespace " $1
echo "operator is assumed to be deployed into " $2

export TARGET_NAMESPACE=$1
export PGO_OPERATOR_NAMESPACE=$2
expenv -f $DIR/rbac.yaml | $PGO_CMD --namespace=$1 delete -f -

expenv -f $DIR/rbac.yaml | $PGO_CMD --namespace=$1 create -f -

