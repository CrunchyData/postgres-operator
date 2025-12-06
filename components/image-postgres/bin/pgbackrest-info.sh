#!/bin/bash

# Copyright 2017 - 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0

printf '|'
pgbackrest info --output=json --log-level-console=info --log-level-stderr=warn
echo
