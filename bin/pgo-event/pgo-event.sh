#!/bin/bash -x

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
        kill -9 $(pidof nsqd)
        kill -9 $(pidof nsqadmin)
}

echo "pgo-event starting"

trap 'trap_sigterm' SIGINT SIGTERM

echo "pgo-event starting nsqadmin"

/usr/local/bin/nsqadmin  --http-address=0.0.0.0:4171  --nsqd-http-address=0.0.0.0:4151 &

sleep 3

echo "pgo-event starting nsqd"

/usr/local/bin/nsqd --data-path=/tmp --http-address=0.0.0.0:4151 --tcp-address=0.0.0.0:4150 --log-level=warn &

echo "pgo-event waiting till sigterm"

wait

echo "end of pgo-event"
