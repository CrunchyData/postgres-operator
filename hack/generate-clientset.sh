#!/usr/bin/env bash

# Copyright 2020 Crunchy Data Solutions, Inc.
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

# Find the Go install path.
[ "${GOBIN:-}" ] || GOBIN="$(go env GOBIN)"
[ "${GOBIN:-}" ] || GOBIN="$(go env GOPATH)/bin"

# Create and cleanup a temporary directory.
DIR="$(mktemp -d)"
trap "rm -rf '$DIR'" EXIT

# Find `client-gen` on the current PATH or install it to the Go install path.
tool="$(command -v client-gen || true)"
[ -n "$tool" ] || tool="$GOBIN/client-gen"
[ -x "$tool" ] || ( cd "$DIR" && go mod init tmp && go get 'k8s.io/code-generator/cmd/client-gen@v0.17.9' )

# Generate ./pkg/generated/clientset/versioned from objects defined in ./pkg/apis/crunchydata.com/...

target_directory='pkg/generated/clientset'
target_package="$(go list -m)/${target_directory}"

"$tool" \
	--clientset-name='versioned' \
	--go-header-file='hack/boilerplate.go.txt' \
	--input-base='' --input="$(go list ./pkg/apis/crunchydata.com/... | paste -sd, -)" \
	--output-base="$DIR" --output-package="$target_package" \

[ ! -d "$target_directory" ] || rm -r "$target_directory"
mv "${DIR}/${target_package}" "${target_directory}"
