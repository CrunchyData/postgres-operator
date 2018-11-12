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

gcloud auth login
gcloud config set project container-suite

REGISTRY='us.gcr.io/container-suite'
LOCALREGISTRY=cortado-k1:5000/crunchydata
CCP_IMAGE_TAG=centos7-11.0-2.2.0-rc6
for CNAME in crunchy-scheduler crunchy-postgres crunchy-collect crunchy-grafana crunchy-prometheus crunchy-backup
do
	docker pull us.gcr.io/container-suite/$CNAME:$CCP_IMAGE_TAG
	docker tag $REGISTRY/$CNAME:$CCP_IMAGE_TAG $LOCALREGISTRY/$CNAME:$CCP_IMAGE_TAG
	docker push $LOCALREGISTRY/$CNAME:$CCP_IMAGE_TAG
done
