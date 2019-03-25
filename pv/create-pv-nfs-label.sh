#!/bin/bash
# Copyright 2017 Crunchy Data Solutions, Inc.
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


DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo "create the test PV and PVC using the NFS dir"
for i in {1..180}
do
   	echo "creating PV crunchy-pv$i"
	export COUNTER=$i
	$PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE delete pv crunchy-pv$i
	expenv -f $DIR/crunchy-pv-nfs-label.json | $PGO_CMD --namespace=$PGO_OPERATOR_NAMESPACE create -f -
done
