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

CONFIG=/sshd

echo "PGBACKREST env vars are set to:"
set | grep PGBACKREST

echo "CONFIG is.."
ls $CONFIG
echo "REPO is ..."
ls $REPO

mkdir ~/.ssh/
cp $CONFIG/config ~/.ssh/
cp $CONFIG/id_rsa /tmp
chmod 400 /tmp/id_rsa ~/.ssh/config

# start sshd which is used by pgbackrest for remote connections
/usr/sbin/sshd -D -f $CONFIG/sshd_config   &

# create the directory the restore will go into
mkdir $PGBACKREST_DB_PATH

echo "sleep 5 secs to let sshd come up before running pgbackrest command"
sleep 5

if [ "$PITR_TARGET" = "" ]
then
	echo "PITR_TARGET is  empty"
	pgbackrest restore $COMMAND_OPTS
else
	echo PITR_TARGET is not empty [$PITR_TARGET]
	pgbackrest restore $COMMAND_OPTS "--target=$PITR_TARGET"
fi


#/opt/cpm/bin/pgo-backrest-restore

