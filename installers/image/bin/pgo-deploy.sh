#!/bin/bash

#  Copyright 2020 Crunchy Data Solutions, Inc.
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#       http://www.apache.org/licenses/LICENSE-2.0
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

# Operator Defaults
export BACKREST_PORT=${BACKREST_PORT:-2022}
export CRUNCHY_DEBUG=${CRUNCHY_DEBUG:-false}
export DELETE_METRICS_NAMESPACE=${DELETE_METRICS_NAMESPACE:-false}
export DELETE_OPERATOR_NAMESPACE=${DELETE_OPERATOR_NAMESPACE:-false}
export DELETE_WATCHED_NAMESPACE=${DELETE_WATCHED_NAMESPACE:-false}
export PGO_ADD_OS_CA_STORE=${PGO_ADD_OS_CA_STORE:-false}
export NAMESPACE_MODE=${NAMESPACE_MODE:-dynamic}
export POD_ANTI_AFFINITY=${POD_ANTI_AFFINITY:-preferred}
export PGO_APISERVER_PORT=${PGO_APISERVER_PORT:-8443}
export PGO_APISERVER_URL=${PGO_APISERVER_URL:-https://postgres-operator}
export PGO_CLIENT_CERT_SECRET=${PGO_CLIENT_CERT_SECRET:-pgo.tls}
export PGO_CLIENT_CONTAINER_INSTALL=${PGO_CLIENT_CONTAINER_INSTALL:-false}
export PGO_CLIENT_INSTALL=${PGO_CLIENT_INSTALL:-true}
export PGO_CLUSTER_ADMIN=${PGO_CLUSTER_ADMIN:-false}
export PGO_DISABLE_EVENTING=${PGO_DISABLE_EVENTING:-false}
export PGO_DISABLE_TLS=${PGO_DISABLE_TLS:-false}
export PGO_TLS_NO_VERIFY=${PGO_TLS_NO_VERIFY:-false}
export SERVICE_TYPE=${SERVICE_TYPE:-ClusterIP}

export DEPLOY_ACTION=${DEPLOY_ACTION:-install}

cat /inventory_template | envsubst > /tmp/inventory
/usr/bin/env ansible-playbook -i /tmp/inventory --tags=$DEPLOY_ACTION /ansible/main.yml
