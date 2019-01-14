export GOPATH=$HOME/odev
export GOBIN=$GOPATH/bin
export PATH=$PATH:$GOBIN
export CO_NAMESPACE=demo
export CO_CMD=kubectl
export COROOT=$GOPATH/src/github.com/crunchydata/postgres-operator
export CO_IMAGE_PREFIX=crunchydata
export CO_BASEOS=centos7
export CO_VERSION=3.5.0
export CO_IMAGE_TAG=$CO_BASEOS-$CO_VERSION

# for the pgo CLI auth
export PGO_CA_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$COROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$COROOT/conf/postgres-operator/server.key

# useful aliases
alias setip='export CO_APISERVER_URL=https://`kubectl get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443'
alias alog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c apiserver'
alias olog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c operator'
alias slog='kubectl logs `kubectl get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c scheduler'

