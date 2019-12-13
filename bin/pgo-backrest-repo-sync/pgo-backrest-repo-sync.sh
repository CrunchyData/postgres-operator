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


function trap_sigterm() {
	echo "Signal trap triggered, beginning shutdown.."
	killall sshd
}

trap 'trap_sigterm' SIGINT SIGTERM

SSHD_CONFIG=/sshd

echo "PGBACKREST env vars are set to:"
set | grep PGBACKREST

echo "SSHD_CONFIG is.."
ls $SSHD_CONFIG

mkdir ~/.ssh/
cp $SSHD_CONFIG/config ~/.ssh/
cp $SSHD_CONFIG/id_rsa /tmp
chmod 400 /tmp/id_rsa ~/.ssh/config

# start sshd which is used by pgbackrest for remote connections
/usr/sbin/sshd -D -f $SSHD_CONFIG/sshd_config   &

echo "sleep 5 secs to let sshd come up before running rsync command"
sleep 5

echo "rsync pgbackrest from $PGBACKREST_REPO1_HOST:$PGBACKREST_REPO1_PATH/ to $NEW_PGBACKREST_REPO"
# note, the "/" after the repo path is important, as we do not want to sync
# the top level directory
rsync -a --progress ${PGBACKREST_REPO1_HOST}:${PGBACKREST_REPO1_PATH}/ $NEW_PGBACKREST_REPO
