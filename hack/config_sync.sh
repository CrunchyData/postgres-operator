#!/bin/bash

# Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

INSTALLER_DIR="$PGOROOT/installers"
MASTER_CONFIG="$PGOROOT/installers/ansible/values.yaml"

yq write --inplace --doc 2 "$INSTALLER_DIR/kubectl/postgres-operator.yml" 'data"values.yaml"' -- "$(cat $MASTER_CONFIG)"
yq write --inplace --doc 2 "$INSTALLER_DIR/kubectl/postgres-operator-ocp311.yml" 'data"values.yaml"' -- "$(cat $MASTER_CONFIG)"

cat "$INSTALLER_DIR/helm/postgres-operator/helm_template.yaml" "$MASTER_CONFIG" > "$INSTALLER_DIR/helm/postgres-operator/values.yaml"
