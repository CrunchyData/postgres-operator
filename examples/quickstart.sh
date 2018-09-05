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
if [[ "$CO_VERSION" != "" ]]; then
	echo "CO_VERSION is set to " $CO_VERSION
else
	export CO_VERSION=3.2.0
fi
if [[ "$CO_NAMESPACE" != "" ]]; then
	echo "CO_NAMESPACE is set to " $CO_NAMESPACE
fi

echo -n "Which Operator version do you want to install? ["$CO_VERSION"]"
read REPLY
if [[ "$REPLY" != "" ]]; then
	echo "setting CO_VERSION="$REPLY
	export CO_VERSION=$REPLY
fi
echo $CO_VERSION is the version entered | tee -a $LOG

echo -n "Is this a 'kube' install or an 'ocp' install?[kube]"
read REPLY
case $REPLY in
kube)
	echo "user has selected a kube install" | tee -a $LOG
	export CO_CMD=kubectl
	;;
ocp)
	echo "user has selected an ocp install" | tee -a $LOG
	export CO_CMD=oc
	;;
*)
	echo "user has selected a kube install" | tee -a $LOG
	export CO_CMD=kubectl
	;;
esac

echo -n "use centos7 or rhel7 based images?[centos7], NOTE:  rhel images available only to crunchy customers)"
read REPLY
case $REPLY in
centos7)
	echo "user has selected centos images" | tee -a $LOG
	export CO_BASEOS=centos7
	;;
rhel7)
	echo "user has selected rhel images" | tee -a $LOG
	export CO_BASEOS=rhel7
	;;
*)
	echo "user has selected centos images" | tee -a $LOG
	export CO_BASEOS=centos7
	;;
esac

echo -n "enter operator image prefix ["$CO_IMAGE_PREFIX"]"
read REPLY
if [[ "$REPLY" != "" ]]; then
	echo "setting CO_IMAGE_PREFIX="$REPLY
	export CO_IMAGE_PREFIX=$REPLY
else
	echo "setting CO_IMAGE_PREFIX to " $CO_IMAGE_PREFIX
fi
echo "user has entered "$CO_IMAGE_PREFIX " for the operator image prefix"| tee -a $LOG

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
export CO_NAMESPACE=`$CO_CMD config view | grep namespace:| cut -f2 -d':' | cut -f2 -d' '`
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

mv pgo $GOBIN
mv pgo-mac $GOBIN
mv pgo.exe $GOBIN
mv expenv.exe $GOBIN
mv expenv-mac $GOBIN
mv expenv $GOBIN

echo "The available storage classes on your system:"
$CO_CMD get sc
echo ""
echo -n "Enter the name of the storage class to use: "
read STORAGE_CLASS

echo ""
echo "Setting up pgo storage configuration for the selected storageclass..." | tee -a $LOG
cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/standard/'"$STORAGE_CLASS"'/' $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/crunchydata/'"$CO_IMAGE_PREFIX"'/' $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/centos7/'"$CO_BASEOS"'/' $COROOT/conf/apiserver/pgo.yaml
sed --in-place=.bak 's/demo/'"$CO_NAMESPACE"'/' $COROOT/deploy/cluster-rbac.yaml
sed --in-place=.bak 's/demo/'"$CO_NAMESPACE"'/' $COROOT/deploy/rbac.yaml

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
