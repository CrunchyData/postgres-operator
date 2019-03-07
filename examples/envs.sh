export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN

# NAMESPACE is the list of namespaces the Operator will watch
export NAMESPACE=pgouser1,pgouser2

# CO_NAMESPACE is the namespace the Operator is deployed into
export CO_NAMESPACE=pgo

# CO_CMD values are either kubectl or oc, use oc if Openshift
export CO_CMD=kubectl

# the directory location of the Operator scripts
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator

# the version of the Operator you run is set by these vars
export CO_IMAGE_PREFIX=crunchydata
export CO_BASEOS=centos7
export CO_VERSION=4.0.0-rc1
export CO_IMAGE_TAG=$CO_BASEOS-$CO_VERSION

# for the pgo CLI to authenticate with using TLS
export PGO_CA_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/postgres-operator/server.key

# common bash functions for working with the Operator
setip() 
{ 
	export CO_APISERVER_URL=https://`$CO_CMD -n "$CO_NAMESPACE" get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443 
}

alog() {
kubectl  -n "$CO_NAMESPACE" logs `$CO_CMD  -n "$CO_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c apiserver
}

olog () {
kubectl  -n "$CO_NAMESPACE" logs `$CO_CMD  -n "$CO_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c operator
}

slog () {
kubectl  -n "$CO_NAMESPACE" logs `$CO_CMD  -n "$CO_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c scheduler
}

