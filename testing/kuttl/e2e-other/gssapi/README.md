# GSSAPI Authentication

This test verifies that it is possible to properly configure PostgreSQL for GSSAPI
authentication.  This is done by configuring a PostgresCluster for GSSAPI authentication,
and then utilizing a Kerberos ticket that has been issued by a Kerberos KDC server to log into
PostgreSQL.

## Assumptions

- A Kerberos Key Distribution Center (KDC) Pod named `krb5-kdc-0` is deployed inside of a `krb5`
namespace within the Kubernetes cluster
- The KDC server (`krb5-kdc-0`) contains a `/krb5-conf/krb5.sh` script that can be run as part
of the test to create the Kerberos principals, keytab secret and client configuration needed to
successfully run the test
