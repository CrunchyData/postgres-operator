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

echo "Getting project dependencies..."
BINDIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
POSTGRES_EXPORTER_VERSION=0.10.1

# Download Postgres Exporter, only required to build the Crunchy Postgres Exporter container
wget -O $PGOROOT/postgres_exporter.tar.gz https://github.com/prometheus-community/postgres_exporter/releases/download/v${POSTGRES_EXPORTER_VERSION?}/postgres_exporter-${POSTGRES_EXPORTER_VERSION?}.linux-amd64.tar.gz

# pgMonitor Setup
source $BINDIR/get-pgmonitor.sh
