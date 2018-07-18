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

export PGORELEASE=3.1
PGO_RELEASE_REMOTE_URL="https://github.com/CrunchyData/postgres-operator/releases/download/$PGORELEASE/postgres-operator.$PGORELEASE.tar.gz"
PGO_RELEASE_LOCAL_PATH="/tmp/postgres-operator.$PGORELEASE.tar.gz"

echo "Testing for dependencies..." | tee -a $LOG

which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	which curl > /dev/null 2> /dev/null
	if [[ $? -ne 0 ]]; then
		echo "The required dependency wget and/or curl is missing on your system." | tee -a $LOG
		exit 1
	else
		PGO_HTTP_CMD="curl -L -s -o ${PGO_RELEASE_LOCAL_PATH} ${PGO_RELEASE_REMOTE_URL}"
	fi
else
	PGO_HTTP_CMD="wget --quiet ${PGO_RELEASE_REMOTE_URL} -O ${PGO_RELEASE_LOCAL_PATH}"
fi
which kubectl > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency kubectl is missing on your system." | tee -a $LOG
	exit 1
fi

echo ""
echo "Testing kubectl connection..." | tee -a $LOG
echo ""
kubectl get namespaces
if [[ $? -ne 0 ]]; then
	echo "kubectl is not connecting to your Kubernetes Cluster. A successful connection is required to proceed." | tee -a $LOG
	exit 1
fi

echo "Connected to Kubernetes." | tee -a $LOG
echo ""

NAMESPACE=`kubectl config current-context`
echo "The postgres-operator will be installed into the current namespace which is ["$NAMESPACE"]."

echo -n "Do you want to continue the installation? [yes no] "
read REPLY
if [[ "$REPLY" != "yes" ]]; then
	echo "Aborting installation."
	exit 1
fi

export CO_CMD=kubectl
export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOPATH/bin
export CO_IMAGE_PREFIX=crunchydata
export CO_IMAGE_TAG=centos7-3.1
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/apiserver/server.key

echo "Setting environment variables in $HOME/.bashrc..." | tee -a $LOG

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/apiserver/server.key
#
EOF

echo "Setting up installation directory..." | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/postgres-operator

echo ""
echo "Installing pgo server configuration..." | tee -a $LOG
`${PGO_HTTP_CMD}`

cd $COROOT
tar xzf /tmp/postgres-operator.$PGORELEASE.tar.gz
if [[ $? -ne 0 ]]; then
	echo "ERROR: Problem unpackaging the $PGORELEASE release."
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

echo "The available storage classes on your system:"
kubectl get sc
echo ""
echo -n "Enter the name of the storage class to use: "
read STORAGE_CLASS

echo ""
echo "Setting up pgo storage configuration for the selected storageclass..." | tee -a $LOG
cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml
sed -i .bak 's/standard/'"$STORAGE_CLASS"'/' $COROOT/conf/apiserver/pgo.yaml
sed -i .bak 's/demo/'"$NAMESPACE"'/' $COROOT/deploy/service-account.yaml
sed -i .bak 's/demo/'"$NAMESPACE"'/' $COROOT/deploy/rbac.yaml

echo ""
echo "Setting up pgo client authentication..." | tee -a $LOG
echo "username:password" > $HOME/.pgouser

echo "For pgo bash completion you will need to install the bash-completion package." | tee -a $LOG

cp $COROOT/examples/pgo-bash-completion $HOME/.bash_completion

echo -n "Do you want to deploy the operator? [yes no] "
read REPLY
if [[ "$REPLY" == "yes" ]]; then
	echo "Deploying the operator to the Kubernetes cluster..." | tee -a $LOG
	$COROOT/deploy/deploy.sh | tee -a $LOG
fi

echo "Installation complete." | tee -a $LOG
echo ""

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`kubectl get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo "kubectl port-forward " $podname " 18443:8443"
echo "Run this in another terminal or in the background."

echo ""
echo "WARNING: For the postgres-operator settings to take effect, it is necessary to log out of your session and back in or reload your .bashrc file."

echo ""
echo "NOTE: In order to access the pgo CLI, place it within your PATH from its default location in $HOME/odev/bin/pgo."
