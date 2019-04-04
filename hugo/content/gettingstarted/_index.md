---
title: "Getting Started"
date:
draft: false
weight: 2
---

Operator users typically will just need to install the *pgo* client
in order to work with an already deployed Postgres Operator.

For installing the Operator on the Kubernetes server, see
the [Installation Guide](https://crunchydata.github.io/postgres-operator/stable/installation/).

### pgo Client Installation

The *pgo* binary is pre-compiled and available to download from the projects
github repository [Releases](https://github.com/crunchydata/postgres-operator)  page.

Add the *pgo* binary to your PATH and make it executable.

Next, create a *.pgouser* file in your $HOME directory or 
set the PGOUSER environment variable to point to a *.pgouser*
file location:

    export PGOUSER=/somepath/.pgouser

Alternatively, if you create a *.pgouser* file in your $HOME, the *pgo*
client will find the file there.

Set the name of the Kubernetes namespace that you want to 
access, on a Linux host you would enter:

    export PGO_NAMESPACE=pgouser1

Set the URL of the Operator REST API, in this example the Operator is running 
on a host with IP address 192.168.0.120, see your administrator for
the correct IP address, on a Linux host you would enter:

    export PGO_APISERVER_URL=https://192.168.0.120:8443

Next, add the TLS keys required for the *pgo* client to connect to the
Operator REST API, see your administrator for access to these keys, on a Linux host you would enter:

    export PGO_CA_CERT=/somepath/someserver.crt
    export PGO_CLIENT_CERT=/somepath/someserver.crt
    export PGO_CLIENT_KEY=/somepath/someserver.key

Lastly, test out the connection:

    pgo version
