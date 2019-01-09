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

APISERVER_FILES=postgres-operator/files/apiserver

openssl req \
-x509 \
-nodes \
-newkey rsa:2048 \
-keyout "${APISERVER_FILES}"/server.key \
-out "${APISERVER_FILES}"/server.crt \
-days 3650 \
-subj "/C=US/ST=Texas/L=Austin/O=TestOrg/OU=TestDepartment/CN=*"

#
# generate ssh keys for pgBackRest
#

BACKREST_REPO_FILES=postgres-operator/files/pgo-backrest-repo

ssh-keygen -f "${BACKREST_REPO_FILES}"/id_rsa -t rsa -N ''
ssh-keygen -t rsa -f "${BACKREST_REPO_FILES}"/ssh_host_rsa_key -N ''
ssh-keygen -t ecdsa -f "${BACKREST_REPO_FILES}"/ssh_host_ecdsa_key -N ''
ssh-keygen -t ed25519 -f "${BACKREST_REPO_FILES}"/ssh_host_ed25519_key -N ''
cp "${BACKREST_REPO_FILES}"/id_rsa.pub "${BACKREST_REPO_FILES}"/authorized_keys

rm -rf "${BACKREST_REPO_FILES}"/*.pub
