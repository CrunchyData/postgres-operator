#!/bin/bash

echo "installing deps if necessary"


which git > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "git is missing on your system, a required dependency"
	exit 1
fi
which go > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "golang is missing on your system, a required dependency"
	exit 1
fi
which wget > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "wget is missing on your system, a required dependency"
	exit 1
fi
which kubectl > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "kubectl is missing on your system, a required dependency"
	exit 1
fi

echo "testing kubectl connection"
kubectl get namespaces
if [[ $? -ne 0 ]]; then
	echo "kubectl is not connecting to your Kube Cluster, required to proceed"
	exit 1
fi


echo "setting environment variables"

cat <<'EOF' >> $HOME/.bashrc

# operator env vars
export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_BASEOS=centos7
export CO_VERSION=2.4
export CO_IMAGE_TAG=$CO_BASEOS-$CO_VERSION
export CO_NAMESPACE=demo
export CO_CMD=kubectl
export CO_APISERVER_URL=https://127.0.0.1:8443
export PGO_CA_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/apiserver/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/apiserver/server.key
# 

EOF

source $HOME/.bashrc

echo "setting up directory structure"

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/

echo "installing deps if necessary"

go get github.com/blang/expenv
if [[ $? -ne 0 ]]; then
	echo "problem installing expenv dependency"
	exit 1
fi


echo "installing pgo server config"
cd $GOPATH/src/github.com/crunchydata
git clone https://github.com/CrunchyData/postgres-operator.git
if [[ $? -ne 0 ]]; then
	echo "problem getting pgo server config"
	exit 1
fi
cd $COROOT
#git checkout 2.4
git checkout master
if [[ $? -ne 0 ]]; then
	echo "problem getting 2.4 release"
	exit 1
fi

echo "installing pgo client"

cd $HOME
wget https://github.com/CrunchyData/postgres-operator/releases/download/2.4/postgres-operator.2.4.tar.gz
if [[ $? -ne 0 ]]; then
	echo "problem getting postgres-operator release"
	exit 1
fi

tar xvzf $HOME/postgres-operator.2.4.tar.gz

mv pgo $GOBIN
mv pgo-mac $GOBIN

echo "creating demo namespace"

kubectl create -f $COROOT/examples/demo-namespace.json
#if [[ $? -ne 0 ]]; then
#	echo "problem creating Kube demo namespace"
##	exit 1
#fi
Kubectl get namespaces
kubectl config view

echo "enter your Kube cluster name:"
read CLUSTERNAME
echo "enter your Kube user name:"
read USERNAME

kubectl config set-context demo --namespace=demo --cluster=$CLUSTERNAME --user=$USERNAME
kubectl config use-context demo

echo "setting up pgo storage configuration for GCE standard storageclass"
cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml

echo "deploy the operator to the Kube cluster"
cd $COROOT
#./deploy/deploy.sh

echo "setting up pgo client auth"
cp $COROOT/conf/apiserver/pgouser $HOME/.pgouser

echo "for pgo bash completion you will need to install the bash-completion package"

mv $HOME/pgo-bash-completion $HOME/.bash_completion

echo "install complete"

echo "At this point you can access the operator by using a port-forward command similar to:"
echo "kubectl port-forward postgres-operator-3590887357-7h5ht 8443:8443"
echo "do this in another terminal or run in the background"

