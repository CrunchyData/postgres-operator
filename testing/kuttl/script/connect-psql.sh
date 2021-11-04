#!/bin/bash
CLUSTER=${CLUSTER:-default}
PG_CLUSTER_USER_SECRET_NAME=${PG_CLUSTER_USER_SECRET_NAME:-$CLUSTER-pguser-$CLUSTER}

PGPASSWORD=$(kubectl get secrets -n $NAMESPACE "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.password | base64decode}}') \
PGUSER=$(kubectl get secrets -n $NAMESPACE "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.user | base64decode}}') \
PGDATABASE=$(kubectl get secrets -n $NAMESPACE "${PG_CLUSTER_USER_SECRET_NAME}" -o go-template='{{.data.dbname | base64decode}}') \
psql -h localhost -c "select version();"
