<!--
# Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
-->

Server
------

Patroni uses Python's `ssl` module to protect its REST API, `patroni`.

- `restapi.cafile` is used for client verification. It is the path to a file of
  trusted certificates concatenated in PEM format.

  See https://docs.python.org/3/library/ssl.html#ssl.SSLContext.load_verify_locations

- `restapi.certfile` is the server certificate. It is the path to a file in PEM
  format containing the certificate as well as any number of CA certificates
  needed to establish its authenticity.

  See https://docs.python.org/3/library/ssl.html#ssl.SSLContext.load_cert_chain

- `restapi.keyfile` is the server certificate's private key. This can be omitted
  if the contents are included in the certificate file.

  See https://docs.python.org/3/library/ssl.html#combined-key-and-certificate


Client
------

Patroni uses the `urllib3` module to call the REST API from `patronictl`. That,
in turn, uses Python's `ssl` module for HTTPS.

- `ctl.cacert` is used for server verification. It is the path to a file of
  trusted certificates concatenated in PEM format.

  See https://docs.python.org/3/library/ssl.html#ssl.SSLContext.load_verify_locations

- `ctl.certfile` is the client certificate. It is the path to a file in PEM
  format containing the certificate as well as any number of CA certificates
  needed to establish its authenticity.

  See https://urllib3.readthedocs.io/en/stable/reference/urllib3.connection.html
  See https://docs.python.org/3/library/ssl.html#ssl.SSLContext.load_cert_chain

- `ctl.keyfile` is the client certificate's private key. This can be omitted
  if the contents are included in the certificate file.

  See https://docs.python.org/3/library/ssl.html#combined-key-and-certificate
