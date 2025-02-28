#!/usr/bin/env bash
#
# Copyright 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# This runs `$GO get` without changing the "go" directive in the "go.mod" file.
# To change that, pass a "go@go{version}" argument.
#
# https://go.dev/doc/toolchain
#
# Usage: $0 help
# Usage: $0 -u golang.org/x/crypto
# Usage: $0 -u golang.org/x/crypto go@go1.99.0
#

set -eu
: "${GO:=go}"

if [[ "$#" -eq 0 ]] || [[ "$1" == 'help' ]] || [[ "$*" == *'--help'* ]] || [[ "$*" == *'--version'* ]]
then
	self=$(command -v -- "$0")
	content=$(< "${self}")
	content="${content%%$'\n\n'*}"
	content="#${content#*$'\n#'}"
	content="${content//$'$GO'/${GO}}"
	exec echo "${content//$'$0'/$0}"
fi

version=$(${GO} list -m -f 'go@go{{.GoVersion}}')

for arg in "$@"
do case "${arg}" in go@go*) version="${arg}" ;; *) esac
done

${GO} get "$@" "${version}" 'toolchain@none'
${GO} mod tidy
