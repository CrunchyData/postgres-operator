#!/bin/bash -e

# Copyright 2020 - 2021 Crunchy Data Solutions, Inc.
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

echo "Ensuring project dependencies..."

if ! command -v go &> /dev/null; then
	echo 'Cannot find `go`. Perhaps:'
	echo '  sudo yum install golang'
	exit 1
fi
if ! sort -VC <<< $'go1.13\n'"$( read -ra array <<< "$(go version)"; echo "${array[2]-}" )"; then
	echo 'Old version of `go`: «' "$(go version)" '» Perhaps:'
	echo '  sudo yum update golang'
	exit 1
fi

if ! command -v buildah &> /dev/null; then
	echo 'Cannot find `buildah`. Perhaps:'
	echo '  sudo yum install buildah'
	exit 1
fi
if ! sort -VC <<< $'1.14.9\n'"$( read -ra array <<< "$(buildah --version)"; echo "${array[2]-}" )"; then
	echo 'Old version of `buildah`: «' "$(buildah --version)" '» Perhaps:'
	echo '  sudo yum update buildah'
	exit 1
fi
