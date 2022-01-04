#!/usr/bin/env bash

# Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
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

# Inputs / outputs
IN_PACKAGES=("$@")
OUT_DIR=licenses

# Clean up before we start our work
rm -rf ${OUT_DIR:?}/*/

# Download dependencies of the requested packages, excluding the main module.
# - https://golang.org/ref/mod#glos-main-module
module=$(go list -m)
modules=$(go list -deps -f '{{with .Module}}{{.Path}}{{"\t"}}{{.Dir}}{{end}}' "${IN_PACKAGES[@]}")
dependencies=$(grep -v "^${module}" <<< "${modules}")

while IFS=$'\t' read -r module directory; do
	licenses=$(find "${directory}" -type f -ipath '*license*' -not -name '*.go')
	[ -n "${licenses}" ] || continue

	while IFS= read -r license; do
		# Replace the local module directory with the module path.
		# - https://golang.org/ref/mod#module-path
		relative="${module}${license:${#directory}}"

		# Copy the license file with the same layout as the module.
		destination="${OUT_DIR}/${relative%/*}"
		install -d "${destination}"
		install -m 0644 "${license}" "${destination}"
	done <<< "${licenses}"
done <<< "${dependencies}"
