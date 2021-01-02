#!/bin/bash

# Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

# Enforce required environment variables
test="${PGO_CMD:?Need to set PGO_CMD env variable}"
test="${PGOROOT:?Need to set PGOROOT env variable}"
test="${PGO_OPERATOR_NAMESPACE:?Need to set PGO_OPERATOR_NAMESPACE env variable}"
test="${PGO_INSTALLATION_NAME:?Need to set PGO_INSTALLATION_NAME env variable}"
test="${PGO_CONF_DIR:?Need to set PGO_CONF_DIR env variable}"

if [[ -z "$1" ]]; then
	echo "usage:  add-targeted-namespace-reconcile-rbac.sh mynewnamespace"
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

$PGO_CMD -n $1 delete --ignore-not-found rolebinding pgo-target-role-binding pgo-local-ns
$PGO_CMD -n $1 delete --ignore-not-found role pgo-target-role pgo-local-ns

cat $PGO_CONF_DIR/pgo-configs/pgo-target-role.json | sed 's/{{.TargetNamespace}}/'"$1"'/' | $PGO_CMD -n $1 create -f -
cat $DIR/local-namespace-rbac.yaml | envsubst | $PGO_CMD -n $1 create -f -
