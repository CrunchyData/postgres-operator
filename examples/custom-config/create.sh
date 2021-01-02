#!/bin/bash

# Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

RED="\033[0;31m"
GREEN="\033[0;32m"
RESET="\033[0m"

function echo_err() {
    echo -e "${RED?}$(date) ERROR: ${1?}${RESET?}"
}

function echo_info() {
    echo -e "${GREEN?}$(date) INFO: ${1?}${RESET?}"
}


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# PGO_CMD should either be "kubectl" or "oc" -- defaulting to kubectl
PGO_CMD=${PGO_CMD:-kubectl}

#Error is PGO_NAMESPACE not set
if [[ -z ${PGO_NAMESPACE} ]]
then
        echo_err "PGO_NAMESPACE is not set."
fi

# If both PGO_CMD and PGO_NAMESPACE are set, config map can be created.
if [[ ! -z ${PGO_CMD} ]] && [[ ! -z ${PGO_NAMESPACE} ]]
then

	echo_info "PGO_NAMESPACE=${PGO_NAMESPACE}"

	$PGO_CMD delete configmap pgo-custom-pg-config -n ${PGO_NAMESPACE}

	$PGO_CMD create configmap pgo-custom-pg-config --from-file=$DIR -n ${PGO_NAMESPACE}
fi
