#!/bin/bash 

# Copyright 2017-2018 Crunchy Data Solutions, Inc.
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

#echo $1 is the password
#echo $2 is the host ip

SLEEP_TIME=2 
FAILURES=0
MAX_FAILURES=7
while true; do
        sleep $SLEEP_TIME
        /usr/pgsql-10/bin/pg_isready  --dbname=postgres --host=$2 --port=5432  --username=postgres
        if [ $? -eq 0 ]
        then
                echo "Successfully reached master @ " `date`
		break
        else
                echo "Could not reach master @ " `date`
                FAILURES=$[$FAILURES+1]
                if [[ $FAILURES -lt $MAX_FAILURES ]]; then
                        continue
                fi
                echo "Maximum failures reached"
                exit 3
        fi
done

PGPASSWORD=$1 psql -p 5432 -h $2 -U postgres postgres
