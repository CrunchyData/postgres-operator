#!/usr/bin/env bash

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

set -eu

# Create and cleanup a temporary directory.
DIR="$(mktemp -d)"
trap "rm -rf '$DIR'" EXIT

# Build code generators that match the version of "k8s.io/client-go".
version="$(go list -f '{{ .Version }}' -m k8s.io/client-go)"
( cd "$DIR" && go mod init tmp && go get "k8s.io/code-generator/cmd/...@${version}" )

# Check that the script exists and expects Bash.
tool="$(go list -f '{{ .Dir }}' -m "k8s.io/code-generator@${version}")/generate-groups.sh"
bang="$(head -n1 "$tool")"
grep -wq bash <<< "$bang"

# Generate ./pkg/generated/{clientset/versioned,informers,listers} from objects defined in ./pkg/apis

groups_parent_directory='pkg/apis'
groups_parent_package="$(go list -m)/${groups_parent_directory}"
target_directory='pkg/generated'
target_package="$(go list -m)/${target_directory}"

# space-separated list of Groups with comma-separated lists of Versions 'g1:v1,v2 g2:v3,v4'
groups_with_versions='crunchydata.com:v1'

bash -- "$tool" all \
	"$target_package" "$groups_parent_package" "$groups_with_versions" \
	--go-header-file 'hack/boilerplate.go.txt' \
	--output-base="$DIR"

[ ! -d "$target_directory" ] || rm -r "$target_directory"
mv "${DIR}/${target_package}" "${target_directory}"


# DeepCopy functions are also generated in the group packages.

shopt -s globstar
for file in "${DIR}/${groups_parent_package}"/**/*.go ; do
	mv "$file" "${groups_parent_directory}${file#${DIR}/${groups_parent_package}}"
done
