// Copyright 2021 - 2024 Crunchy Data Solutions, Inc.
//
// SPDX-License-Identifier: Apache-2.0

// Package pki provides types and functions to support the public key
// infrastructure of the Postgres Operator. It enforces a two layer system
// of certificate authorities and certificates.
//
// NewRootCertificateAuthority() creates a new root CA.
// GenerateLeafCertificate() creates a new leaf certificate.
//
// Certificate and PrivateKey are primitives that can be marshaled.
package pki
