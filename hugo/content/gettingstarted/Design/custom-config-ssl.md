---
title: "Custom SSL Configuration"
date:
draft: false
weight: 5
---

## Custom Postgres SSL Configurations

The Crunchy Data Postgres Operator can create clusters that use SSL authentication by 
utilizing custom configmaps.

#### Configuration Files for SSL Authentication

Users and administrators can specify a
custom set of Postgres configuration files to be used when creating
a new Postgres cluster. This example uses the files below- 

 * postgresql.conf
 * pg_hba.conf
 * pg_ident.conf 

along with generated security certificates, to setup a custom SSL configuration.

#### Config Files Purpose

The *postgresql.conf* file is the main Postgresql configuration file that allows
the definition of a wide variety of tuning parameters and features.

The *pg_hba.conf* file is the way Postgresql secures client access.

The *pg_ident.conf* is the ident map file and defines user name maps.

#### ConfigMap Creation

This example shows how you can configure PostgreSQL to use SSL for client authentication.

The example requires SSL certificates and keys to be created. Included in the examples directory is the script called by create.sh to create self-signed certificates (server and client) for the example: 
```
$PGOROOT/examples/ssl-creator.sh. 
```
Additionally, this script requires the certstrap utility to be installed. An install script is provided to install the correct version for the example if not already installed.

The relevant configuration files are located in the configs directory and will configure the clsuter to use SSL client authentication. These, along with the client certificate for the user 'testuser' and a server certificate for 'pgo-custom-ssl-container', will make up the necessary configuration items to be stored in the 'pgo-custom-ssl-config' configmap.

#### Example Steps

Run the script as follow:
```
cd $PGOROOT/examples/custom-config-ssl
./create.sh
```
This will generate a configmap named 'pgo-custom-ssl-config'.

Once this configmap is created, run
```
pgo create cluster customsslcluster --custom-config pgo-custom-ssl-config -n ${PGO_NAMESPACE}
```
A required step to make this example work is to define in your /etc/hosts file an entry that maps 'pgo-custom-ssl-container' to the service cluster IP address for the container created above.

For instance, if your service has an address as follows:
```
${PGO_CMD} get service -n ${PGO_NAMESPACE}
NAME                    CLUSTER-IP       EXTERNAL-IP   PORT(S)                   AGE
customsslcluster        172.30.211.108   <none>        5432/TCP
```
Then your /etc/hosts file needs an entry like this:
```
172.30.211.108 pgo-custom-ssl-container
```
For production Kubernetes and OpenShift installations, it will likely be preferred for DNS names to resolve to the PostgreSQL service name and generate server certificates using the DNS names instead of the example name pgo-custom-ssl-container.

If as a client it’s required to confirm the identity of the server, verify-full can be specified for ssl-mode in the connection string. This will check if the server and the server certificate have the same name. Additionally, the proper connection parameters must be specified in the connection string for the certificate information required to trust and verify the identity of the server (sslrootcert and sslcrl), and to authenticate the client using a certificate (sslcert and sslkey):
```
psql "postgresql://testuser@pgo-custom-ssl-container:5432/userdb?sslmode=verify-full&sslrootcert=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/ca.crt&sslcrl=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/ca.crl&sslcert=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/client.crt&sslkey=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/client.key"
```
To connect via IP, sslmode can be changed to require. This will verify the server by checking the certificate chain up to the trusted certificate authority, but will not verify that the hostname matches the certificate, as occurs with verify-full. The same connection parameters as above can be then provided for the client and server certificate information.
i
```
psql "postgresql://testuser@IP_OF_PGSQL:5432/userdb?sslmode=require&sslrootcert=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/ca.crt&sslcrl=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/ca.crl&sslcert=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/client.crt&sslkey=/home/pgo/odev/src/github.com/crunchydata/postgres-operator/examples/custom-config-ssl/certs/client.key"
```
You should see a connection that looks like the following:
```
psql (11.4)
SSL connection (protocol: TLSv1.2, cipher: ECDHE-RSA-AES256-GCM-SHA384, bits: 256, compression: off)
Type "help" for help.

userdb=>
```
#### Important Notes

Because SSL will be required for connections, certain features of the Operator will not function as expected. These include the following:
```
pgo test
pgo load
pgo apply
```
