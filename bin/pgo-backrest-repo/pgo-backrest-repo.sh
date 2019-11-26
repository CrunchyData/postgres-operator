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


function trap_sigterm() {
	echo "Signal trap triggered, beginning shutdown.." 
	killall sshd
}

trap 'trap_sigterm' SIGINT SIGTERM

CONFIG=/sshd
REPO=/backrestrepo

echo "PGBACKREST env vars are set to:"
set | grep PGBACKREST

echo "CONFIG is.."
ls $CONFIG
echo "REPO is ..."
ls $REPO

if [ ! -d $PGBACKREST_REPO_PATH ]; then
	echo "creating " $PGBACKREST_REPO_PATH
	mkdir -p $PGBACKREST_REPO_PATH
fi

mkdir ~/.ssh/
cp $CONFIG/config ~/.ssh/
#cp $CONFIG/authorized_keys ~/.ssh/
cp $CONFIG/id_rsa /tmp
chmod 400 /tmp/id_rsa ~/.ssh/config

# start sshd which is used by pgbackrest for remote connections
/usr/sbin/sshd -D -f $CONFIG/sshd_config   &

wait

