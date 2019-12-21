#!/bin/bash
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


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# fill out these variables if you want to change the
# default pgo bootstrap user and role
export PGOADMIN_USERNAME=pgoadmin
export PGOADMIN_PASSWORD=examplepassword
export PGOADMIN_ROLENAME=pgoadmin
export PGOADMIN_PERMS="*"

# see if the bootstrap pgorole Secret exists or not, deleting it if found
$PGO_CMD get secret pgorole-$PGOADMIN_ROLENAME -n $PGO_OPERATOR_NAMESPACE 2> /dev/null > /dev/null
if [ $? -eq 0 ]; then
	$PGO_CMD delete secret pgorole-$PGOADMIN_ROLENAME -n $PGO_OPERATOR_NAMESPACE
fi

expenv -f $DIR/pgorole-pgoadmin.yaml | $PGO_CMD create -f -

# see if the bootstrap pgouser Secret exists or not, deleting it if found
$PGO_CMD get secret pgouser-$PGOADMIN_USERNAME -n $PGO_OPERATOR_NAMESPACE  2> /dev/null > /dev/null
if [ $? -eq 0 ]; then
	$PGO_CMD delete secret pgouser-$PGOADMIN_USERNAME -n $PGO_OPERATOR_NAMESPACE
fi
expenv -f $DIR/pgouser-admin.yaml | $PGO_CMD create -f -
