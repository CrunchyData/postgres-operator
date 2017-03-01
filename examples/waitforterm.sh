#!/bin/bash

# Copyright 2016 Crunchy Data Solutions, Inc.
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

#
# wait for a pod to terminate
#

echo "waiting on " $1
POD=$1
CMD=$2

while true; do
	$CMD get pod $POD > /dev/null
	rc=$?
#	echo $rc " is the rc"
	if   [ $rc -ne 0 ]; then
		echo "dead " $?
		break
	fi
	if  [ $rc -eq 0 ]; then
		echo -n "."
	fi
	sleep 1
done
