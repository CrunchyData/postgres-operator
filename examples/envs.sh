export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN
export NAMESPACE=pgouser1,pgouser2
export CO_NAMESPACE=pgo
export CO_CMD=kubectl
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_IMAGE_PREFIX=crunchydata
export CO_BASEOS=centos7
export CO_VERSION=4.0.0
export CO_IMAGE_TAG=$CO_BASEOS-$CO_VERSION

# for the pgo CLI auth
export PGO_CA_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/postgres-operator/server.key

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

