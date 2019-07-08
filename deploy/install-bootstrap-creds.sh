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

export PGOADMIN_PERMS=`echo -n "UpdatePgorole,ShowPgorole,DeletePgorole,CreatePgorole,UpdatePgouser,ShowPgouser,DeletePgouser,CreatePgouser,Cat,Ls,ShowNamespace,CreateDump,RestoreDump,ScaleCluster,CreateSchedule,DeleteSchedule,ShowSchedule,DeletePgbouncer,CreatePgbouncer,DeletePgpool,CreatePgpool,Restore,RestorePgbasebackup,ShowSecrets,Reload,ShowConfig,Status,DfCluster,DeleteCluster,ShowCluster,CreateCluster,TestCluster,ShowBackup,DeleteBackup,CreateBackup,Label,Load,CreatePolicy,DeletePolicy,ShowPolicy,ApplyPolicy,ShowWorkflow,ShowPVC,CreateUpgrade,CreateUser,DeleteUser,User,Version,CreateFailover,UpdateCluster,CreateBenchmark,ShowBenchmark,DeleteBenchmark" | base64 --wrap=0`

# see if the bootstrap Secret exists or not, creating it if not found
$PGO_CMD get secret pgorole-pgoadmin -n $PGO_OPERATOR_NAMESPACE
if [ $? -eq 1 ]; then
	expenv -f $DIR/pgorole-pgoadmin.yaml | $PGO_CMD create -f -
fi

# see if the bootstrap pgorole Secret exists or not, creating it if not found
$PGO_CMD get secret pgouser-admin -n $PGO_OPERATOR_NAMESPACE
if [ $? -eq 1 ]; then
	expenv -f $DIR/pgouser-admin.yaml | $PGO_CMD create -f -
fi


