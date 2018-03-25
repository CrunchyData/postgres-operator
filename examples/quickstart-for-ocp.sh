#!/bin/bash

# Copyright 2018 Crunchy Data Solutions, Inc.
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

LOG="pgo-installer.log"

export PGORELEASE=2.6

echo "installing deps if necessary" | tee -a $LOG

which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "wget is missing on your system, a required dependency" | tee -a $LOG
	exit 1
fi
which oc > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "oc is missing on your system, a required dependency" | tee -a $LOG
	exit 1
fi

echo "testing oc connection" | tee -a $LOG
oc project
if [[ $? -ne 0 ]]; then
	echo "oc is not connecting to your Openshift Cluster, required to proceed" | tee -a $LOG
	exit 1
fi

echo "this script will install the postgres operator into the project listed below...."

oc project

echo -n "do you want to continue the installation? [yes no] "
read REPLY
if [[ "$REPLY" == "no" ]]; then
	exit 0
fi

echo "setting environment variables" | tee -a $LOG

export GOBIN=$GOPATH/bin
export GOPATH=$HOME/odev
export PATH=$PATH:$GOPATH/bin
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_APISERVER_URL=https://127.0.0.1:8443
export PGO_CA_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/apiserver/server.key

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export PATH=$PATH:$GOPATH/bin
export CO_APISERVER_URL=https://127.0.0.1:8443
export PGO_CA_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/apiserver/server.key
# 

EOF

echo "setting up directory structure" | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/postgres-operator

echo "installing pgo server config" | tee -a $LOG
wget --quiet https://github.com/CrunchyData/postgres-operator/releases/download/$PGORELEASE/postgres-operator.$PGORELEASE.tar.gz -O /tmp/postgres-operator.$PGORELEASE.tar.gz
if [[ $? -ne 0 ]]; then
	echo "problem getting pgo server config"
	exit 1
fi
cd $COROOT
tar xzf /tmp/postgres-operator.$PGORELEASE.tar.gz
if [[ $? -ne 0 ]]; then
	echo "problem getting 2.6 release"
	exit 1
fi

echo "installing pgo client" | tee -a $LOG

mv pgo $GOBIN
mv pgo-mac $GOBIN
mv pgo.exe $GOBIN
mv expenv.exe $GOBIN
mv expenv-mac $GOBIN
mv expenv $GOBIN

echo -n "enter the name of the storage class to use"
read STORAGE_CLASS

echo "setting up pgo storage configuration for storageclass" | tee -a $LOG
cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml
sed 's/standard/'"$STORAGE_CLASS"'/' $COROOT/conf/apiserver/pgo.yaml

echo "setting up pgo client auth" | tee -a $LOG
tail --lines=1 $COROOT/conf/apiserver/pgouser > $HOME/.pgouser

echo "for pgo bash completion you will need to install the bash-completion package" | tee -a $LOG

cp $COROOT/examples/pgo-bash-completion $HOME/.bash_completion
echo -n "do you want to deploy the operator? [yes no] "
read REPLY
if [[ "$REPLY" == "yes" ]]; then
	echo "deploy the operator to the OCP cluster" | tee -a $LOG
	$COROOT/deploy/deploy.sh > /dev/null 2> /dev/null | tee -a $LOG
fi

echo "install complete" | tee -a $LOG

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`oc get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo "oc port-forward " $podname " 8443:8443"
echo "do this in another terminal or run in the background"

echo "WARNING:  for the postgres-operator settings to take effect, log out of your session and back in, or reload your .bashrc file"

