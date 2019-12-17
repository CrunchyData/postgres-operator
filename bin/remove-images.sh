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

for CNAME in pgo-event pgo-scheduler pgo-sqlrunner pgo-backrest-restore pgo-backrest-repo postgres-operator pgo-load pgo-apiserver pgo-rmdata pgo-backrest pgo-client
do
	sudo buildah rmi -f localhost/$PGO_IMAGE_PREFIX/$CNAME:$PGO_IMAGE_TAG
	docker rmi -f $CNAME docker.io/$PGO_IMAGE_PREFIX/$CNAME:$PGO_IMAGE_TAG
done
