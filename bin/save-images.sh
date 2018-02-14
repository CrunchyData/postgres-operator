#!/bin/bash 

# Copyright 2018 Crunchy Data Solutions, Inc.
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

docker save crunchydata/lspvc:$CO_IMAGE_TAG > /tmp/lspvc-$CO_IMAGE_TAG.tar
docker save crunchydata/postgres-operator:$CO_IMAGE_TAG > /tmp/postgres-operator-$CO_IMAGE_TAG.tar
docker save crunchydata/csvload:$CO_IMAGE_TAG > /tmp/csvload-$CO_IMAGE_TAG.tar
docker save crunchydata/apiserver:$CO_IMAGE_TAG > /tmp/apiserver-$CO_IMAGE_TAG.tar
