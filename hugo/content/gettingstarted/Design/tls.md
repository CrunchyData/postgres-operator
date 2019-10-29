---
title: "TLS"
date:
draft: false
weight: 6
---

## TLS Configuration

Should you desire to alter the default TLS settings for the Postgres Operator, you can set the
following variables as described below.

### Server Settings

To disable TLS and make an unsecured connection on port 8080 instead of connecting securely over
the default port, 8443, set:

Bash environment variables    

```bash
DISABLE_TLS=true
PGO_APISERVER_PORT=8080		
```

Or inventory variables if using Ansible

```
pgo_disable_tls='true'
pgo_apiserver_port=8080
```

To disable TLS verifcation, set the follwing as a Bash environment variable

```bash
export TLS_NO_VERIFY=false
```

Or the following in the inventory file is using Ansible

```
pgo_tls_no_verify='false'
```

### Connection Settings

If TLS authentication has been disabled, or if the Operator's apiserver port is changed, be sure to 
update the PGO_APISERVER_URL accordingly. 

For example with an Ansible installation, 

```bash
export PGO_APISERVER_URL='https://<apiserver IP>:8443'
```

would become

```bash
export PGO_APISERVER_URL='http://<apiserver IP>:8080'
```

With a Bash installation,

```bash
setip()
{
   export PGO_APISERVER_URL=https://`$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8443
}
```

would become

```bash
setip()
{
   export PGO_APISERVER_URL=http://`$PGO_CMD -n "$PGO_OPERATOR_NAMESPACE" get service postgres-operator -o=jsonpath="{.spec.clusterIP}"`:8080
}
```

### Client Settings

Finally, if TLS has been disabled for the Operator's apiserver, the PGO client connection must be set to match
the given settings.

Two options are available, either the Bash environment variable

```bash
DISABLE_TLS=true
```

must be configured, or the --disable-tls flag must be included when using the client, i.e.

```bash
pgo version --disable-tls
```
