### Password Change Test with Kuttl

This Kuttl routine runs through the following steps:

#### Create cluster and test connection

- 00: Creates the cluster and verifies that it exists and is ready for connection
- 01: Connects to the cluster with the PGO-generated password (both with env vars and with the URI)

#### Default user connection tests

- 02: Change the password (using Kuttl's update object method on the secret's `data` field) and verify that the password changes by asserting that the `verifier` field is not blank (using KUTTL's `errors` method, which makes sure that a state is _not_ met by a certain time)
- 03: Connects to the cluster with the user-defined password (both with env vars and with the URI)
- 04: Change the password and verifier (using Kuttl's update object method on the secret's `stringData` field) and verify that the password changes by asserting that the `uri` field is not blank (using KUTTL's `errors` method, which makes sure that a state is _not_ met by a certain time)
- 05: Connects to the cluster with the second user-defined password (both with env vars and with the URI)

#### Create custom user and test connection

- 06: Updates the postgrescluster spec with a custom user and password
- 07: Connects to the cluster with the PGO-generated password (both with env vars and with the URI) for the custom user

#### Custom user connection tests

- 08: Change the custom user's password (using Kuttl's update object method on the secret's `data` field) and verify that the password changes by asserting that the `verifier` field is not blank (using KUTTL's `errors` method, which makes sure that a state is _not_ met by a certain time)
- 09: Connects to the cluster with the user-defined password (both with env vars and with the URI) for the custom user
- 10: Change the custom user's password and verifier (using Kuttl's update object method on the secret's `stringData` field) and verify that the password changes by asserting that the `uri` field is not blank (using KUTTL's `errors` method, which makes sure that a state is _not_ met by a certain time)
- 11: Connects to the cluster with the second user-defined password (both with env vars and with the URI) for the custom user
