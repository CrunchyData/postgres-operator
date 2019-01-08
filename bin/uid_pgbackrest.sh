#!/bin/bash

if ! whoami &> /dev/null
then
    if [[ -w /etc/passwd ]]
    then
        sed  "/pgbackrest:x:2000:/d" /etc/passwd >> /tmp/uid.tmp
        cp /tmp/uid.tmp /etc/passwd
        rm -f /tmp/uid.tmp
        echo "${USER_NAME:-pgbackrest}:x:$(id -u):0:${USER_NAME:-pgbackrest} user:${HOME}:/bin/bash" >> /etc/passwd
    fi

    if [[ -w /etc/group ]]
    then
        sed  "/pgbackrest:x:2000/d" /etc/group >> /tmp/gid.tmp
        cp /tmp/gid.tmp /etc/group
        rm -f /tmp/gid.tmp
        echo "nfsnobody:x:65534:" >> /etc/group
        echo "pgbackrest:x:$(id -g):pgbackrest" >> /etc/group
    fi
fi
exec "$@"
