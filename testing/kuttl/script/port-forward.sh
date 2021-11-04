#!/bin/bash
CLUSTER=${CLUSTER:-default}
export PG_CLUSTER_PRIMARY_POD=$(kubectl get pod -n $NAMESPACE -o name -l postgres-operator.crunchydata.com/cluster=$CLUSTER,postgres-operator.crunchydata.com/role=master)
kubectl -n $NAMESPACE port-forward "${PG_CLUSTER_PRIMARY_POD}" 5432:5432