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

$DIR/cleanup-rbac.sh

if [[ $(oc version) =~ "openshift" ]]; then
	# create PGO SCC used with pgclusters (must use 'oc' not 'kubectl')
	oc create -f $DIR/pgo-scc.yaml
fi

# see if CRDs need to be created
$PGO_CMD get crd pgclusters.crunchydata.com > /dev/null
if [ $? -eq 1 ]; then
	$PGO_CMD create -f $DIR/crd.yaml
fi

# create the initial pgo admin credential
$DIR/install-bootstrap-creds.sh

# create the cluster roles one time for the entire Kube cluster
expenv -f $DIR/cluster-roles.yaml | $PGO_CMD create -f -


# create the Operator service accounts
expenv -f $DIR/service-accounts.yaml | $PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create -f -

if [ -r "$PGO_IMAGE_PULL_SECRET_MANIFEST" ]; then
	$PGO_CMD -n $PGO_OPERATOR_NAMESPACE create -f "$PGO_IMAGE_PULL_SECRET_MANIFEST"
fi

if [ -n "$PGO_IMAGE_PULL_SECRET" ]; then
	patch='{"imagePullSecrets": [{ "name": "'"$PGO_IMAGE_PULL_SECRET"'" }]}'

	$PGO_CMD -n $PGO_OPERATOR_NAMESPACE patch --type=strategic --patch="$patch" serviceaccount/postgres-operator
fi

# create the cluster role bindings to the Operator service accounts
# postgres-operator and pgo-backrest, here we are assuming a single
# Operator in the PGO_OPERATOR_NAMESPACE env variable
expenv -f $DIR/cluster-role-bindings.yaml | $PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create -f -

expenv -f $DIR/roles.yaml | $PGO_CMD -n $PGO_OPERATOR_NAMESPACE create -f -
expenv -f $DIR/role-bindings.yaml | $PGO_CMD -n $PGO_OPERATOR_NAMESPACE create -f -

# create the keys used for pgo API
source $DIR/gen-api-keys.sh

# create the sshd keys for pgbackrest repo functionality
source $DIR/gen-sshd-keys.sh

