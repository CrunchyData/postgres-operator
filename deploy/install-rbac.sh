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

# see if CRDs need to be created
$CO_CMD get crd pgclusters.cr.client-go.k8s.io
if [ $? -eq 1 ]; then
	$CO_CMD create -f $DIR/crd.yaml
fi

# create the cluster roles one time for the entire Kube cluster
expenv -f $DIR/cluster-roles.yaml | $CO_CMD create -f -

# create the Operator service accounts
expenv -f $DIR/service-accounts.yaml | $CO_CMD --namespace=$CO_NAMESPACE create -f -

# create the cluster role bindings to the Operator service accounts
# postgres-operator and pgo-backrest, here we are assuming a single
# Operator in the CO_NAMESPACE env variable
expenv -f $DIR/cluster-role-bindings.yaml | $CO_CMD --namespace=$CO_NAMESPACE create -f -

# create the role, role-binding and add to the service account
# these are created within the namespace the Operator is running
expenv -f $DIR/rbac.yaml | $CO_CMD create --namespace=$CO_NAMESPACE -f -

# create the keys used for pgo API
source $DIR/gen-api-keys.sh

# create the sshd keys for pgbackrest repo functionality
source $DIR/gen-sshd-keys.sh

# create a pgo-backrest-repo-config Secret into each namespace the
# Operator will be watching

IFS=', ' read -r -a array <<< "$NAMESPACE"

echo ""
echo "create pgo-backrest-repo-config Secret into each namespace the Operator is watching..."
for ns in "${array[@]}"
do
        $CO_CMD get secret pgo-backrest-repo-config --namespace=$ns > /dev/null 2> /dev/null
        if [ $? -eq 0 ]
        then
                $CO_CMD delete secret  pgo-backrest-repo-config --namespace=$ns > /dev/null 2> /dev/null
        fi
        $DIR/create-pgo-backrest-ssh-secret.sh $ns
done

