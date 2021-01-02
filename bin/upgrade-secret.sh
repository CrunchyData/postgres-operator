#!/bin/bash

# Copyright 2018 - 2021 Crunchy Data Solutions, Inc.
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

# this is a script used to upgrade a database user credential from
# the old pre-2.5 release format to the 2.5 post format
# it will prompt the user along the way

echo "CLUSTER is " $1

CLUSTER=$1

CURRENT_POSTGRES_PASSWORD=`kubectl get secret $CLUSTER-root-secret -o jsonpath="{.data.password}"`
echo "current decoded postgres password is..."
POSTGRES_PASSWORD=`echo -n $CURRENT_POSTGRES_PASSWORD | base64 --decode`
echo $POSTGRES_PASSWORD

USERNAME=postgres

kubectl create secret generic $CLUSTER-$USERNAME-secret \
	--from-literal=username=$USERNAME \
	--from-literal=password=$POSTGRES_PASSWORD

kubectl label secret $CLUSTER-$USERNAME-secret pg-cluster=$CLUSTER

# do the same for the primaryuser

CURRENT_PASSWORD=`kubectl get secret $CLUSTER-primary-secret -o jsonpath="{.data.password}"`
echo "current decoded primaryuser password is..."
POSTGRES_PASSWORD=`echo -n $CURRENT_PASSWORD | base64 --decode`
echo $POSTGRES_PASSWORD

USERNAME=primaryuser

kubectl create secret generic $CLUSTER-$USERNAME-secret \
	--from-literal=username=$USERNAME \
	--from-literal=password=$POSTGRES_PASSWORD

kubectl label secret $CLUSTER-$USERNAME-secret pg-cluster=$CLUSTER

# do the same for the testuser

USERNAME=testuser

CURRENT_PASSWORD=`kubectl get secret $CLUSTER-user-secret -o jsonpath="{.data.password}"`
echo "current decoded testuser password is..."
POSTGRES_PASSWORD=`echo -n $CURRENT_PASSWORD | base64 --decode`
echo $POSTGRES_PASSWORD

kubectl create secret generic $CLUSTER-$USERNAME-secret \
	--from-literal=username=$USERNAME \
	--from-literal=password=$POSTGRES_PASSWORD

kubectl label secret $CLUSTER-$USERNAME-secret pg-cluster=$CLUSTER
