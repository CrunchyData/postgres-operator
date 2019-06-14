#!/bin/bash -x

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

function trap_sigterm() {
    echo "Signal trap triggered, beginning shutdown.."

    if ! pgrep nsqlookupd > /dev/null
    then
        kill -9 $(pidof nsqlookupd)
    fi
    if ! pgrep nsqd > /dev/null
    then
        kill -9 $(pidof nsqd)
    fi
}

echo "pgo-event starting"

trap 'trap_sigterm' SIGINT SIGTERM

/usr/local/bin/nsqlookupd &

sleep 3

echo "pgo-event starting nsqd"

/usr/local/bin/nsqd --data-path=/tmp --lookupd-tcp-address=127.0.0.1:4160

sleep 30

echo "end of pgo-event"
