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

echo "testing for dependencies " | tee -a $LOG

which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "wget is missing on your system, a required dependency" | tee -a $LOG
	exit 1
fi
which kubectl > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "kubectl is missing on your system, a required dependency" | tee -a $LOG
	exit 1
fi

echo ""
echo "testing kubectl connection" | tee -a $LOG
echo ""
kubectl get namespaces 
if [[ $? -ne 0 ]]; then
	echo "kubectl is not connecting to your Kube Cluster, required to proceed" | tee -a $LOG
	exit 1
fi

NAMESPACE=`kubectl config current-context`
echo "will install postgres-operator into the current namespace which is ["$NAMESPACE"]"

echo -n "do you want to continue the installation? [yes no] "
read REPLY
if [[ "$REPLY" != "yes" ]]; then
	echo "aborting installation"
	exit 1
fi

export CO_CMD=kubectl
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

echo "setting environment variables in $HOME/.bashrc" | tee -a $LOG

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.key
#
EOF

echo "setting up installation directory " | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/postgres-operator

echo ""
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

echo ""
echo "installing pgo client" | tee -a $LOG

mv pgo $GOBIN
mv pgo-mac $GOBIN
mv pgo.exe $GOBIN
mv expenv.exe $GOBIN
mv expenv-mac $GOBIN
mv expenv $GOBIN

echo "storage classes on your system..."
kubectl get sc
echo ""
echo -n "enter the name of the storage class to use: "
read STORAGE_CLASS

echo ""
echo "setting up pgo storage configuration for storageclass" | tee -a $LOG
cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/standard/'"$STORAGE_CLASS"'/' $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/demo/'"$NAMESPACE"'/' $COROOT/deploy/service-account.yaml
sed --in-place=.bak 's/demo/'"$NAMESPACE"'/' $COROOT/deploy/rbac.yaml

echo ""
echo "setting up pgo client auth" | tee -a $LOG
echo "username:password" > $HOME/.pgouser

echo "for pgo bash completion you will need to install the bash-completion package" | tee -a $LOG

cp $COROOT/examples/pgo-bash-completion $HOME/.bash_completion

echo -n "do you want to deploy the operator? [yes no] "
read REPLY
if [[ "$REPLY" == "yes" ]]; then
	echo "deploy the operator to the Kube cluster" | tee -a $LOG
	$COROOT/deploy/deploy.sh | tee -a $LOG
fi

echo "install complete" | tee -a $LOG
echo ""

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`kubectl get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo "kubectl port-forward " $podname " 18443:8443"
echo "do this in another terminal or run in the background"

echo ""
echo "WARNING:  for the postgres-operator settings to take effect, log out of your session and back in, or reload your .bashrc file"

echo ""
echo "NOTE:  to access the pgo CLI, place it within your PATH, it is located in $HOME/odev/bin/pgo"


