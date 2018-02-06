#!/bin/bash

LOG="pgo-installer.log"

echo "installing deps if necessary" | tee -a $LOG


which git > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "git is missing on your system, a required dependency" | tee -a $LOG
	exit 1
fi
which go > /dev/null 2> /dev/null
if [[ $? -ne 0 ]]; then
	echo "golang is missing on your system, a required dependency" | tee -a $LOG
	exit 1
fi
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

echo "testing kubectl connection" | tee -a $LOG
kubectl get namespaces | tee -a $LOG
if [[ $? -ne 0 ]]; then
	echo "kubectl is not connecting to your Kube Cluster, required to proceed" | tee -a $LOG
	exit 1
fi


echo "setting environment variables" | tee -a $LOG

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

echo "setting up directory structure" | tee -a $LOG

mkdir -p $HOME/odev/src $HOME/odev/bin $HOME/odev/pkg
mkdir -p $GOPATH/src/github.com/crunchydata/

echo "installing deps if necessary" | tee -a $LOG

go get github.com/blang/expenv
if [[ $? -ne 0 ]]; then
	echo "problem installing expenv dependency" | tee -a $LOG
	exit 1
fi


echo "installing pgo server config" | tee -a $LOG
cd $GOPATH/src/github.com/crunchydata
git clone --quiet https://github.com/CrunchyData/postgres-operator.git 
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

echo "installing pgo client" | tee -a $LOG

cd $HOME
wget --quiet https://github.com/CrunchyData/postgres-operator/releases/download/2.4/postgres-operator.2.4.tar.gz
if [[ $? -ne 0 ]]; then
	echo "problem getting postgres-operator release" | tee -a $LOG
	exit 1
fi

tar xzf $HOME/postgres-operator.2.4.tar.gz

mv pgo $GOBIN
mv pgo-mac $GOBIN

echo -n "do you want to create the demo namespace? [yes no] "
read REPLY
if [[ "$REPLY" == "yes" ]]; then
	echo "creating demo namespace" | tee -a $LOG

	kubectl create -f $COROOT/examples/demo-namespace.json
	#if [[ $? -ne 0 ]]; then
	#	echo "problem creating Kube demo namespace"
	##	exit 1
	#fi
	Kubectl get namespaces
	kubectl config view

	echo "enter your Kube cluster name: "
	read CLUSTERNAME
	echo "enter your Kube user name: "
	read USERNAME

	kubectl config set-context demo --namespace=demo --cluster=$CLUSTERNAME --user=$USERNAME
	kubectl config use-context demo
fi

echo -n "do you want to deploy the operator? [yes no] "
read REPLY
if [[ "$REPLY" == "yes" ]]; then
	echo "setting up pgo storage configuration for GCE standard storageclass" | tee -a $LOG
	cp $COROOT/examples/pgo.yaml.storageclass $COROOT/conf/apiserver/pgo.yaml

	echo "deploy the operator to the Kube cluster" | tee -a $LOG
	cd $COROOT
	#./deploy/deploy.sh
fi

echo "setting up pgo client auth" | tee -a $LOG
tail --lines=1 $COROOT/conf/apiserver/pgouser > $HOME/.pgouser

echo "for pgo bash completion you will need to install the bash-completion package" | tee -a $LOG

mv $HOME/pgo-bash-completion $HOME/.bash_completion

echo "install complete" | tee -a $LOG

echo "At this point you can access the operator by using a port-forward command similar to:"
podname=`kubectl get pod --selector=name=postgres-operator -o jsonpath={..metadata.name}`
echo "kubectl port-forward " $podname " 8443:8443"
echo "do this in another terminal or run in the background"

