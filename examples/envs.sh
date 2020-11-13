# NAMESPACE is the list of namespaces the Operator will watch
export NAMESPACE=pgouser1,pgouser2

# PGO_INSTALLATION_NAME is the unique name given to this Operator install
# this supports multi-deployments of the Operator on the same Kube cluster
export PGO_INSTALLATION_NAME=devtest

# PGO_OPERATOR_NAMESPACE is the namespace the Operator is deployed into
export PGO_OPERATOR_NAMESPACE=pgo

# PGO_CMD values are either kubectl or oc, use oc if Openshift
export PGO_CMD=kubectl

# the directory location of the Operator scripts
export PGOROOT=$HOME/postgres-operator

# the directory location of the Json Config Templates
export PGO_CONF_DIR=$PGOROOT/installers/ansible/roles/pgo-operator/files

# the version of the Operator you run is set by these vars
export PGO_IMAGE_PREFIX=registry.developers.crunchydata.com/crunchydata
export PGO_BASEOS=centos7
export PGO_VERSION=4.5.0
export PGO_IMAGE_TAG=$PGO_BASEOS-$PGO_VERSION

# for setting the pgo apiserver port, disabling TLS or not verifying TLS
# if TLS is disabled, ensure setip() function port is updated and http is used in place of https
export PGO_APISERVER_PORT=8443		# Defaults: 8443 for TLS enabled, 8080 for TLS disabled
export DISABLE_TLS=false
export TLS_NO_VERIFY=false
export TLS_CA_TRUST=""
export ADD_OS_TRUSTSTORE=false
export NOAUTH_ROUTES=""

# Disable default inclusion of OS trust in PGO clients
export EXCLUDE_OS_TRUST=false

# for the pgo CLI to authenticate with using TLS
export PGO_CA_CERT=$PGOROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_CERT=$PGOROOT/conf/postgres-operator/server.crt
export PGO_CLIENT_KEY=$PGOROOT/conf/postgres-operator/server.key

# During a Bash install determines which namespace permissions are assigned to the PostgreSQL
# Operator using a ClusterRole.  Options: `dynamic`, `readonly`, and `disabled`
export PGO_NAMESPACE_MODE=dynamic

# During a Bash install determines whether or not the PostgreSQL Operator will granted the
# permissions needed to reconcile RBAC within targeted namespaces.
export PGO_RECONCILE_RBAC=true

# common bash functions for working with the Operator
setip()
{
	export PGO_APISERVER_URL=https://`$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443
}

alog() {
$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" logs `$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c apiserver
}

olog () {
$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" logs `$PGO_CMD  -n "$PGO_OPERATOR_NAMESPACE" get pod --selector=name=postgres-operator -o jsonpath="{.items[0].metadata.name}"` -c operator
}
