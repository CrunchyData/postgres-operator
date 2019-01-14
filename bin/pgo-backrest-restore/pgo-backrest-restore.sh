#!/bin/bash -x

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

