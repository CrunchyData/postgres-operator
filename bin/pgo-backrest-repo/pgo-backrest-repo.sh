#!/bin/bash

CONFIG=/config
REPO=/backrestrepo

echo "CONFIG is.."
ls $CONFIG
echo "REPO is ..."
ls $REPO

# start sshd which is used by pgbackrest for remote connections
/usr/sbin/sshd -c $CONFIG/server.crt -h $CONFIG/server.key

while [ 1 -lt 4 ]
do
	sleep 10
	echo "sleep.."
done
