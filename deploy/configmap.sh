#!/bin/bash
set -e

DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

CERTS_DIR=cp4s_certs

mkdir -p $CERTS_DIR

#create root csr
openssl req -new -nodes -text -out $CERTS_DIR/root.csr  -keyout $CERTS_DIR/root.key -subj "/CN=isc-cases-postgres"

#self sign root cert
openssl x509 -req -in $CERTS_DIR/root.csr -text -days 3650 -extfile /usr/local/etc/openssl/openssl.cnf -extensions v3_ca -signkey $CERTS_DIR/root.key -out $CERTS_DIR/root.crt

#create server csr
openssl req -new -nodes -text -out $CERTS_DIR/server.csr -keyout $CERTS_DIR/server.key -subj "/CN=isc-cases-postgres"

#sign server csr with root csr
openssl x509 -req -in $CERTS_DIR/server.csr -text -days 365 -CA $CERTS_DIR/root.crt -CAkey $CERTS_DIR/root.key -CAcreateserial -out $CERTS_DIR/server.crt

# make crt available to cases application to establish trust
cat <<EOF | oc apply -f -
apiVersion: v1
kind: ConfigMap
metadata:
  name: isc-cases-postgres-ca-cert
binaryData:
  postgres_cert: `cat $CERTS_DIR/root.crt | base64`
EOF

# create configmap required by postgres operator
oc create configmap isc-cases-pgcluster-configmap \
    --from-file=$CERTS_DIR/server.crt \
    --from-file=$CERTS_DIR/server.key \
    --from-file=$DIR/../conf/postgres/pg_hba.conf \
    --from-file=$DIR/../conf/postgres/postgresql.conf
