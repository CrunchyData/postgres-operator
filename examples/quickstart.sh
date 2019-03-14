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

LOG="pgo-installer.log"

if [[ "$PGO_BASEOS" != "" ]]; then
	echo "PGO_BASEOS is set to " $PGO_BASEOS
else
	export PGO_BASEOS=centos7
	echo -n "Which crunchy images do you want to install (rhel7 or centos7)? ["$REPLY"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting PGO_BASEOS="$REPLY
		export PGO_BASEOS=$REPLY
	fi
fi
echo $PGO_BASEOS is the baseos entered | tee -a $LOG

if [[ "$PGO_VERSION" != "" ]]; then
	echo "PGO_VERSION is set to " $PGO_VERSION
else
	export PGO_VERSION=3.4.0
	echo -n "Which Operator version do you want to install? ["$PGO_VERSION"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting PGO_VERSION="$REPLY
		export PGO_VERSION=$REPLY
	fi
fi
echo $PGO_VERSION is the version entered | tee -a $LOG

if [[ "$PGO_NAMESPACE" != "" ]]; then
	echo "PGO_NAMESPACE is set to " $PGO_NAMESPACE
else
	export PGO_NAMESPACE=demo
	echo -n "Which namespace do you want to install into? ["$PGO_NAMESPACE"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting PGO_NAMESPACE="$REPLY
		export PGO_NAMESPACE=$REPLY
	fi
fi
echo $PGO_NAMESPACE is the namespace entered | tee -a $LOG
export PGO_NAMESPACE=$PGO_NAMESPACE

if [[ "$CCP_IMAGE_TAG" != "" ]]; then
	echo "CCP_IMAGE_TAG is set to " $CCP_IMAGE_TAG
else
	export CCP_IMAGE_TAG=$PGO_BASEOS-11.1-2.3.0
	echo -n "Which CCP image tag do you want to use? ["$CCP_IMAGE_TAG"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting CCP_IMAGE_TAG="$REPLY
		export CCP_IMAGE_TAG=$REPLY
	fi
fi
echo $CCP_IMAGE_TAG is the CCP image tag entered | tee -a $LOG

if [[ "$PGO_IMAGE_PREFIX" != "" ]]; then
	echo "PGO_IMAGE_PREFIX is set to " $PGO_IMAGE_PREFIX
	CCP_IMAGE_PREFIX=$PGO_IMAGE_PREFIX
	echo "CCP_IMAGE_PREFIX is set to " $CCP_IMAGE_PREFIX
else
	export PGO_IMAGE_PREFIX=crunchydata
	export CCP_IMAGE_PREFIX=$PGO_IMAGE_PREFIX
	echo -n "Which image prefix do you want to use? ["$PGO_IMAGE_PREFIX"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting PGO_IMAGE_PREFIX="$REPLY
		echo "setting CCP_IMAGE_PREFIX="$REPLY
		export PGO_IMAGE_PREFIX=$REPLY
		export CCP_IMAGE_PREFIX=$REPLY
	fi
fi
echo $PGO_IMAGE_PREFIX is the CO image prefix entered | tee -a $LOG
echo $CCP_IMAGE_PREFIX is the CCP image prefix entered | tee -a $LOG

export PGO_CMD=kubectl
REPLY=kube
echo -n "Is this a 'kube' install or an 'ocp' install?["$REPLY"]"
read REPLY
case $REPLY in
ocp)
	export PGO_CMD=oc
	;;
esac

echo "Testing for dependencies..." | tee -a $LOG

which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency wget is missing on your system." | tee -a $LOG
	exit 1
fi
which $PGO_CMD > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency "$PGO_CMD " is missing on your system." | tee -a $LOG
	exit 1
fi

echo ""
echo "Testing "$PGO_CMD" connection..." | tee -a $LOG
echo ""
$PGO_CMD get namespaces
if [[ $? -ne 0 ]]; then
	echo $PGO_CMD  " is not connecting to your Cluster. A successful connection is required to proceed." | tee -a $LOG
	exit 1
fi

echo "Connected to cluster" | tee -a $LOG
echo ""

echo -n "Do you want to continue the installation? [Yn] "
read REPLY
case $REPLY in
n)
	echo "Aborting installation."
	exit 1
	;;
esac


export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOPATH/bin
export PGO_IMAGE_TAG=$PGO_BASEOS-$PGO_VERSION
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export PGO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/postgres-operator/server.key

echo "Setting environment variables in $HOME/.bashrc..." | tee -a $LOG

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export PATH=$PATH:$HOME/odev/bin
export PGO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/postgres-operator/server.key
alias setip='export PGO_APISERVER_URL=https://`kubectl get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443'
alias alog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c apiserver'
alias olog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c operator'
#
EOF
echo "export CCP_IMAGE_TAG="$CCP_IMAGE_TAG >> $HOME/.bashrc
echo "export CCP_IMAGE_PREFIX="$CCP_IMAGE_PREFIX >> $HOME/.bashrc
echo "export PGO_CMD="$PGO_CMD >> $HOME/.bashrc
echo "export PGO_BASEOS="$PGO_BASEOS >> $HOME/.bashrc
echo "export PGO_VERSION="$PGO_VERSION >> $HOME/.bashrc
echo "export PGO_NAMESPACE="$PGO_NAMESPACE >> $HOME/.bashrc
echo "export PGO_IMAGE_TAG="$PGO_IMAGE_TAG >> $HOME/.bashrc
echo "export PGO_IMAGE_PREFIX="$PGO_IMAGE_PREFIX >> $HOME/.bashrc

echo "Setting up installation directory..." | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/postgres-operator

echo ""
echo "Installing pgo server configuration..." | tee -a $LOG
wget --quiet https://github.com/CrunchyData/postgres-operator/releases/download/$PGO_VERSION/postgres-operator.$PGO_VERSION.tar.gz -O /tmp/postgres-operator.$PGO_VERSION.tar.gz
if [[ $? -ne 0 ]]; then
	echo "ERROR: Problem getting the pgo server configuration."
	exit 1
fi

cd $COROOT
tar xzf /tmp/postgres-operator.$PGO_VERSION.tar.gz
if [[ $? -ne 0 ]]; then
	echo "ERROR: Problem unpackaging the $PGO_VERSION release."
	exit 1
fi

echo ""
echo "Installing pgo client..." | tee -a $LOG

unameOut="$(uname -s)"
case "${unameOut}" in
    Linux*)
        cp -a pgo $GOBIN/pgo
        cp -a expenv $GOBIN/expenv
        ;;
    Darwin*)
        cp -a pgo-mac $GOBIN/pgo
        cp -a expenv-mac $GOBIN/expenv
        ;;
    CYGWIN*)
        cp -a pgo.exe $GOBIN/pgo
        cp -a expenv.exe $GOBIN/expenv
        ;;
    MINGW*)
        cp -a pgo.exe $GOBIN/pgo
        cp -a expenv.exe $GOBIN/expenv
        ;;
    *)
        machine="UNKNOWN:${unameOut}"
esac

echo "The available storage classes on your system:"
$PGO_CMD get sc
echo ""
echo -n "Enter the name of the storage class to use: "
read REPLY
export STORAGE_CLASS=$REPLY

echo ""

echo "Configuring pgo.yaml file..."
cp $COROOT/deploy/cluster-rbac.yaml $COROOT/deploy/cluster-rbac.yaml.bak
expenv -f $COROOT/deploy/cluster-rbac.yaml > /tmp/cluster-rbac.yaml
cp /tmp/cluster-rbac.yaml $COROOT/deploy/cluster-rbac.yaml
cp $COROOT/deploy/rbac.yaml  $COROOT/deploy/rbac.yaml.bak
expenv -f $COROOT/deploy/rbac.yaml > /tmp/rbac.yaml
cp /tmp/rbac.yaml $COROOT/deploy/rbac.yaml
expenv -f $COROOT/conf/postgres-operator/pgo.yaml.quickstart > $COROOT/conf/postgres-operator/pgo.yaml

echo ""
echo "Setting up pgo client authentication..." | tee -a $LOG
echo "username:password" > $HOME/.pgouser

echo "For pgo bash completion you will need to install the bash-completion package." | tee -a $LOG

cp $COROOT/examples/pgo-bash-completion $HOME/.bash_completion

echo -n "Do you want to deploy the operator? [Yn] "
read REPLY
case $REPLY in
n)
	echo "Aborting installation."
	exit 1
	;;
esac

echo "Installing the CRDs and Kube RBAC for the operator to " | tee -a $LOG
echo "the Kubernetes cluster. " | tee -a $LOG
echo "NOTE:  this step requires cluster-admin privs..." | tee -a $LOG
echo "in another terminal window, log in as a cluster-admin and " | tee -a $LOG
echo "execute the following command..." | tee -a $LOG

echo ""
echo "export PGO_CMD="$PGO_CMD | tee -a $LOG
echo "export PGO_NAMESPACE="$PGO_NAMESPACE | tee -a $LOG
echo "export COROOT="$COROOT | tee -a $LOG
echo "export PATH=$PATH:$HOME/odev/bin" | tee -a $LOG
echo "$COROOT/deploy/install-rbac.sh" | tee -a $LOG

echo ""
echo -n "Are you ready to continue the install?[Yn]"
read REPLY
case $REPLY in
n)
	echo "Aborting installation."
	exit 1
	;;
esac
echo "Deploying the operator to the Kubernetes cluster..." | tee -a $LOG
echo "PGO_IMAGE_PREFIX here is " $PGO_IMAGE_PREFIX
$COROOT/deploy/deploy.sh | tee -a $LOG

echo "Installation complete." | tee -a $LOG
echo ""

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`$PGO_CMD get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo $PGO_CMD" port-forward " $podname " 18443:8443"
echo "Run this in another terminal or in the background."

echo ""
echo "WARNING: For the postgres-operator settings to take effect, it is necessary to log out of your session and back in or reload your .bashrc file."

echo ""
echo "NOTE: In order to access the pgo CLI, place it within your PATH from its default location in $HOME/odev/bin/pgo."
