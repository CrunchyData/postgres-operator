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

# A namespace that exists in NAMESPACE env var - see examples/envs.sh
export NS=pgouser1

# remove any existing resources from a previous run
$PGO_CMD delete secret -n $NS \
	fromcrd-postgresuser-secret \
	fromcrd-primaryuser-secret \
	fromcrd-testuser-secret \
	fromcrd-backrest-repo-config > /dev/null
$PGO_CMD delete pgcluster fromcrd -n $NS
$PGO_CMD delete pvc fromcrd fromcrd-pgbr-repo  -n $NS

# generate a SSH public/private keypair for use by pgBackRest
ssh-keygen -N '' -f $DIR/fromcrd-key

# base64 encoded the keys for the generation of the Kube secret, and place
# them into variables temporarily
export PUBLIC_KEY=$(base64 -i $DIR/fromcrd-key.pub)
export PRIVATE_KEY=$(base64 -i $DIR/fromcrd-key)

# copy the backrest-repo-config example file, and substitute in the newly
# created keys
cp $DIR/backrest-repo-config.example.yml $DIR/backrest-repo-config.yaml

# OS X / macOS has its own implementation of sed inline command
if [[ "$OSTYPE" == "darwin"* ]]; then
	sed -i '' "s/{{ PUBLIC_KEY }}/$PUBLIC_KEY/g" $DIR/backrest-repo-config.yaml
	sed -i '' "s/{{ PRIVATE_KEY }}/$PRIVATE_KEY/g" $DIR/backrest-repo-config.yaml
else
	sed -i "s/{{ PUBLIC_KEY }}/$PUBLIC_KEY/g" $DIR/backrest-repo-config.yaml
	sed -i "s/{{ PRIVATE_KEY }}/$PRIVATE_KEY/g" $DIR/backrest-repo-config.yaml
fi

# unset the *_KEY environmental variables
unset PUBLIC_KEY
unset PRIVATE_KEY

# create the required postgres credentials for the fromcrd cluster
$PGO_CMD -n $NS create -f $DIR/postgres-secret.yaml
$PGO_CMD -n $NS create -f $DIR/primaryuser-secret.yaml
$PGO_CMD -n $NS create -f $DIR/testuser-secret.yaml
$PGO_CMD -n $NS create -f $DIR/backrest-repo-config.yaml

# create the pgcluster CRD for the fromcrd cluster
$PGO_CMD -n $NS create -f $DIR/fromcrd.json
