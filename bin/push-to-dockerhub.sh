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

docker login
declare -a arr=(
"postgres-operator" "pgo-apiserver" "pgo-lspvc" "pgo-sqlrunner"
"pgo-rmdata" "pgo-backrest" "pgo-load" "pgo-scheduler"
"pgo-backrest-repo" "pgo-backrest-restore")

for image in "${arr[@]}"
do
docker push $PGO_IMAGE_PREFIX/$image:$PGO_IMAGE_TAG   
done
