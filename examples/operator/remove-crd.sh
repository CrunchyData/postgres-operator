#!/bin/bash -x
# Copyright 2016 Crunchy Data Solutions, Inc.
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

source $DIR/setup.sh

$CO_CMD delete pgbackups --all
$CO_CMD delete pgclones --all
$CO_CMD delete pgclusters --all
$CO_CMD delete pgpolicies --all
$CO_CMD delete pgpolicylogs --all
$CO_CMD delete pgupgrades --all

$CO_CMD delete crd \
	examples.cr.client-go.k8s.io \
	pgbackups.cr.client-go.k8s.io \
	pgclusters.cr.client-go.k8s.io \
	pgpolicies.cr.client-go.k8s.io \
	pgpolicylogs.cr.client-go.k8s.io \
	pgupgrades.cr.client-go.k8s.io

