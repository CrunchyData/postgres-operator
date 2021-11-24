#!/usr/bin/env bash

# Copyright 2021 Crunchy Data Solutions, Inc.
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

set -eu

while [[ $# -gt 0 ]]; do
  key="$1"

  case $key in
    --path)
      OUTPUT_PATH="$2"
      shift # past argument
      shift # past value
      ;;
    --version)
      PGO_VERSION="$2"
      shift # past argument
      shift # past value
      ;;
    *)
      shift # past argument
      ;;
  esac
done


cat <<EOF > "${OUTPUT_PATH}/labels.yaml"
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: postgresclusters.postgres-operator.crunchydata.com
  labels:
    app.kubernetes.io/name: pgo
    app.kubernetes.io/version: ${PGO_VERSION}
EOF
