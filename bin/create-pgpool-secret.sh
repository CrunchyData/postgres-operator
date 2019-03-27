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
echo -n "enter the cluster name:"
read CLUSTERNAME
echo -n "enter the directory of your pgpool config files:"
read FROMDIR
if [ ! -f $FROMDIR/pgpool.conf ]; then
	echo $FROMDIR/pgpool.conf not found...aborting
	exit 2
fi
if [ ! -f $FROMDIR/pool_hba.conf ]; then
	echo $FROMDIR/pool_hba.conf not found...aborting
	exit 2
fi
if [ ! -f $FROMDIR/pool_passwd ]; then
	echo $FROMDIR/pool_passwd not found...aborting
	exit 2
fi
$PGO_CMD create secret generic $CLUSTERNAME-pgpool-secret \
	--from-file=pgpool.conf=$FROMDIR/pgpool.conf \
	--from-file=pool_hba.conf=$FROMDIR/pool_hba.conf \
	--from-file=pool_passwd=$FROMDIR/pool_passwd
