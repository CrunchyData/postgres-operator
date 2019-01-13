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

#
# generate self signed cert for apiserver REST service
#

openssl req \
-x509 \
-nodes \
-newkey rsa:2048 \
-keyout $COROOT/conf/postgres-operator/server.key \
-out $COROOT/conf/postgres-operator/server.crt \
-days 3650 \
-subj "/C=US/ST=Texas/L=Austin/O=TestOrg/OU=TestDepartment/CN=*"

# generate CA
#openssl genrsa -out $COROOT/conf/apiserver/rootCA.key 4096
#openssl req -x509 -new -key $COROOT/conf/apiserver/rootCA.key -days 3650 -out $COROOT/conf/apiserver/rootCA.crt

# generate cert for secure.domain.com signed with the created CA
#openssl genrsa -out $COROOT/conf/apiserver/secure.domain.com.key 2048
#openssl req -new -key $COROOT/conf/apiserver/secure.domain.com.key -out $COROOT/conf/apiserver/secure.domain.com.csr
#In answer to question `Common Name (e.g. server FQDN or YOUR name) []:` you should set `secure.domain.com` (your real domain name)
#openssl x509 -req -in $COROOT/conf/apiserver/secure.domain.com.csr -CA $COROOT/conf/apiserver/rootCA.crt -CAkey $COROOT/conf/apiserver/rootCA.key -CAcreateserial -days 365 -out $COROOT/conf/apiserver/secure.domain.com.crt

#openssl genrsa 2048 > $COROOT/conf/apiserver/key.pem
#openssl req -new -x509 -key $COROOT/conf/apiserver/key.pem -out $COROOT/conf/apiserver/cert.pem -days 1095
