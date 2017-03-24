#!/bin/bash
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

source $COROOT/examples/envvars.sh

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

$DIR/cleanup.sh

sudo mkdir /data
sudo chmod 777 /data

# copy all the operator templates to the PVC location
sudo cp -r $COROOT/conf/postgres-operator /data

kubectl create -f $DIR/crunchy-pv.json

sleep 3
kubectl create -f $DIR/crunchy-pvc.json

sleep 3
kubectl create -f $DIR/deployment.json
