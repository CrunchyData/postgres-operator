#!/bin/bash 
# Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

$CO_CMD --namespace=$CO_NAMESPACE delete pgreplicas --all
$CO_CMD --namespace=$CO_NAMESPACE delete pgbackups --all
$CO_CMD --namespace=$CO_NAMESPACE delete pgclusters --all
$CO_CMD --namespace=$CO_NAMESPACE delete pgpolicies --all
$CO_CMD --namespace=$CO_NAMESPACE delete pgupgrades --all
$CO_CMD --namespace=$CO_NAMESPACE delete pgtasks --all

$CO_CMD --namespace=$CO_NAMESPACE delete crd \
	pgbackups.cr.client-go.k8s.io \
	pgreplicas.cr.client-go.k8s.io \
	pgclusters.cr.client-go.k8s.io \
	pgpolicies.cr.client-go.k8s.io \
	pgtasks.cr.client-go.k8s.io \
	pgupgrades.cr.client-go.k8s.io \

$CO_CMD --namespace=$CO_NAMESPACE delete jobs --selector=pgrmdata=true
$CO_CMD --namespace=$CO_NAMESPACE delete jobs --selector=pgbackup=true
$CO_CMD --namespace=$CO_NAMESPACE delete jobs --selector=pgo-load=true
