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


if [[ -z "$1" ]]; then
	echo "usage:  add-targeted-namespace.sh mynewnamespace"
	exit
fi


# create the namespace if necessary
$PGO_CMD get ns $1  > /dev/null
if [ $? -eq 0 ]; then
	echo "namespace" $1 "already exists"
else
	echo "namespace" $1 "is new"
	TARGET_NAMESPACE=$1 expenv -f $DIR/target-namespace.yaml | $PGO_CMD create -f -
fi

# set the labels so that this namespace is owned by this installation
$PGO_CMD label namespace/$1 pgo-created-by=add-script
$PGO_CMD label namespace/$1 vendor=crunchydata
$PGO_CMD label namespace/$1 pgo-installation-name=$PGO_INSTALLATION_NAME

# create RBAC
$PGO_CMD -n $1 delete sa pgo-backrest 
$PGO_CMD -n $1 delete sa pgo-target
$PGO_CMD -n $1 delete role pgo-target-role pgo-backrest-role
$PGO_CMD -n $1 delete rolebinding pgo-target-role-binding pgo-backrest-role-binding

cat $PGOROOT/conf/postgres-operator/pgo-backrest-sa.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGOROOT/conf/postgres-operator/pgo-target-sa.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGOROOT/conf/postgres-operator/pgo-target-role.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGOROOT/conf/postgres-operator/pgo-target-role-binding.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | sed 's/{{.OperatorNamespace}}/'"$PGO_OPERATOR_NAMESPACE"'/' | $PGO_CMD -n $1 create -f -
cat $PGOROOT/conf/postgres-operator/pgo-backrest-role.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGOROOT/conf/postgres-operator/pgo-backrest-role-binding.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
