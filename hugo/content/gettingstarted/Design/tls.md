---
title: "TLS"
date:
draft: false
weight: 6
---

## TLS Configuration

Should you desire to alter the default TLS settings for the Postgres Operator, you can set the
following variables in bash:

To disable TLS and make an unsecured connection on port 8080 instead of connecting securely over
the default port, 8443, set:

Bash environment variables    

    DISABLE_TLS=true
    PGO_APISERVER_PORT=8080		

Or inventory variables if using Ansible

    pgo_disable_tls='true'
    pgo_apiserver_port=8080

To disable TLS verifcation, set the follwing as a Bash environment variable

    export TLS_NO_VERIFY=false

Or the following in the inventory file is using Ansible

    pgo_tls_no_verify='false'