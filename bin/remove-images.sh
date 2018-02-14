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

docker rmi -f pgo-lspvc crunchydata/pgo-lspvc:$CO_IMAGE_TAG  \
postgres-operator crunchydata/postgres-operator:$CO_IMAGE_TAG  \
pgo-load crunchydata/pgo-load:$CO_IMAGE_TAG  \
pgo-apiserver crunchydata/pgo-apiserver:$CO_IMAGE_TAG \
pgo-rmdata crunchydata/pgo-rmdata:$CO_IMAGE_TAG 
