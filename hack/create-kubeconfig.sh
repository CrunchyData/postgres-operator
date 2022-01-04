#!/usr/bin/env bash

# Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

directory=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )

declare -r namespace="$1" account="$2"
declare -r directory="${directory}/.kube"

[[ -z "${KUBECONFIG:-}" ]] && KUBECONFIG="${HOME}/.kube/config"
if [[ ! -f "${KUBECONFIG}" ]]; then
    echo "unable to find kubeconfig"
    exit 1
fi
echo "using KUBECONFIG=${KUBECONFIG} as base for ${namespace}/${account}"

# copy the current KUBECONFIG
kubeconfig="${directory}/${namespace}/${account}"
mkdir -p "${directory}/${namespace}"
kubectl config view --minify --raw > "${kubeconfig}"

# grab the service account token
token=$(kubectl get secret -n "${namespace}" -o go-template='
{{- range .items }}
	{{- if and (eq (or .type "") "kubernetes.io/service-account-token") .metadata.annotations }}
	{{- if (eq (or (index .metadata.annotations "kubernetes.io/service-account.name") "") "'"${account}"'") }}
	{{- if (ne (or (index .metadata.annotations "kubernetes.io/created-by") "") "openshift.io/create-dockercfg-secrets") }}
	{{- .data.token | base64decode }}
	{{- end }}
	{{- end }}
	{{- end }}
{{- end }}')
kubectl config --kubeconfig="${kubeconfig}" set-credentials "${account}" --token="${token}"

# remove any namespace setting, replace the username, and minify once more
kubectl config --kubeconfig="${kubeconfig}" set-context --current --namespace= --user="${account}"
minimal=$(kubectl config --kubeconfig="${kubeconfig}" view --minify --raw)
cat <<< "${minimal}" > "${kubeconfig}"
