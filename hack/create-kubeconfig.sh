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

# Grab the service account token. If one has not already been generated,
# create a secret to do so. See the LegacyServiceAccountTokenNoAutoGeneration
# feature gate.
for i in 1 2; do
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

	[[ -n "${token}" ]] && break

	kubectl apply -n "${namespace}" --server-side --filename=- <<< "
apiVersion: v1
kind: Secret
type: kubernetes.io/service-account-token
metadata: {
	name: ${account}-token,
	annotations: { kubernetes.io/service-account.name: ${account} }
}"
done
kubectl config --kubeconfig="${kubeconfig}" set-credentials "${account}" --token="${token}"

# remove any namespace setting, replace the username, and minify once more
kubectl config --kubeconfig="${kubeconfig}" set-context --current --namespace= --user="${account}"
minimal=$(kubectl config --kubeconfig="${kubeconfig}" view --minify --raw)
cat <<< "${minimal}" > "${kubeconfig}"
