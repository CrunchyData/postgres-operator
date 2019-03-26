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

# remove any existing resources from a previous run
$PGO_CMD $NS delete secret \
	fromcrd-postgresuser-secret \
	fromcrd-primaryuser-secret \
	fromcrd-testuser-secret	> /dev/null
$PGO_CMD $NS delete pgcluster fromcrd
$PGO_CMD $NS delete pvc fromcrd

# create the required postgres credentials for the fromcrd cluster
$PGO_CMD $NS create -f $DIR/postgres-secret.yaml
$PGO_CMD $NS create -f $DIR/primaryuser-secret.yaml
$PGO_CMD $NS create -f $DIR/testuser-secret.yaml

# create the pgcluster CRD for the fromcrd cluster
$PGO_CMD $NS create -f $DIR/fromcrd.json

