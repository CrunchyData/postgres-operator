#!/bin/bash -x

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

env

/usr/local/bin/pgo-rmdata -pg-cluster=$PG_CLUSTER \
	-replica-name=$REPLICA_NAME \
	-namespace=$NAMESPACE \
	-remove-data=$REMOVE_DATA \
	-remove-backup=$REMOVE_BACKUP \
	-is-backup=$IS_BACKUP \
	-is-replica=$IS_REPLICA \
	-pgha-scope=$PGHA_SCOPE
