# Copyright 2024 - 2025 Crunchy Data Solutions, Inc.
#
# SPDX-License-Identifier: Apache-2.0
#
# We set the digest method to `hashlib.sha1` directly to
# make pgAdmin compatible with FIPS and Red Hat Python.
# Without this config, the pgAdmin dependency `pallets/itsdangerous`
# sets the digest method to a lazy loader (`_lazy_sha1`),
# which indirectly loads `hashlib.sha1`.
# Rather than rely on an indirect load of `hashlib.sha1`,
# we rely on a direct setting of the same hashing function
# because Red Hat Python in a FIPS environment will reject
# any hashing function that is not recognized.
# --https://bugzilla.redhat.com/show_bug.cgi?id=2064343#c4

from itsdangerous import signer
import hashlib

signer.Signer.default_digest_method = staticmethod(hashlib.sha1)
