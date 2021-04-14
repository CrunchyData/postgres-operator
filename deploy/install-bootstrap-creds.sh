#!/bin/bash

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

set -eu

# fill out these variables if you want to change the
# default pgo bootstrap user and role
PGOADMIN_USERNAME="${PGOADMIN_USERNAME:-admin}"
PGOADMIN_PASSWORD="${PGOADMIN_PASSWORD:-examplepassword}"
PGOADMIN_ROLENAME="${PGOADMIN_ROLENAME:-pgoadmin}"
PGOADMIN_PERMS="*"


$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" delete secret "pgorole-$PGOADMIN_ROLENAME" --ignore-not-found
$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" create secret generic "pgorole-$PGOADMIN_ROLENAME" \
	--from-literal="rolename=$PGOADMIN_ROLENAME" \
	--from-literal="permissions=$PGOADMIN_PERMS"
$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" label secret "pgorole-$PGOADMIN_ROLENAME" \
	'vendor=crunchydata' 'pgo-pgorole=true' "rolename=$PGOADMIN_ROLENAME"

$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" delete secret "pgouser-$PGOADMIN_USERNAME" --ignore-not-found
$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" create secret generic "pgouser-$PGOADMIN_USERNAME" \
	--from-literal="username=$PGOADMIN_USERNAME" \
	--from-literal="password=$PGOADMIN_PASSWORD" \
	--from-literal="roles=$PGOADMIN_ROLENAME"
$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" label secret "pgouser-$PGOADMIN_USERNAME" \
	'vendor=crunchydata' 'pgo-pgouser=true' "username=$PGOADMIN_USERNAME"
