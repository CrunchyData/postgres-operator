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

LOC=$COROOT/conf/pgo-backrest-repo

ssh-keygen -f $LOC/id_rsa -t rsa -N ''
ssh-keygen -t rsa -f $LOC/ssh_host_rsa_key -N ''
ssh-keygen -t ecdsa -f $LOC/ssh_host_ecdsa_key -N ''
ssh-keygen -t ed25519 -f $LOC/ssh_host_ed25519_key -N ''
cp $LOC/id_rsa.pub $LOC/authorized_keys
