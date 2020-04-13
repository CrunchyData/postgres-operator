#!/bin/bash

# This script should be run after the operator has been deployed
export PGO_OPERATOR_NAMESPACE=${PGO_OPERATOR_NAMESPACE:-pgo}

# Check that the pgouser-admin secret exists
if [ -z "$(kubectl get secret -n $PGO_OPERATOR_NAMESPACE pgouser-admin)" ]
then
    echo "pgouser-admin Secret not found in namespace: $PGO_OPERATOR_NAMESPACE"
    echo "Please ensure that the PostgreSQL Operator has been installed."
    echo "Exiting..."
    exit 1
fi

# Check that the pgo.tls secret exists
if [ -z "$(kubectl get secret -n $PGO_OPERATOR_NAMESPACE pgo.tls)" ]
then
    echo "pgo.tls Secret not found in namespace: $PGO_OPERATOR_NAMESPACE"
    echo "Please ensure that the PostgreSQL Operator has been installed."
    echo "Exiting..."
    exit 1
fi


# Creates the output directory for files
OUTPUT_DIR=$HOME/.pgo/$PGO_OPERATOR_NAMESPACE
mkdir -p $OUTPUT_DIR

# Use the pgouser-admin secret to generate pgouser file
kubectl get secret -n $PGO_OPERATOR_NAMESPACE pgouser-admin -o 'jsonpath={.data.username}' | base64 --decode >> $OUTPUT_DIR/pgouser
printf ":" >> $OUTPUT_DIR/pgouser
kubectl get secret -n $PGO_OPERATOR_NAMESPACE pgouser-admin -o 'jsonpath={.data.password}' | base64 --decode >> $OUTPUT_DIR/pgouser

# Use the pgo.tls secret to generate the client cert files
kubectl get secret -n $PGO_OPERATOR_NAMESPACE pgo.tls -o 'jsonpath={.data.tls\.crt}' | base64 --decode >> $OUTPUT_DIR/client.crt
kubectl get secret -n $PGO_OPERATOR_NAMESPACE pgo.tls -o 'jsonpath={.data.tls\.key}' | base64 --decode >> $OUTPUT_DIR/client.key


echo "pgo client files have been generated, please add the following to your bashrc"
echo "export PGOUSER=$OUTPUT_DIR/pgouser"
echo "export PGO_CA_CERT=$OUTPUT_DIR/client.crt"
echo "export PGO_CLIENT_CERT=$OUTPUT_DIR/client.crt"
echo "export PGO_CLIENT_KEY=$OUTPUT_DIR/client.key"