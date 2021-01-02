#!/bin/bash

# Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

##########
# SETUP #
#########

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
# PGO_CMD should either be "kubectl" or "oc" -- defaulting to kubectl
PGO_CMD=${PGO_CMD:-kubectl}

# A namespace that exists in NAMESPACE env var - see examples/envs.sh
export NS=pgouser1

###########
# CLEANUP #
##########

# remove any existing resources from a previous run
$PGO_CMD delete secret -n $NS \
	fromcrd-postgres-secret \
	fromcrd-primaryuser-secret \
	fromcrd-testuser-secret \
	fromcrd-backrest-repo-config > /dev/null
$PGO_CMD delete pgcluster fromcrd -n $NS
$PGO_CMD delete pvc fromcrd fromcrd-pgbr-repo  -n $NS
# remove the public/private keypair from the previous run
rm $DIR/fromcrd-key $DIR/fromcrd-key.pub

###############
# EXAMPLE RUN #
###############

# create the required postgres credentials for the fromcrd cluster
$PGO_CMD -n $NS create -f $DIR/postgres-secret.yaml
$PGO_CMD -n $NS create -f $DIR/testuser-secret.yaml

# create the pgcluster CRD for the fromcrd cluster
$PGO_CMD -n $NS create -f $DIR/fromcrd.json
