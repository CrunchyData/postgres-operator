#!/bin/bash

# Copyright 2019 - 2020 Crunchy Data Solutions, Inc.
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

echo "Test Runner...."

TESTS=(
    TestBackrestBackup
    TestBenchmark
    TestDf
    TestReload
    TestCreateLabel
    TestLs
    TestCat
    TestCreatePolicy
    TestShowPolicy
    TestScale
    TestStatus
    TestShowConfig
    TestVersion
    TestBouncer
    TestPool
    TestCreateUser
    TestUpdateUser
    TestShowUser
    TestDeleteUser
    TestShowCluster
    TestPGOTestCluster
)

echo "TESTS: "

#printf '%s\n' "${TESTS[@]}"

var=0
for each in "${TESTS[@]}"
do
  echo "$var $each"
  let "var++"
done

echo "Enter the value of the test you'd like to run, or 'all' to run all tests"
read runme


if [ $runme = "all" ]; then
	go test ./... -v --kubeconfig=$HOME/.kube/config -clustername=foomatic -namespace=pgouser1
else
  go test -run ${TESTS[runme]} -v --kubeconfig=$HOME/.kube/config -clustername=foomatic -namespace=pgouser1
fi
