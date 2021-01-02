#!/bin/bash

# Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

# the name of the service account utilized by the PG pods
PG_SA="pgo-pg"

# Enforce required environment variables
test="${PGO_CMD:?Need to set PGO_CMD env variable}"
test="${PGOROOT:?Need to set PGOROOT env variable}"
test="${PGO_OPERATOR_NAMESPACE:?Need to set PGO_OPERATOR_NAMESPACE env variable}"
test="${PGO_INSTALLATION_NAME:?Need to set PGO_INSTALLATION_NAME env variable}"
test="${PGO_CONF_DIR:?Need to set PGO_CONF_DIR env variable}"

if [[ -z "$1" ]]; then
	echo "usage:  add-targeted-namespace.sh mynewnamespace"
	exit
fi

# create the namespace if necessary
$PGO_CMD get ns $1  > /dev/null
if [ $? -eq 0 ]; then
	echo "namespace" $1 "already exists, adding labels"
	# set the labels so that existing namespace is owned by this installation
	$PGO_CMD label namespace/$1 pgo-created-by=add-script
	$PGO_CMD label namespace/$1 vendor=crunchydata
	$PGO_CMD label namespace/$1 pgo-installation-name=$PGO_INSTALLATION_NAME
else
	echo "namespace" $1 "is new"
	cat $DIR/target-namespace.yaml | sed -e 's/$TARGET_NAMESPACE/'"$1"'/' -e 's/$PGO_INSTALLATION_NAME/'"$PGO_INSTALLATION_NAME"'/' | $PGO_CMD create -f -
fi

# determine if an existing pod is using the 'pgo-pg' service account.  if so, do not delete
# and recreate the SA or its associated role and role binding.  this is to avoid any undesired
# behavior with existing PG clusters that are actively utilizing the SA.
$PGO_CMD -n $1 get pods -o yaml | grep "serviceAccount: ${PG_SA}"  > /dev/null
if [ $? -ne 0 ]; then
	$PGO_CMD -n $1 delete --ignore-not-found sa pgo-pg
	$PGO_CMD -n $1 delete --ignore-not-found role pgo-pg-role
	$PGO_CMD -n $1 delete --ignore-not-found rolebinding pgo-pg-role-binding

	cat $PGO_CONF_DIR/pgo-configs/pgo-pg-sa.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
	cat $PGO_CONF_DIR/pgo-configs/pgo-pg-role.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
	cat $PGO_CONF_DIR/pgo-configs/pgo-pg-role-binding.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
else
	echo "Running pods found using SA '${PG_SA}' in namespace $1, will not recreate"
fi

# create RBAC
$PGO_CMD -n $1 delete --ignore-not-found sa pgo-backrest pgo-default pgo-target
$PGO_CMD -n $1 delete --ignore-not-found role pgo-backrest-role pgo-target-role
$PGO_CMD -n $1 delete --ignore-not-found rolebinding pgo-backrest-role-binding pgo-target-role-binding

cat $PGO_CONF_DIR/pgo-configs/pgo-default-sa.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGO_CONF_DIR/pgo-configs/pgo-target-sa.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGO_CONF_DIR/pgo-configs/pgo-target-role.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGO_CONF_DIR/pgo-configs/pgo-target-role-binding.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | sed 's/{{.OperatorNamespace}}/'"$PGO_OPERATOR_NAMESPACE"'/' | $PGO_CMD -n $1 create -f -
cat $PGO_CONF_DIR/pgo-configs/pgo-backrest-sa.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGO_CONF_DIR/pgo-configs/pgo-backrest-role.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $PGO_CONF_DIR/pgo-configs/pgo-backrest-role-binding.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -

if [ -r "$PGO_IMAGE_PULL_SECRET_MANIFEST" ]; then
	$PGO_CMD -n $1 create -f "$PGO_IMAGE_PULL_SECRET_MANIFEST"
fi

if [ -n "$PGO_IMAGE_PULL_SECRET" ]; then
	patch='{"imagePullSecrets": [{ "name": "'"$PGO_IMAGE_PULL_SECRET"'" }]}'

	$PGO_CMD -n $1 patch --type=strategic --patch="$patch" serviceaccount/pgo-backrest
	$PGO_CMD -n $1 patch --type=strategic --patch="$patch" serviceaccount/pgo-default
	$PGO_CMD -n $1 patch --type=strategic --patch="$patch" serviceaccount/pgo-pg
	$PGO_CMD -n $1 patch --type=strategic --patch="$patch" serviceaccount/pgo-target
fi
