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

$CO_CMD delete svc --selector=pg-cluster
$CO_CMD delete secret --selector=pg-database
$CO_CMD delete pvc --selector=pgremove
$CO_CMD delete pgbackups   --all
$CO_CMD delete pgclusters --all  
$CO_CMD delete pgpolicies --all
$CO_CMD delete pgreplicas --all
$CO_CMD delete pgtasks  --all
$CO_CMD delete pgupgrades --all
