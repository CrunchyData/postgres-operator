<!--
 Copyright 2021 - 2022 Crunchy Data Solutions, Inc.
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at

 http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

Server
------

pgBackRest uses OpenSSL to protect connections between machines. The [TLS server](tls-server.md)
listens on a TCP port, encrypts connections with its server certificate, and
verifies client certificates against a certificate authority.

- `tls-server-ca-file` is used for client verification. It is the path to a file
  of trusted certificates concatenated in PEM format. When this is set, clients
  are also authorized according to `tls-server-auth`.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_load_verify_locations.html

- `tls-server-cert-file` is the server certificate. It is the path to a file in
  PEM format containing the certificate as well as any number of CA certificates
  needed to establish its authenticity.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_use_certificate_chain_file.html

- `tls-server-key-file` is the server certificate's private key. It is the path
  to a file in PEM format.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_use_PrivateKey_file.html


Clients
-------

pgBackRest uses OpenSSL to protect connections it makes to PostgreSQL instances
and repository hosts. It presents a client certificate that is verified by the
server and must contain a common name (CN) that is authorized according to `tls-server-auth`.

- `pg-host-ca-file` is used for server verification when connecting to
  pgBackRest on a PostgreSQL instance. It is the path to a file of trusted
  certificates concatenated in PEM format.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_load_verify_locations.html

- `pg-host-cert-file` is the client certificate to present when connecting to
  pgBackRest on a PostgreSQL instance. It is the path to a file in PEM format
  containing the certificate as well as any number of CA certificates needed to
  establish its authenticity.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_use_certificate_chain_file.html

- `pg-host-key-file` is the client certificate's private key. It is the path
  to a file in PEM format.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_use_PrivateKey_file.html

- `repo-host-ca-file` is used for server verification when connecting to
  pgBackRest on a repository host. It is the path to a file of trusted
  certificates concatenated in PEM format.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_load_verify_locations.html

- `repo-host-cert-file` is the client certificate to present when connecting to
  pgBackRest on a repository host. It is the path to a file in PEM format
  containing the certificate as well as any number of CA certificates needed to
  establish its authenticity.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_use_certificate_chain_file.html

- `repo-host-key-file` is the client certificate's private key. It is the path
  to a file in PEM format.

  See https://www.openssl.org/docs/man1.1.1/man3/SSL_CTX_use_PrivateKey_file.html

