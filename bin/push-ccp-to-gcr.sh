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

GCR_IMAGE_PREFIX=gcr.io/crunchy-dev-test

CCP_IMAGE_PREFIX=crunchydata
CCP_IMAGE_TAG=centos7-11.4-2.4.1

IMAGES=(
crunchy-prometheus
crunchy-grafana
crunchy-collect
crunchy-pgbadger
crunchy-pgpool
crunchy-backup
crunchy-postgres
crunchy-pgbouncer
)

for image in "${IMAGES[@]}"
do
        docker tag $CCP_IMAGE_PREFIX/$image:$CCP_IMAGE_TAG   \
                $GCR_IMAGE_PREFIX/$image:$CCP_IMAGE_TAG
        gcloud docker -- push $GCR_IMAGE_PREFIX/$image:$CCP_IMAGE_TAG
done

