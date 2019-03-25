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

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

IFS=', ' read -r -a array <<< "$PGO_OPERATOR_NAMESPACE"

echo "creating namespaces to deploy the Operator into..."
for ns in "${array[@]}"
do
	$PGO_CMD get namespace $ns > /dev/null 2> /dev/null
	if [ $? -eq 0 ]
	then
		echo namespace $ns is already created
	else
		$PGO_CMD create namespace $ns > /dev/null
		echo namespace $ns created
	fi
done

IFS=', ' read -r -a array <<< "$NAMESPACE"

echo ""
echo "creating namespaces for the Operator to watch and create PG clusters into..."
for ns in "${array[@]}"
do
	$PGO_CMD get namespace $ns > /dev/null 2> /dev/null
	if [ $? -eq 0 ]
	then
		echo namespace $ns is already created
	else
		$PGO_CMD create namespace $ns > /dev/null
		echo namespace $ns created
	fi
done
