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

test="${PGOROOT:?Need to set PGOROOT env variable}"

# sync a master config file with kubectl and helm installers
sync_config() {

    KUBECTL_SPEC_PREFIX=$1
    INSTALLER_ROOT=$2
    MASTER_CONFIG=$3

    yq write --inplace --doc 2 "$INSTALLER_ROOT/kubectl/$KUBECTL_SPEC_PREFIX.yml" 'data"values.yaml"' -- "$(cat $MASTER_CONFIG)"
    yq write --inplace --doc 2 "$INSTALLER_ROOT/kubectl/$KUBECTL_SPEC_PREFIX-ocp311.yml" 'data"values.yaml"' -- "$(cat $MASTER_CONFIG)"

    cat "$INSTALLER_ROOT/helm/helm_template.yaml" "$MASTER_CONFIG" > "$INSTALLER_ROOT/helm/values.yaml"
}

# sync operator configuration
sync_config "postgres-operator" "$PGOROOT/installers" "$PGOROOT/installers/ansible/values.yaml"

# sync metrics configuration
sync_config "postgres-operator-metrics" "$PGOROOT/installers/metrics" "$PGOROOT/installers/metrics/ansible/values.yaml"

echo "Configuration sync complete"
