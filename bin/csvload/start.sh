#!/bin/bash -x

# Copyright 2017 Crunchy Data Solutions, Inc.
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

#
# start the csv load job
#
# /pgdata is a volume that gets mapped into this container
# $TABLE_TO_LOAD postgres table to copy data into
# $CSV_FILE_PATH path to csv file
# $DB_HOST host we are connecting to
# $DB_USER pg user we are connecting with
# $DB_PASS pg user password we are connecting with
# $DB_PORT pg port we are connecting to
#

function ose_hack() {
        export USER_ID=$(id -u)
        export GROUP_ID=$(id -g)
        envsubst < /opt/cpm/conf/passwd.template > /tmp/passwd
        export LD_PRELOAD=/usr/lib64/libnss_wrapper.so
        export NSS_WRAPPER_PASSWD=/tmp/passwd
        export NSS_WRAPPER_GROUP=/etc/group
}


ose_hack

echo $CSVFILE_PATH

export PGPASSFILE=/tmp/pgpass

echo "*:*:*:"$DB_USER":"$DB_PASS  >> $PGPASSFILE

chmod 600 $PGPASSFILE

chown $UID:$UID $PGPASSFILE

# cat $PGPASSFILE

#pg_basebackup --label=$BACKUP_LABEL --xlog --pgdata $CSVFILE_PATH --host=$DB_HOST --port=$DB_PORT -U $DB_USER

#chown -R $UID:$UID $CSVFILE_PATH
# 
# open up permissions for the OSE Dedicated random UID scenario
#
#chmod -R o+rx $CSVFILE_PATH
echo "COPY $TABLE_TO_LOAD  FROM '/pgdata/$CSV_FILE_PATH' WITH (FORMAT csv);" > /tmp/copycommand
psql -U $DB_USER -h $DB_HOST $DB_DATABASE -f /tmp/copycommand
echo "csvload has ended!"
