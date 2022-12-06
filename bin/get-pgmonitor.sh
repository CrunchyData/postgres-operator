#!/bin/bash -e

# Copyright 2017 - 2022 Crunchy Data Solutions, Inc.
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

echo "Getting pgMonitor..."
PGMONITOR_COMMIT='v4.8.0'

# pgMonitor Setup
if [[ -d ${PGOROOT?}/tools/pgmonitor ]]
then
    rm -rf ${PGOROOT?}/tools/pgmonitor
fi

git clone https://github.com/CrunchyData/pgmonitor.git ${PGOROOT?}/tools/pgmonitor
cd ${PGOROOT?}/tools/pgmonitor
git checkout ${PGMONITOR_COMMIT?}
