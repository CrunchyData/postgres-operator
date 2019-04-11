#!/bin/bash

CERTSTRAP_VERSION=1.1.1

# manually install certstrap into $GOBIN for running the SSL examples
wget -O $PGOROOT/certstrap https://github.com/square/certstrap/releases/download/v${CERTSTRAP_VERSION}/certstrap-v${CERTSTRAP_VERSION}-linux-amd64 && \
    mv $PGOROOT/certstrap $GOBIN && \
    chmod +x $GOBIN/certstrap

