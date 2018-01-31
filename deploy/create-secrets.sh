#!/bin/bash 
# Copyright 2016 Crunchy Data Solutions, Inc.
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

# there are 3 users defined (primaryuser, postgres, testuser)
# each has a default password set to 'password'
# each credential is stored in a secrets file and read at runtime
# by the pgo-apiserver

echo -n "primaryuser" > $DIR/username.txt
echo -n "password" > $DIR/password.txt
$CO_CMD --namespace=$CO_NAMESPACE create secret generic pgo-primary-user-pass \
	--from-file=$DIR/username.txt --from-file=$DIR/password.txt
echo "created pgo-primary-user-pass secret"

echo -n "postgres" > $DIR/username.txt
echo -n "password" > $DIR/password.txt
$CO_CMD --namespace=$CO_NAMESPACE create secret generic pgo-postgres-user-pass \
	--from-file=$DIR/username.txt --from-file=$DIR/password.txt
echo "created pgo-postgres-user-pass secret"

echo -n "testuser" > $DIR/username.txt
echo -n "password" > $DIR/password.txt
$CO_CMD --namespace=$CO_NAMESPACE create secret generic pgo-testuser-user-pass \
	--from-file=$DIR/username.txt --from-file=$DIR/password.txt
echo "created pgo-testuser-user-pass secret"


