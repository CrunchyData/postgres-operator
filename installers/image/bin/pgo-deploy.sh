#!/bin/bash

#  Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
#  Licensed under the Apache License, Version 2.0 (the "License");
#  you may not use this file except in compliance with the License.
#  You may obtain a copy of the License at
#       http://www.apache.org/licenses/LICENSE-2.0
#  Unless required by applicable law or agreed to in writing, software
#  distributed under the License is distributed on an "AS IS" BASIS,
#  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
#  See the License for the specific language governing permissions and
#  limitations under the License.

export DEPLOY_ACTION=${DEPLOY_ACTION:-install}

/usr/bin/env ansible-playbook \
    -i "/ansible/${PLAYBOOK:-postgres-operator}/inventory.yaml" \
    --extra-vars "kubernetes_in_cluster=true" \
    --extra-vars "config_path=/conf/values.yaml" \
    --tags=$DEPLOY_ACTION \
    "/ansible/${PLAYBOOK:-postgres-operator}/main.yml"
