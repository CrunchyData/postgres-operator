#!/bin/bash

# Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

if [ $# -eq 2 ] && [ -f $1 ] && [ -f $2 ]; then
    ROLE_INPUT=${1}
    USER_INPUT=${2}
    echo "Using command line input"
fi

fail=false
if [ ! -f $ROLE_INPUT ]; then
    echo "File not found in $ROLE_INPUT"
    fail=true
fi
if [ ! -f $USER_INPUT ]; then
    echo "File not found in $USER_INPUT"
    fail=true
fi

if $fail; then
    echo "Please provide the path for your pgouser and pgorole files"
    echo "upgrade-certs.sh /path/to/pgorole /path/to/pgouser"
    exit 1
fi

while read -r line
do
    IFS=':' read -r role perms <<< $line
    if [ -z "$role" ] || [ -z "$perms" ]; then
        echo "Role input file invalid. Expected format \"rolename:perm1,perm2,perm3\""
        exit 1
    fi

    export PGO_ROLENAME=$role
    export PGO_PERMS=$perms

    # see if the bootstrap pgorole Secret exists or not, deleting it if found
    $PGO_CMD get secret pgorole-$PGO_ROLENAME -n $PGO_OPERATOR_NAMESPACE 2> /dev/null
    if [ $? -eq 0 ]; then
        $PGO_CMD delete secret pgorole-$PGO_ROLENAME -n $PGO_OPERATOR_NAMESPACE
    fi

    cat $DIR/pgorole.yaml | envsubst | $PGO_CMD create -f -
done < "$ROLE_INPUT"


while read -r line
do
    IFS=':' read -r user pass role <<< $line
    if [ -z "$user" ] || [ -z "$pass" ] || [ -z "$role" ]; then
        echo "User input file invalid. Expected format \"username:password:rolename\""
        exit 1
    fi

    export PGO_USERNAME=$user
    export PGO_PASSWORD=$pass
    export PGO_ROLENAME=$role

    # see if the bootstrap pgouser Secret exists or not, deleting it if found
    $PGO_CMD get secret pgouser-$PGO_USERNAME -n $PGO_OPERATOR_NAMESPACE  2> /dev/null
    if [ $? -eq 0 ]; then
        $PGO_CMD delete secret pgouser-$PGO_USERNAME -n $PGO_OPERATOR_NAMESPACE
    fi
    cat $DIR/pgouser.yaml | envsubst | $PGO_CMD create -f -
done < "$USER_INPUT"
