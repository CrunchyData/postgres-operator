#!/usr/bin/env bash

# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
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

directory=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
clusters_dir="${directory}/../build/crd/postgresclusters"
upgrades_dir="${directory}/../build/crd/pgupgrades"

# Generate a Kustomize patch file for removing any TODOs we inherit from the Kubernetes API.
# Right now there is one TODO in our CRD. This script focuses on removing the specific TODO
# anywhere they are found in the CRD.

# The TODO comes from the following:
# https://github.com/kubernetes/api/blob/25b7aa9e86de7bba38c35cbe56701d2c1ff207e9/core/v1/types.go#L5609
# Additionally, the hope is that this step can be removed once the following issue is addressed
# in the kubebuilder controller-tools project:
# https://github.com/kubernetes-sigs/controller-tools/issues/649

echo "Generating Kustomize patch file for removing Kube API TODOs"

# Get the description of the "name" field with the TODO from any place it is used in the CRD and
# store it in a variable. Then, create another variable with the TODO stripped out.
name_desc_with_todo=$(
  python3 -m yq -r \
    .spec.versions[0].schema.openAPIV3Schema.properties.spec.properties.customTLSSecret.properties.name.description \
    "${clusters_dir}/generated/postgres-operator.crunchydata.com_postgresclusters.yaml"
)
name_desc_without_todo=$(sed 's/ TODO.*//g' <<< "${name_desc_with_todo}")

# Generate a JSON patch file to update the "name" description for all applicable paths in the CRD.
python3 -m yq -y --arg old "${name_desc_with_todo}" --arg new "${name_desc_without_todo}" '
	[{ op: "add", path: "/work", value: $new }] +
	[paths(select(. == $old)) | { op: "copy", from: "/work", path: "/\(map(tostring) | join("/"))" }] +
	[{ op: "remove", path: "/work" }]
' \
	"${clusters_dir}/generated/postgres-operator.crunchydata.com_postgresclusters.yaml" > "${clusters_dir}/todos.yaml"

python3 -m yq -y --arg old "${name_desc_with_todo}" --arg new "${name_desc_without_todo}" '
	[{ op: "add", path: "/work", value: $new }] +
	[paths(select(. == $old)) | { op: "copy", from: "/work", path: "/\(map(tostring) | join("/"))" }] +
	[{ op: "remove", path: "/work" }]
' \
	"${upgrades_dir}/generated/postgres-operator.crunchydata.com_pgupgrades.yaml" > "${upgrades_dir}/todos.yaml"
