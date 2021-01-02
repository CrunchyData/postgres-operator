#!/bin/bash

# Copyright 2019 - 2021 Crunchy Data Solutions, Inc.
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

function trap_sigterm() {
    echo "Signal trap triggered, beginning shutdown.."

    if ! pgrep pgo-scheduler > /dev/null
    then
        kill -9 $(pidof pgo-scheduler)
    fi
}

trap 'trap_sigterm' SIGINT SIGTERM

/opt/cpm/bin/pgo-scheduler &

wait
