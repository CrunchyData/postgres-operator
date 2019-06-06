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

#
# generate self signed cert for apiserver REST service
#

openssl req \
-x509 \
-nodes \
-newkey rsa:2048 \
-keyout $PGOROOT/conf/postgres-operator/server.key \
-out $PGOROOT/conf/postgres-operator/server.crt \
-days 3650 \
-subj "/C=US/ST=Texas/L=Austin/O=TestOrg/OU=TestDepartment/CN=*"

# generate CA
#openssl genrsa -out $PGOROOT/conf/apiserver/rootCA.key 4096
#openssl req -x509 -new -key $PGOROOT/conf/apiserver/rootCA.key -days 3650 -out $PGOROOT/conf/apiserver/rootCA.crt

# generate cert for secure.domain.com signed with the created CA
#openssl genrsa -out $PGOROOT/conf/apiserver/secure.domain.com.key 2048
#openssl req -new -key $PGOROOT/conf/apiserver/secure.domain.com.key -out $PGOROOT/conf/apiserver/secure.domain.com.csr
#In answer to question `Common Name (e.g. server FQDN or YOUR name) []:` you should set `secure.domain.com` (your real domain name)
#openssl x509 -req -in $PGOROOT/conf/apiserver/secure.domain.com.csr -CA $PGOROOT/conf/apiserver/rootCA.crt -CAkey $PGOROOT/conf/apiserver/rootCA.key -CAcreateserial -days 365 -out $PGOROOT/conf/apiserver/secure.domain.com.crt

#openssl genrsa 2048 > $PGOROOT/conf/apiserver/key.pem
#openssl req -new -x509 -key $PGOROOT/conf/apiserver/key.pem -out $PGOROOT/conf/apiserver/cert.pem -days 1095
