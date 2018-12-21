#!/bin/bash
# Copyright 2017 - 2018 Crunchy Data Solutions, Inc.
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

${DIR}/cleanup.sh

$CO_CMD $NS create secret generic pgo-backrest-repo-config \
--from-file=config=$COROOT/conf/pgo-backrest-repo/config \
--from-file=ssh_host_rsa_key=$COROOT/conf/pgo-backrest-repo/ssh_host_rsa_key \
--from-file=authorized_keys=$COROOT/conf/pgo-backrest-repo/authorized_keys \
--from-file=id_rsa=$COROOT/conf/pgo-backrest-repo/id_rsa \
--from-file=ssh_host_ecdsa_key=$COROOT/conf/pgo-backrest-repo/ssh_host_ecdsa_key \
--from-file=ssh_host_ed25519_key=$COROOT/conf/pgo-backrest-repo/ssh_host_ed25519_key \
--from-file=sshd_config=$COROOT/conf/pgo-backrest-repo/sshd_config

$CO_CMD $NS create -f $DIR/pvc.json
if [[ $? -ne 0 ]]
then
    echo_err "Failed to create storage, exiting.."
    exit 1
fi

expenv -f $DIR/service.json | $CO_CMD create --namespace=$NS -f -
expenv -f $DIR/pgo-backrest-repo.json | $CO_CMD create --namespace=$NS -f -
