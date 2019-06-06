#!/bin/bash 
# Copyright 2019 Crunchy Data Solutions, Inc.
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

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete pgreplicas --all
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete pgbackups --all
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete pgclusters --all
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete pgpolicies --all
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete pgtasks --all

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete crd \
	pgbackups.crunchydata.com \
	pgreplicas.crunchydata.com \
	pgclusters.crunchydata.com \
	pgpolicies.crunchydata.com \
	pgtasks.crunchydata.com 

$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete jobs --selector=pgrmdata=true
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete jobs --selector=pgbackup=true
$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete jobs --selector=pgo-load=true
