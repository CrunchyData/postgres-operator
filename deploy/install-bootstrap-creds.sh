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

# fill out these variables if you want to change the
# default pgo bootstrap user and role
export EXAMPLE_USERNAME=pgoadmin
export EXAMPLE_PASSWORD=examplepassword
export EXAMPLE_ROLENAME=pgoadmin
export PGOADMIN_PERMS=`echo -n "DeleteNamespace,CreateNamespace,UpdatePgorole,ShowPgorole,DeletePgorole,CreatePgorole,UpdatePgouser,ShowPgouser,DeletePgouser,CreatePgouser,Cat,Ls,ShowNamespace,CreateDump,RestoreDump,ScaleCluster,CreateSchedule,DeleteSchedule,ShowSchedule,DeletePgbouncer,CreatePgbouncer,DeletePgpool,CreatePgpool,Restore,RestorePgbasebackup,ShowSecrets,Reload,ShowConfig,Status,DfCluster,DeleteCluster,ShowCluster,CreateCluster,TestCluster,ShowBackup,DeleteBackup,CreateBackup,Label,Load,CreatePolicy,DeletePolicy,ShowPolicy,ApplyPolicy,ShowWorkflow,ShowPVC,CreateUpgrade,CreateUser,DeleteUser,UpdateUser,ShowUser,Version,CreateFailover,UpdateCluster,CreateBenchmark,ShowBenchmark,DeleteBenchmark,UpdateNamespace" | base64 --wrap=0`
export PGOADMIN_ROLENAME=`echo -n $EXAMPLE_ROLENAME | base64 --wrap=0`

# see if the bootstrap pgorole Secret exists or not, deleting it if found
$PGO_CMD get secret pgorole-$EXAMPLE_ROLENAME -n $PGO_OPERATOR_NAMESPACE 2> /dev/null
if [ $? -eq 0 ]; then
	$PGO_CMD delete secret pgorole-$EXAMPLE_ROLENAME -n $PGO_OPERATOR_NAMESPACE
fi

expenv -f $DIR/pgorole-pgoadmin.yaml | $PGO_CMD create -f -

# see if the bootstrap pgouser Secret exists or not, deleting it if found
export PGOADMIN_USERNAME=`echo -n $EXAMPLE_USERNAME | base64 --wrap=0`
export PGOADMIN_PASSWORD=`echo -n $EXAMPLE_PASSWORD | base64 --wrap=0`
$PGO_CMD get secret pgouser-$EXAMPLE_USERNAME -n $PGO_OPERATOR_NAMESPACE  2> /dev/null
if [ $? -eq 0 ]; then
	$PGO_CMD delete secret pgouser-$EXAMPLE_USERNAME -n $PGO_OPERATOR_NAMESPACE
fi
expenv -f $DIR/pgouser-admin.yaml | $PGO_CMD create -f -


