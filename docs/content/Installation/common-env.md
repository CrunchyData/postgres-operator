---
title: "Operator Environment"
date:
draft: false
weight: 50
---

For various scripts used by the Operator, the `expenv` utility is required as are certain environment variables.

Download the `expenv` utility from its [Github Releases page](https://github.com/blang/expenv/releases), and place it into your PATH (e.g. $HOME/odev/bin).

The following environment variables are heavily used in the Bash installation procedures and may be used in Operator helper scripts.

Variable | Ansible Inventory | Example | Description
-------- | ----------------- | ------- | -----------
`DISABLE_EVENTING` | pgo_disable_eventing | false | Disable Operator eventing subsystem
`DISABLE_TLS` | pgo_disable_tls | false | Disable TLS for Operator
`GOPATH` |  | $HOME/odev | Golang project directory
`GOBIN` |  | $GOPATH/bin | Golang binary target directory
`NAMESPACE` | namespace | pgouser1 | Namespaces monitored by Operator
`PGOROOT` |  | $GOPATH/src/github.com/crunchydata/postgres-operator | Operator repository location
`PGO_APISERVER_PORT` | pgo_apiserver_port | 8443 | HTTP(S) port for Operator API server
`PGO_BASEOS` |  | centos7 | Base OS for container images
`PGO_CA_CERT` |  | $PGOROOT/conf/postgres-operator/server.crt | Server certificate and CA trust
`PGO_CMD` |  | kubectl | Cluster management tool executable
`PGO_CLIENT_CERT` |  | $PGOROOT/conf/postgres-operator/server.crt | TLS Client certificate
`PGO_CLIENT_KEY` |  | $PGOROOT/conf/postgres-operator/server.crt | TLS Client certificate private key
`PGO_IMAGE_PREFIX` | pgo_image_prefix | crunchydata | Container image prefix
`PGO_IMAGE_TAG` | pgo_image_tag | $PGO_BASEOS-$PGO_VERSION | OS/Version tagging info for images
`PGO_INSTALLATION_NAME` | pgo_installation_name | devtest | Unique name given to Operator installation
`PGO_OPERATOR_NAMESPACE` | pgo_operator_namespace | pgo | Kubernetes namespace for the operator
`PGO_VERSION` |  | 4.3.0 | Operator version 
`TLS_NO_VERIFY` | pgo_tls_no_verify | false | Disable certificate verification (e.g. strict hostname checking)
`TLS_CA_TRUST` | pgo_tls_ca_store | /var/pki/my_cas.crt | PEM-encoded list of trusted CA certificates
`ADD_OS_TRUSTSTORE` | pgo_add_os_ca_store | false | Adds OS root trust collection to apiserver
`NOAUTH_ROUTES` | pgo_noauth_routes | "/health" | Disable mTLS and HTTP BasicAuth for listed routes
`EXCLUDE_OS_TRUST` |  | false* | Excludes OS root trust from pgo client (defaults to true for windows clients)

{{% notice tip %}}
`examples/envs.sh` contains the above variable definitions as well
{{% /notice %}}
