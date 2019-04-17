#!/bin/bash
# Copyright 2018 - 2019 Crunchy Data Solutions, Inc.
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

set -e

USERNAME="$1"
SERVER="$2"
OUTPUT_DIR="$3"

if [[ -z ${USERNAME?} ]] || [[ -z ${SERVER?} ]] || [[ -z ${OUTPUT_DIR?} ]]
then
    echo "Usage: ssl-creator.sh <CLIENT USERNAME> <SERVER NAME> <OUTPUT_DIR>"
    exit 1
fi

if [[ ! -x "$(command -v certstrap)" ]]
then
    echo "Certstrap is not installed.."
    echo "To install certstrap run: installcertstrap.sh"
    echo "Exiting.."
    exit 1
fi

certstrap --depot-path ${OUTPUT_DIR?}/out \
    init --common-name RootCA \
  --key-bits 4096 \
  --organization "Crunchy Data" \
  --locality "Charleston" \
  --province "SC" \
  --country "US" \
  --passphrase "" \
  --years 1

certstrap --depot-path ${OUTPUT_DIR?}/out request-cert --passphrase '' --common-name ${SERVER?}
certstrap --depot-path ${OUTPUT_DIR?}/out sign ${SERVER?} --passphrase '' --CA RootCA --years 1

certstrap --depot-path ${OUTPUT_DIR?}/out request-cert --passphrase '' --common-name ${USERNAME?}
certstrap --depot-path ${OUTPUT_DIR?}/out sign ${USERNAME?} --passphrase '' --CA RootCA --years 1

mkdir ${OUTPUT_DIR?}/certs

cp ${OUTPUT_DIR?}/out/RootCA.crt ${OUTPUT_DIR?}/certs/ca.crt
cp ${OUTPUT_DIR?}/out/RootCA.crl ${OUTPUT_DIR?}/certs/ca.crl

# Server
cp ${OUTPUT_DIR?}/out/${SERVER?}.key ${OUTPUT_DIR?}/certs/server.key
cat ${OUTPUT_DIR?}/out/${SERVER?}.crt ${OUTPUT_DIR?}/out/RootCA.crt > ${OUTPUT_DIR?}/certs/server.crt

# Client
cp ${OUTPUT_DIR?}/out/${USERNAME?}.key ${OUTPUT_DIR?}/certs/client.key
cat ${OUTPUT_DIR?}/out/${USERNAME?}.crt ${OUTPUT_DIR?}/out/RootCA.crt > ${OUTPUT_DIR?}/certs/client.crt

chmod 600 ${OUTPUT_DIR?}/certs/client.key ${OUTPUT_DIR?}/certs/client.crt

exit 0
