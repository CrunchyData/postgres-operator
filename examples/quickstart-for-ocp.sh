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

echo "Testing for dependencies..." | tee -a $LOG

which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency wget is missing on your system." | tee -a $LOG
	exit 1
fi
which oc > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency oc is missing on your system." | tee -a $LOG
	exit 1
fi

echo ""
echo "Testing oc connection..." | tee -a $LOG
echo ""
oc project
if [[ $? -ne 0 ]]; then
	echo "oc is not able to connect to your OpenShift Cluster." | tee -a $LOG
	exit 1
fi

echo "Connected to project." | tee -a $LOG
echo ""
echo "This script will install the postgres operator into the project listed below."
oc project
echo ""

PROJECT=`oc project -q`
echo "Installing into project ["$PROJECT"]."
echo ""
echo -n "Do you want to continue the installation? [yes no] "
read REPLY
if [[ "$REPLY" == "no" ]]; then
	exit 0
fi

export CO_CMD=oc
export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOPATH/bin
export CO_IMAGE_PREFIX=crunchydata
export CO_IMAGE_TAG=centos7-2.6
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/apiserver/server.key

echo ""
echo "Setting environment variables in $HOME/.bashrc..." | tee -a $LOG
echo ""

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.key
#

EOF

echo ""
echo "Setting up installation directory..." | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/postgres-operator

echo ""
echo "Installing the pgo server config..." | tee -a $LOG
wget --quiet https://github.com/CrunchyData/postgres-operator/releases/download/$PGORELEASE/postgres-operator.$PGORELEASE.tar.gz -O /tmp/postgres-operator.$PGORELEASE.tar.gz
#if [[ $? -ne 0 ]]; then
#	echo "problem getting pgo server config"
#	exit 1
#fi
cd $COROOT
tar xzf /tmp/postgres-operator.$PGORELEASE.tar.gz
if [[ $? -ne 0 ]]; then
	echo "Error: problem with unpackaging the $PGORELEASE release."
	exit 1
fi

echo ""
echo "Installing pgo client..." | tee -a $LOG

mv pgo $GOBIN
mv pgo-mac $GOBIN
mv pgo.exe $GOBIN
mv expenv.exe $GOBIN
mv expenv-mac $GOBIN
mv expenv $GOBIN

echo "Storage classes on your system:"
oc get sc
echo ""
echo -n "Enter the name of the storage class to use: "
read STORAGE_CLASS

echo ""
echo "Setting up pgo storage configuration for the selected storageclass..." | tee -a $LOG
cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/standard/'"$STORAGE_CLASS"'/' $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/demo/'"$PROJECT"'/' $COROOT/deploy/service-account.yaml
sed --in-place=.bak 's/demo/'"$PROJECT"'/' $COROOT/deploy/rbac.yaml

echo ""
echo -n "Storage classes can require a fsgroup setting to be specified in the security context of your pods. Typically, this value is 26, but on some storage providers this value is blank. Enter your fsgroup setting if required or leave blank if not required: "
read FSGROUP
sed --in-place=.bak 's/26/'"$FSGROUP"'/' $COROOT/conf/apiserver/pgo.yaml

echo ""
echo "Setting up pgo client authentication..." | tee -a $LOG
echo "username:password" > $HOME/.pgouser

echo ""
echo "For pgo bash completion you will need to install the bash-completion package." | tee -a $LOG

cp $COROOT/examples/pgo-bash-completion $HOME/.bash_completion
echo ""
echo -n "Do you want to deploy the operator? [yes no] "
read REPLY
if [[ "$REPLY" == "yes" ]]; then
	echo "Deploying the operator to the OCP cluster..." | tee -a $LOG
#	$COROOT/deploy/deploy.sh > /dev/null 2> /dev/null | tee -a $LOG
	$COROOT/deploy/deploy.sh | tee -a $LOG
fi

echo "Installation complete." | tee -a $LOG
echo ""

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`oc get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo "oc port-forward " $podname " 18443:8443"
echo "Do this in another terminal or run in the background."

echo ""
echo "WARNING: for the postgres-operator settings to take effect, it will be necessary to log out of your session and back in or reload your .bashrc file."

echo ""
echo "NOTE: In order to access the pgo CLI, it will be necessary to place it within your PATH. The default location after installing is within $HOME/odev/bin/pgo"
