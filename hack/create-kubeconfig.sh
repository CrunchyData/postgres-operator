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

kube_dir="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )/.kube"

context_name=postgres-operator
operator_namespace=postgres-operator
operator_username=postgres-operator

# create hack/.kube dir and copy the kubeconfig currently configured
mkdir -p "${kube_dir}"
[[ -z "${KUBECONFIG}" ]] && KUBECONFIG="${HOME}/.kube/config"
if [[ ! -f "${KUBECONFIG}" ]]; then
    echo "unable to find kubeconfig"
    exit 1
fi
echo "using KUBECONFIG=${KUBECONFIG} as base kubeconfig"

sa_kubeconfig="${kube_dir}/${context_name}"
kubectl config view --minify --raw > "${sa_kubeconfig}"
echo "postgres-operator KUBECONFIG=${sa_kubeconfig}"

# create a new postgres-operator context with a postgres-operator user and the same cluster as the 
# current context as obtained from the KUBECONFIG above
current_cluster=$(kubectl config --kubeconfig "${sa_kubeconfig}" view -o jsonpath="{.contexts[0].context.cluster}")
kubectl config --kubeconfig "${sa_kubeconfig}" set-context "${context_name}" \
    --cluster="${current_cluster}" --user="${operator_username}"

# now grab the token for the postgres-operator SA and use it as the credentials for the postgres-operator user
# postgres-operator users
token=$(kubectl describe secret postgres-operator-token -n "${operator_namespace}" | awk '/^token:/ { print $2 }')
kubectl config --kubeconfig "${sa_kubeconfig}" set-credentials "${operator_username}" --token="${token}"

# set the postgres-operator context as the current context
kubectl config --kubeconfig "${sa_kubeconfig}" use-context "${context_name}"
