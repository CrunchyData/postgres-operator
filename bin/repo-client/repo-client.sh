#!/bin/bash

function trap_sigterm() {
echo "Signal trap triggered, beginning shutdown.."

if [ -f $PGDATA/postmaster.pid ]; then
	PI=$(ps ax | grep sshd | cut -f4 -d' ')
	echo "sending SIGINT to " $PI
	kill -SIGINT $PI
fi
}

trap 'trap_sigterm' SIGINT SIGTERM

CONFIG=/sshd
REPO=/backrestrepo

mkdir ~/.ssh
cp $CONFIG/config ~/.ssh/
cp $CONFIG/id_rsa /tmp
chmod 400 /tmp/id_rsa ~/.ssh/config


echo "CONFIG is.."
ls $CONFIG
echo "REPO is ..."
ls $REPO

# start sshd which is used by pgbackrest for remote connections
/usr/sbin/sshd -D -f $CONFIG/sshd_config

