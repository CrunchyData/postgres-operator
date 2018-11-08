#!/bin/sh
/opt/cpm/bin/pgo-backrest

echo $UID "is the UID in the script"

chown -R $UID:$UID $PGBACKREST_DB_PATH

chmod -R o+rx $PGBACKREST_DB_PATH
