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

CCP_IMAGE_PREFIX=crunchydata
CCP_IMAGE_TAG=centos7-10.5-2.1.0

for CNAME in crunchy-postgres crunchy-collect crunchy-grafana crunchy-prometheus crunchy-backup crunchy-backrest-restore
do
	docker pull $CCP_IMAGE_PREFIX/$CNAME:$CCP_IMAGE_TAG
done
