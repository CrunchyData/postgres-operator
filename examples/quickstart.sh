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

if [[ "$CO_BASEOS" != "" ]]; then
	echo "CO_BASEOS is set to " $CO_BASEOS
else
	export CO_BASEOS=centos7
	echo -n "Which crunchy images do you want to install (rhel7 or centos7)? ["$REPLY"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting CO_BASEOS="$REPLY
		export CO_BASEOS=$REPLY
	fi
fi
echo $CO_BASEOS is the baseos entered | tee -a $LOG

if [[ "$CO_VERSION" != "" ]]; then
	echo "CO_VERSION is set to " $CO_VERSION
else
	export CO_VERSION=3.4.0
	echo -n "Which Operator version do you want to install? ["$CO_VERSION"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting CO_VERSION="$REPLY
		export CO_VERSION=$REPLY
	fi
fi
echo $CO_VERSION is the version entered | tee -a $LOG

if [[ "$CO_NAMESPACE" != "" ]]; then
	echo "CO_NAMESPACE is set to " $CO_NAMESPACE
else
	export CO_NAMESPACE=demo
	echo -n "Which namespace do you want to install into? ["$CO_NAMESPACE"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting CO_NAMESPACE="$REPLY
		export CO_NAMESPACE=$REPLY
	fi
fi
echo $CO_NAMESPACE is the namespace entered | tee -a $LOG

if [[ "$CCP_IMAGE_TAG" != "" ]]; then
	echo "CCP_IMAGE_TAG is set to " $CCP_IMAGE_TAG
else
	export CCP_IMAGE_TAG=$CO_BASEOS-10.6-2.2.0
	echo -n "Which CCP image tag do you want to use? ["$CCP_IMAGE_TAG"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting CCP_IMAGE_TAG="$REPLY
		export CCP_IMAGE_TAG=$REPLY
	fi
fi
echo $CCP_IMAGE_TAG is the CCP image tag entered | tee -a $LOG

if [[ "$CO_IMAGE_PREFIX" != "" ]]; then
	echo "CO_IMAGE_PREFIX is set to " $CO_IMAGE_PREFIX
	CCP_IMAGE_PREFIX=$CO_IMAGE_PREFIX
	echo "CCP_IMAGE_PREFIX is set to " $CCP_IMAGE_PREFIX
else
	export CO_IMAGE_PREFIX=crunchydata
	export CCP_IMAGE_PREFIX=$CO_IMAGE_PREFIX
	echo -n "Which image prefix do you want to use? ["$CO_IMAGE_PREFIX"]"
	read REPLY
	if [[ "$REPLY" != "" ]]; then
		echo "setting CO_IMAGE_PREFIX="$REPLY
		echo "setting CCP_IMAGE_PREFIX="$REPLY
		export CO_IMAGE_PREFIX=$REPLY
		export CCP_IMAGE_PREFIX=$REPLY
	fi
fi
echo $CO_IMAGE_PREFIX is the CO image prefix entered | tee -a $LOG
echo $CCP_IMAGE_PREFIX is the CCP image prefix entered | tee -a $LOG

export CO_CMD=kubectl
REPLY=kube
echo -n "Is this a 'kube' install or an 'ocp' install?["$REPLY"]"
read REPLY
case $REPLY in
ocp)
	export CO_CMD=oc
	;;
esac

echo "Testing for dependencies..." | tee -a $LOG

which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency wget is missing on your system." | tee -a $LOG
	exit 1
fi
which $CO_CMD > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "The required dependency "$CO_CMD " is missing on your system." | tee -a $LOG
	exit 1
fi

echo ""
echo "Testing "$CO_CMD" connection..." | tee -a $LOG
echo ""
case $CO_CMD in
kubectl)
$CO_CMD get namespaces

export CO_CURRENT_CONTEXT=`$CO_CMD config current-context 2>/dev/null`
echo ""
echo "Using current context "$CO_CURRENT_CONTEXT | tee -a $LOG
echo ""

export CO_NAMESPACE=`$CO_CMD config view -o "jsonpath={.contexts[?(@.name==\"$CO_CURRENT_CONTEXT\")].context.namespace}"`
	;;
oc)
$CO_CMD project
export CO_NAMESPACE=`eval $CO_CMD project -q`
	;;
esac
if [[ $? -ne 0 ]]; then
	echo $CO_CMD  " is not connecting to your Cluster. A successful connection is required to proceed." | tee -a $LOG
	exit 1
fi

echo "Connected to cluster" | tee -a $LOG
echo ""

echo "The postgres-operator will be installed into the current namespace which is ["$CO_NAMESPACE"]."

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
export CO_IMAGE_TAG=$CO_BASEOS-$CO_VERSION
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/postgres-operator/server.key

echo "Setting environment variables in $HOME/.bashrc..." | tee -a $LOG

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export PATH=$PATH:$HOME/odev/bin
export CO_APISERVER_URL=https://127.0.0.1:18443
export PGO_CA_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$HOME/odev/src/github.com/crunchydata/postgres-operator/conf/postgres-operator/server.key
alias setip='export CO_APISERVER_URL=https://`kubectl get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443'
alias alog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c apiserver'
alias olog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c operator'
#
EOF
echo "export CCP_IMAGE_TAG="$CCP_IMAGE_TAG >> $HOME/.bashrc
echo "export CCP_IMAGE_PREFIX="$CCP_IMAGE_PREFIX >> $HOME/.bashrc
echo "export CO_CMD="$CO_CMD >> $HOME/.bashrc
echo "export CO_BASEOS="$CO_BASEOS >> $HOME/.bashrc
echo "export CO_VERSION="$CO_VERSION >> $HOME/.bashrc
echo "export CO_NAMESPACE="$CO_NAMESPACE >> $HOME/.bashrc
echo "export CO_IMAGE_TAG="$CO_IMAGE_TAG >> $HOME/.bashrc
echo "export CO_IMAGE_PREFIX="$CO_IMAGE_PREFIX >> $HOME/.bashrc

echo "Setting up installation directory..." | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/postgres-operator

echo ""
echo "Installing pgo server configuration..." | tee -a $LOG
wget --quiet https://github.com/CrunchyData/postgres-operator/releases/download/$CO_VERSION/postgres-operator.$CO_VERSION.tar.gz -O /tmp/postgres-operator.$CO_VERSION.tar.gz
if [[ $? -ne 0 ]]; then
	echo "ERROR: Problem getting the pgo server configuration."
	exit 1
fi

cd $COROOT
tar xzf /tmp/postgres-operator.$CO_VERSION.tar.gz
if [[ $? -ne 0 ]]; then
	echo "ERROR: Problem unpackaging the $CO_VERSION release."
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
$CO_CMD get sc
echo ""
echo -n "Enter the name of the storage class to use: "
read STORAGE_CLASS

echo ""
echo "Setting up pgo storage configuration for the selected storageclass..." | tee -a $LOG
sed -i'.bak' -e 's/Storage: nfsstorage/'"Storage: storageos"'/' $COROOT/conf/postgres-operator/pgo.yaml
sed -i'.bak' -e 's/fast/'"$STORAGE_CLASS"'/' $COROOT/conf/postgres-operator/pgo.yaml
sed -i'.bak' -e 's/COImagePrefix:  crunchydata/'"COImagePrefix:  $CO_IMAGE_PREFIX"'/' $COROOT/conf/postgres-operator/pgo.yaml
sed -i'.bak' -e 's/CCPImagePrefix:  crunchydata/'"CCPImagePrefix:  $CCP_IMAGE_PREFIX"'/' $COROOT/conf/postgres-operator/pgo.yaml
sed -i'.bak' -e 's/centos7/'"$CO_BASEOS"'/' $COROOT/conf/postgres-operator/pgo.yaml
sed -i'.bak' -e 's/demo/'"$CO_NAMESPACE"'/' $COROOT/deploy/cluster-rbac.yaml
sed -i'.bak' -e 's/demo/'"$CO_NAMESPACE"'/' $COROOT/deploy/rbac.yaml

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
echo "export CO_CMD="$CO_CMD | tee -a $LOG
echo "export CO_NAMESPACE="$CO_NAMESPACE | tee -a $LOG
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
echo "CO_IMAGE_PREFIX here is " $CO_IMAGE_PREFIX
$COROOT/deploy/deploy.sh | tee -a $LOG

echo "Installation complete." | tee -a $LOG
echo ""

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`$CO_CMD get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo $CO_CMD" port-forward " $podname " 18443:8443"
echo "Run this in another terminal or in the background."

echo ""
echo "WARNING: For the postgres-operator settings to take effect, it is necessary to log out of your session and back in or reload your .bashrc file."

echo ""
echo "NOTE: In order to access the pgo CLI, place it within your PATH from its default location in $HOME/odev/bin/pgo."
